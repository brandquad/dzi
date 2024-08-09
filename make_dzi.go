package dzi

import (
	"archive/zip"
	"fmt"
	"github.com/alitto/pond"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

func makeDZI(pool *pond.WorkerPool, isBW bool, pages []*pageInfo, income, outcome string, c *Config) error {

	for padeIdx, page := range pages {
		sourceFolder := path.Join(income, page.Prefix)
		outcomeFolder := path.Join(outcome, page.Prefix)
		if err := os.MkdirAll(outcomeFolder, DefaultFolderPerm); err != nil {
			return err
		}

		for swatchIdx, swatch := range page.Swatches {
			pool.Submit(func() {
				st := time.Now()
				filepath := swatch.Filepath
				if isBW {
					filepath = strings.ReplaceAll(filepath, "/channels/", "/channels_bw/")
					_ext := path.Ext(filepath)
					filepath = strings.ReplaceAll(filepath, _ext, ".tiff")
					if _, err := os.Stat(filepath); os.IsNotExist(err) {
						log.Println("[-] File does not exist:", filepath)
						return
					}
				}

				sourceFilePath := path.Dir(filepath)

				sourceFileExt := path.Ext(filepath)
				sourceBasename := strings.TrimSuffix(strings.TrimPrefix(filepath, sourceFilePath), sourceFileExt)[1:]
				dziPath := path.Join(outcomeFolder, sourceBasename)

				if sourceFileExt == ".tiff" && !isBW {

					log.Printf("[*] Convert %s to SRGB with profile %s", filepath, c.ICCProfileFilepath)

					jpegFileName := fmt.Sprintf("%s.jpeg", sourceBasename)
					jpegPath := path.Join(sourceFolder, jpegFileName)

					if c.DebugMode {
						log.Printf("[D] vips icc_transform %s %s[Q=95] %s", filepath, jpegPath, c.ICCProfileFilepath)
					}

					_, err := execCmd("vips", "icc_transform", filepath, fmt.Sprintf("%s[Q=95]", jpegPath), c.ICCProfileFilepath)
					if err != nil {
						panic(err)
					}

					if err = os.Remove(filepath); err != nil {
						panic(err)
					}
					filepath = jpegPath
					swatch.Filepath = jpegPath
				}

				defer func() {
					log.Printf("[*] dzsave for %s , at %s", filepath, time.Since(st))
				}()
				dziPath = fmt.Sprintf("%s.zip", dziPath)
				if _, err := execCmd("vips", "dzsave",
					filepath,
					dziPath,
					"--strip",
					"--container=zip",
					"--suffix",
					fmt.Sprintf(".%s%s", c.TileFormat, c.TileSetting),
					fmt.Sprintf("--vips-concurrency=%d", c.MaxCpuCount),
					fmt.Sprintf("--tile-size=%s", c.TileSize),
					fmt.Sprintf("--overlap=%s", c.Overlap)); err != nil {
					panic(err)
				}

				rangesData, err := ranges(dziPath)
				if err != nil {
					panic(err)
				}

				if isBW {
					pages[padeIdx].Swatches[swatchIdx].DziBWPath = dziPath
					pages[padeIdx].Swatches[swatchIdx].DziBWRanges = rangesData
				} else {
					pages[padeIdx].Swatches[swatchIdx].DziColorPath = dziPath
					pages[padeIdx].Swatches[swatchIdx].DziColorRanges = rangesData
				}
			})
		}
	}

	return nil
}

func ranges(zipfile string) (map[string]ZipRange, error) {
	result := make(map[string]ZipRange)
	reader, err := zip.OpenReader(zipfile)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, ".dzi") || strings.HasSuffix(file.Name, ".xml") {
			continue
		}

		offset, err := file.DataOffset()
		if err != nil {
			return nil, err
		}

		parts := strings.Split(file.Name, "/")
		name := path.Join(parts[len(parts)-2], parts[len(parts)-1])
		result[name] = ZipRange{Offset: uint64(offset), Length: file.CompressedSize64}
	}
	return result, nil

}
