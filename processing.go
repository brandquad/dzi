package dzi

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alitto/pond"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/google/uuid"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

const DefaultFolderPerm = 0777

type Config struct {
	S3Host             string
	S3Key              string
	S3Secret           string
	S3Bucket           string
	TileSize           string
	Overlap            string
	Resolution         int
	CoverHeight        string
	ICCProfileFilepath string
	SplitChannels      bool
	DebugMode          bool
	CopyChannelsToS3   bool
	DefaultDPI         float64
	MaxSizePixels      float64
	MaxCpuCount        int
	ExtractText        bool
}

func prepareTopFolders(folders ...string) error {
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
	var _tmp string
	if c.DebugMode {
		log.Println("DEBUG MODE ON")

		_tmp = "_tmp"
		if _, err := os.ReadDir(_tmp); err == nil {
			err = os.RemoveAll(_tmp)
			if err != nil {
				return nil, err
			}
		}
	} else {
		_tmp = os.TempDir()
	}

	tmp := path.Join(_tmp, strconv.Itoa(assetId))
	leads := path.Join(tmp, "leads")
	dzi := path.Join(tmp, "dzi")
	dziBw := path.Join(tmp, "dzi_bw")
	channels := path.Join(tmp, "channels")
	channelsBw := path.Join(tmp, "channels_bw")
	covers := path.Join(tmp, "covers")

	if err := prepareTopFolders(tmp, leads, dzi, dziBw, channels, channelsBw, covers); err != nil {
		return nil, err
	}

	ext := path.Ext(filename)
	basename := strings.TrimSuffix(filename, ext)
	basename = uuid.New().String()
	ext = strings.TrimPrefix(ext, ".")

	log.Println("Max Resolution:", c.Resolution)
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

	probe, err := vips.LoadImageFromFile(originalFilepath, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		if probe != nil {
			probe.Close()
		}
	}()

	Loader := probe.OriginalFormat()

	var pages []*pageInfo
	if Loader == vips.ImageTypePDF {
		log.Println("Processing as PDF file")
		pages, err = extractPDF(originalFilepath, basename, channels, c)
		if err != nil {
			return nil, err
		}
		//log.Println("Done.")
	} else {
		log.Println("Processing as Image file")
		pages, err = extractImage(originalFilepath, basename, channels, c.ICCProfileFilepath, c.SplitChannels)
		if err != nil {
			return nil, err
		}
		//log.Println("Done.")
	}

	//log.Println("Colorize channels")

	if err = colorize(pages, channels, channelsBw, leads, covers, c); err != nil {
		return nil, err
	}

	panicHandler := func(p interface{}) {
		fmt.Printf("Task panicked: %v", p)
	}
	pool := pond.New(c.MaxCpuCount, 1000, pond.MinWorkers(c.MaxCpuCount), pond.PanicHandler(panicHandler))
	if err = makeDZI(pool, pages, channels, dzi, c); err != nil {
		return nil, err
	}
	if err = makeDZI(pool, pages, channelsBw, dziBw, c); err != nil {
		return nil, err
	}

	pool.StopAndWait()
	if pool.FailedTasks() > 0 {
		return nil, errors.New("error on make dzi")
	}

	makeCovers(dzi, c)

	if !c.CopyChannelsToS3 {
		log.Println("Remove Color channels folder")
		if err = os.RemoveAll(channels); err != nil {
			return nil, err
		}
		log.Println("Remove B-W channels folder")
		if err = os.RemoveAll(channelsBw); err != nil {
			return nil, err
		}
	}

	manifest, err := makeManifest(pages, assetId, c, url, basename, filename)
	if err != nil {
		return nil, err
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
		if !c.DebugMode {
			if err := baseFile.Close(); err != nil {
				log.Printf("Error closing file: %v", err)
			}
			if err := os.Remove(originalFilepath); err != nil {
				log.Printf("Error removing file: %v", originalFilepath)
			}
			if err := os.RemoveAll(tmp); err != nil {
				log.Printf("Error removing directory: %v", tmp)
			}
		}
	}()

	return manifest, nil
}
