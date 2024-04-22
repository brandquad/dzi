package dzi_processing

import (
	"encoding/json"
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/ssoroka/slice"
	"log"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const DefaultFolderPerm = 0777

var rgbColormodes = []string{"srgb", "rgb", "rgb16"}
var allowedColormodes = append([]string{"cmyk"}, rgbColormodes...)

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

func Processing(url string, assetId int, c Config) (*Manifest, error) {
	filename := path.Base(url)
	_tmp := os.TempDir()
	//_tmp := "./_tmp"
	if err := os.MkdirAll(_tmp, DefaultFolderPerm); err != nil {
		return nil, err
	}

	tmp := path.Join(_tmp, strconv.Itoa(assetId))

	ext := path.Ext(filename)
	basename := strings.TrimSuffix(filename, ext)
	ext = strings.TrimPrefix(ext, ".")

	log.Println("Resolution:", c.Resolution)
	log.Println("URL:", url)
	log.Println("AssetId:", assetId)
	log.Println("Filename:", filename)
	log.Println("Tmp:", tmp)
	log.Println("Basename:", basename, ext)

	if err := os.MkdirAll(tmp, DefaultFolderPerm); err != nil {
		return nil, err
	}

	leads := path.Join(tmp, "leads")
	if err := os.MkdirAll(leads, DefaultFolderPerm); err != nil {
		return nil, err
	}

	leads1000 := path.Join(tmp, "leads1000")
	if err := os.MkdirAll(leads1000, DefaultFolderPerm); err != nil {
		return nil, err
	}

	dzi := path.Join(tmp, "dzi")
	if err := os.MkdirAll(dzi, DefaultFolderPerm); err != nil {
		return nil, err
	}

	dzi_bw := path.Join(tmp, "dzi_bw")
	if err := os.MkdirAll(dzi_bw, DefaultFolderPerm); err != nil {
		return nil, err
	}

	channels := path.Join(tmp, "channels")
	if err := os.MkdirAll(channels, DefaultFolderPerm); err != nil {
		return nil, err
	}

	channels_bw := path.Join(tmp, "channels_bw")
	if err := os.MkdirAll(channels_bw, DefaultFolderPerm); err != nil {
		return nil, err
	}

	covers := path.Join(tmp, "covers")
	if err := os.MkdirAll(covers, DefaultFolderPerm); err != nil {
		return nil, err
	}

	baseFile, err := downloadFileTemporary(url)
	if err != nil {
		return nil, err
	}
	originalFilepath := baseFile.Name()

	var Units, Width, Height, Colormode, Loader, rgbCopyFilepath string
	var Bands int

	probe, err := vipsProbe(originalFilepath, "vips-loader", "width", "height", "interpretation")
	if err != nil {
		return nil, err
	}

	Loader = probe["vips-loader"]
	Width = probe["width"]
	Height = probe["height"]
	Colormode = probe["interpretation"]

	outputTiff := path.Join(channels, fmt.Sprintf("%s.tiff", basename))

	if Loader == "pdfload" {
		log.Println("Processing as PDF file")
		output, err := execCmd("pdfinfo", originalFilepath)
		if err != nil {
			return nil, err
		}

		var _pageSize string
		for _, line := range strings.Split(string(output), "\n") {
			if f, ok := extractField(line, "Page size:"); ok {
				_pageSize = f
			}
		}

		parts := strings.Split(_pageSize, " ")
		Width = parts[0]
		Height = parts[2]
		Units = parts[3]
		Colormode = "cmyk"

		log.Println("Start PDF rasterization...")

		args := []string{
			"-q",
			"-dBATCH",
			"-dNOPAUSE",
			"-dSAFER",
			"-dSubsetFonts=true",
			"-dMaxBitmap=500000000",
			"-dAlignToPixels=0",
			"-dGridFitTT=2",
			"-dTextAlphaBits=4",
			"-dGraphicsAlphaBits=4",
			"-dFirstPage=1",
			"-dLastPage=1",
			"-sDEVICE=tiffsep",
			fmt.Sprintf("-sOutputFile=%s", outputTiff),
			fmt.Sprintf("-r%s", c.Resolution),
			originalFilepath,
		}

		if _, err = execCmd("gs", args...); err != nil {
			return nil, err
		}

		log.Println("Done.")

	} else {
		log.Println("Processing as Image file")
		Units = "px"

		// Check supported colormodes
		if !slice.Contains(allowedColormodes, Colormode) {
			return nil, fmt.Errorf("unknow colormode, %s", Colormode)
		}

		if slice.Contains(rgbColormodes, Colormode) {
			log.Println("Convert RGB to CMYK with ICC profile:", c.ICCProfileFilepath)

			// Convert RGB to CMYK with ICC profile
			if _, err = execCmd("vips", "icc_export", originalFilepath, outputTiff, "--output-profile", c.ICCProfileFilepath); err != nil {
				return nil, err
			}

			if _, err = execCmd("cp", outputTiff, originalFilepath); err != nil {
				return nil, err
			}

		} else {
			log.Println("Convert to tiff")
			// Covert original file to TIFF
			if _, err = execCmd("vips", "tiffsave", originalFilepath, outputTiff, "--compression", "lzw"); err != nil {
				return nil, err
			}
		}

		Colormode = "cmyk"

		probe, err = vipsProbe(outputTiff, "bands")
		Bands, err = strconv.Atoi(probe["bands"])
		if err != nil {
			return nil, err
		}

		log.Println("Bands:", Bands)
		log.Println("Done.")

		for band := 0; band < Bands; band++ {
			var channelName string

			switch band {
			case 0:
				channelName = "Cyan"
			case 1:
				channelName = "Magenta"
			case 2:
				channelName = "Yellow"
			case 3:
				channelName = "Black"
			case 4:
				channelName = "Alpha"
			}

			outputChannel := path.Join(channels, fmt.Sprintf("%s(%s).tiff", basename, channelName))
			outputChannelInvert := path.Join(channels, fmt.Sprintf("%s(%s)_i.tiff", basename, channelName))

			if _, err = execCmd("vips", "extract_band", outputTiff, outputChannelInvert, strconv.Itoa(band)); err != nil {
				return nil, err
			}

			if _, err = execCmd("vips", "invert", outputChannelInvert, outputChannel); err != nil {
				return nil, err
			}

			if err = os.Remove(outputChannelInvert); err != nil {
				return nil, err
			}
		}
	}

	log.Println("Width:", Width)
	log.Println("Height:", Height)
	log.Println("Units:", Units)
	log.Println("Colormode:", Colormode)
	log.Println("Bands:", Bands)

	log.Println("Colorize channels")

	regExp, err := regexp.Compile(`(.*)\((.*)\)(.*)`)
	if err != nil {
		return nil, err
	}

	log.Println("Make leads and covers")
	entry, err := os.ReadDir(channels)
	if err != nil {
		return nil, err
	}
	var channelsArr []string
	for _, f := range entry {

		if f.IsDir() {
			continue
		}
		fpath := path.Join(channels, f.Name())
		fpath_bw := path.Join(channels_bw, f.Name())

		fext := path.Ext(f.Name())
		fbasename := strings.TrimSuffix(f.Name(), fext)
		log.Println("Processing file", fbasename, fext)

		var localColorMode string
		var localWidth, localHeight int

		vipsLocalProbe, err := vipsProbe(fpath, "interpretation", "width", "height")
		if err != nil {
			return nil, err
		}
		localColorMode = vipsLocalProbe["interpretation"]
		localWidth, _ = strconv.Atoi(vipsLocalProbe["width"])
		localHeight, _ = strconv.Atoi(vipsLocalProbe["height"])

		match := regExp.FindStringSubmatch(fbasename)
		if match != nil {
			if _, err = execCmd("cp", fpath, fpath_bw); err != nil {
				return nil, err
			}

			channelsArr = append(channelsArr, match[2])
			mateColor := CMYK[strings.ToLower(match[2])]
			if mateColor != "" {
				rgbMateColor, err := colorful.Hex(mateColor)
				if err != nil {
					return nil, err
				}
				_r, _g, _b := rgbMateColor.RGB255()
				mateRef, err := createImage(localWidth, localHeight, []float64{float64(_r), float64(_g), float64(_b)})
				if err != nil {
					return nil, err
				}

				channelRef, err := vips.LoadImageFromFile(fpath, nil)
				if err != nil {
					return nil, err
				}

				if err = channelRef.ToColorSpace(vips.InterpretationSRGB); err != nil {
					return nil, err
				}

				if err = mateRef.Composite(channelRef, vips.BlendModeScreen, 0, 0); err != nil {
					return nil, err
				}
				b, _, _ := mateRef.ExportPng(nil)
				os.Remove(fpath)
				fpath = fmt.Sprintf("%s.png", fpath)
				os.WriteFile(fpath, b, 0755)
				mateRef.Close()
				channelRef.Close()
				localColorMode = "srgb"
			}
		} else {
			if rgbCopyFilepath != "" {
				if _, err = execCmd("cp", rgbCopyFilepath, fpath); err != nil {
					return nil, err
				}
			}
		}

		if localColorMode == "" {
			return nil, fmt.Errorf("local colormode error")
		}

		leadPngPath := path.Join(leads, fmt.Sprintf("%s.png", fbasename))
		coverPngPath := path.Join(covers, fmt.Sprintf("%s.png", fbasename))
		leads1000PngPath := path.Join(leads1000, fmt.Sprintf("%s.png", fbasename))

		if localColorMode == "cmyk" {
			_, err = execCmd("vips", "icc_transform", fpath, leadPngPath, "srgb", "--input-profile", c.ICCProfileFilepath)
			if err != nil {
				return nil, err
			}
		} else {
			if _, err = execCmd("vips", "pngsave", fpath, leadPngPath); err != nil {
				return nil, err
			}
		}

		if _, err = execCmd("vips", "thumbnail", leadPngPath, leads1000PngPath, "1000"); err != nil {
			return nil, err
		}

		if _, err = execCmd("vips", "thumbnail", leadPngPath, coverPngPath, c.CoverWidth); err != nil {
			return nil, err
		}

	}

	log.Println("Make DZI - colors")
	if err = makeDZI(leads, dzi, c); err != nil {
		return nil, err
	}

	log.Println("Make DZI - b-w")
	if err = makeDZI(channels_bw, dzi_bw, c); err != nil {
		return nil, err
	}

	log.Println("Make manifest.json")
	var pSize = DziSize{
		Width:      Width,
		Height:     Height,
		CoverWidth: c.CoverWidth,
		Units:      Units,
		Dpi:        c.Resolution,
		Overlap:    c.Overlap,
		TileSize:   c.TileSize,
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
