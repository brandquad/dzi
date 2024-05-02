package dzi

import (
	"encoding/json"
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const DefaultFolderPerm = 0777

type Config struct {
	S3Host             string
	S3Key              string
	S3Secret           string
	S3Bucket           string
	TileSize           string
	Overlap            string
	Resolution         string
	CoverWidth         string
	ICCProfileFilepath string
	DebugMode          bool
}

func prepareFolders(folders ...string) error {
	for _, folder := range folders {
		err := os.MkdirAll(folder, DefaultFolderPerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func Processing(url string, assetId int, c Config) (*Manifest, error) {
	filename := path.Base(url)
	//_tmp := os.TempDir()
	_tmp := "_tmp"

	if _, err := os.ReadDir(_tmp); err == nil {
		err = os.RemoveAll(_tmp)
		if err != nil {
			return nil, err
		}
	}

	tmp := path.Join(_tmp, strconv.Itoa(assetId))

	leads := path.Join(tmp, "leads")
	dzi := path.Join(tmp, "dzi")
	dziBw := path.Join(tmp, "dzi_bw")
	channels := path.Join(tmp, "channels")
	channelsBw := path.Join(tmp, "channels_bw")
	covers := path.Join(tmp, "covers")

	if err := prepareFolders(tmp, leads, dzi, dziBw, channels, channelsBw, covers); err != nil {
		return nil, err
	}

	ext := path.Ext(filename)
	basename := strings.TrimSuffix(filename, ext)
	ext = strings.TrimPrefix(ext, ".")

	log.Println("Resolution:", c.Resolution)
	log.Println("URL:", url)
	log.Println("AssetId:", assetId)
	log.Println("Filename:", filename)
	log.Println("Basename:", basename, ext)
	log.Println("Tmp:", tmp)

	baseFile, err := downloadFileTemporary(url)
	if err != nil {
		return nil, err
	}
	originalFilepath := baseFile.Name()

	var Colormode string

	//log.Println(sss, err)

	//probe, err := vipsProbe(originalFilepath, "vips-loader")
	probe, err := vips.LoadImageFromFile(originalFilepath, nil)
	if err != nil {
		return nil, err
	}

	Loader := probe.OriginalFormat()
	probe.Close()
	Colormode = "CMYK"

	var info *entryInfo
	if Loader == vips.ImageTypePDF {
		log.Println("Processing as PDF file")
		resolution, err := strconv.Atoi(c.Resolution)
		if err != nil {
			return nil, err
		}
		info, err = extractPDF(originalFilepath, basename, channels, resolution)
		if err != nil {
			return nil, err
		}
		log.Println("Done.")
	} else {
		log.Println("Processing as Image file")
		info, err = extractImage(originalFilepath, basename, channels, c.ICCProfileFilepath)
		if err != nil {
			return nil, err
		}
		log.Println("Done.")
	}

	log.Println("Colorize channels")

	coverWidth, _ := strconv.Atoi(c.CoverWidth)
	if err = colorize(info, channels, channelsBw, leads, covers, c.ICCProfileFilepath, coverWidth); err != nil {
		return nil, err
	}

	log.Println("Make DZI - colors")
	if err = makeDZI(leads, dzi, c); err != nil {
		return nil, err
	}

	log.Println("Make DZI - b-w")
	if err = makeDZI(channelsBw, dziBw, c); err != nil {
		return nil, err
	}

	log.Println("Make manifest.json")
	var pSize = DziSize{
		Width:      fmt.Sprintf("%d", int(info.Width)),
		Height:     fmt.Sprintf("%d", int(info.Height)),
		CoverWidth: c.CoverWidth,
		Units:      info.Unit,
		Dpi:        c.Resolution,
		Overlap:    c.Overlap,
		TileSize:   c.TileSize,
	}

	var channelsArr = make([]string, len(info.Swatches))
	for idx, s := range info.Swatches {
		channelsArr[idx] = s.Name
	}
	var manifest = &Manifest{
		ID:        strconv.Itoa(assetId),
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Source:    url,
		Filename:  filename,
		Basename:  basename,
		Mode:      Colormode,
		Size:      pSize,
		Channels:  channelsArr,
	}
	buff, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	if err = os.WriteFile(path.Join(tmp, "manifest.json"), buff, 0777); err != nil {
		return nil, err
	}

	if err = syncToS3(assetId, tmp, c); err != nil {
		return nil, err
	}

	defer func() {
		//if !c.DebugMode {
		//	if err := baseFile.Close(); err != nil {
		//		log.Printf("Error closing file: %v", err)
		//	}
		//	if err := os.Remove(originalFilepath); err != nil {
		//		log.Printf("Error removing file: %v", originalFilepath)
		//	}
		//	if err := os.RemoveAll(tmp); err != nil {
		//		log.Printf("Error removing directory: %v", tmp)
		//	}
		//}
	}()

	return manifest, nil
}

func vipsProbe(path string, params ...string) (map[string]string, error) {
	result := make(map[string]string)

	output, err := execCmd("vipsheader", "-a", path)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(output), "\n") {
		for _, p := range params {
			if f, ok := extractField(line, fmt.Sprintf("%s:", p)); ok {
				result[p] = f
			}
		}
	}

	for _, p := range params {
		if _, ok := result[p]; !ok {
			return nil, fmt.Errorf("param %s not found", p)
		}
	}

	return result, nil
}

func makeDZI(income string, outcome string, c Config) error {
	entry, err := os.ReadDir(income)
	if err != nil {
		return err
	}

	for _, f := range entry {
		if f.IsDir() {
			continue
		}

		fpath := path.Join(income, f.Name())
		fext := path.Ext(f.Name())
		fbasename := strings.TrimSuffix(f.Name(), fext)
		dziPath := path.Join(outcome, fbasename)

		if _, err = execCmd("vips", "dzsave",
			fpath,
			dziPath,
			"--strip",
			"--suffix",
			".webp",
			fmt.Sprintf("--tile-size=%s", c.TileSize),
			fmt.Sprintf("--overlap=%s", c.Overlap)); err != nil {
			return err
		}
	}
	return nil
}
