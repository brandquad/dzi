package dzi

import (
	"fmt"
	"github.com/alitto/pond"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

func makeDZI(pool *pond.WorkerPool, isBW bool, pages []*pageInfo, income, outcome string, c Config) error {

	for padeIdx, page := range pages {
		sourceFolder := path.Join(income, page.Prefix)
		outcomeFolder := path.Join(outcome, page.Prefix)
		if err := os.MkdirAll(outcomeFolder, DefaultFolderPerm); err != nil {
			return err
		}

		for swatchIdx, swatch := range page.Swatches {
			pool.Submit(func() {
				st := time.Now()

				sourceFilePath := path.Dir(swatch.Filepath)

				sourceFileExt := path.Ext(swatch.Filepath)
				sourceBasename := strings.TrimSuffix(strings.TrimPrefix(swatch.Filepath, sourceFilePath), sourceFileExt)[1:]
				dziPath := path.Join(outcomeFolder, sourceBasename)

				if sourceFileExt == ".tiff" && !isBW {

					log.Printf("[*] Convert %s to SRGB with profile %s", swatch.Filepath, c.ICCProfileFilepath)

					jpegFileName := fmt.Sprintf("%s.jpeg", sourceBasename)
					jpegPath := path.Join(sourceFolder, jpegFileName)

					if c.DebugMode {
						log.Printf("[D] vips icc_transform %s %s[Q=95] %s", swatch.Filepath, jpegPath, c.ICCProfileFilepath)
					}

					_, err := execCmd("vips", "icc_transform", swatch.Filepath, fmt.Sprintf("%s[Q=95]", jpegPath), c.ICCProfileFilepath)
					if err != nil {
						panic(err)
					}

					if err = os.Remove(swatch.Filepath); err != nil {
						panic(err)
					}
					swatch.Filepath = jpegPath
				}

				defer func() {
					log.Printf("[*] dzsave for %s , at %s", swatch.Filepath, time.Since(st))
				}()

				if _, err := execCmd("vips", "dzsave",
					swatch.Filepath,
					dziPath,
					"--strip",
					"--suffix",
					fmt.Sprintf(".%s%s", c.TileFormat, c.TileSetting),
					fmt.Sprintf("--vips-concurrency=%d", c.MaxCpuCount),
					fmt.Sprintf("--tile-size=%s", c.TileSize),
					fmt.Sprintf("--overlap=%s", c.Overlap)); err != nil {
					panic(err)
				}

				dziPath = fmt.Sprintf("%s_files/", dziPath)
				if isBW {
					pages[padeIdx].Swatches[swatchIdx].DziColorPath = dziPath
					//swatch.DziColorPath = dziPath
				} else {
					pages[padeIdx].Swatches[swatchIdx].DziBWPath = dziPath
					//swatch.DziBWPath = dziPath
				}
			})
		}
	}

	return nil
}
