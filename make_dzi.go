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

func makeDZI(pool *pond.WorkerPool, info []*pageInfo, income string, outcome string, c Config) error {

	for _, entry := range info {

		sourceFolder := path.Join(income, entry.Prefix)
		outcomeFolder := path.Join(outcome, entry.Prefix)

		if err := os.MkdirAll(outcomeFolder, DefaultFolderPerm); err != nil {
			return err
		}

		files, err := os.ReadDir(sourceFolder)
		if err != nil {
			return err
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}

			pool.Submit(func() {
				st := time.Now()

				fpath := path.Join(sourceFolder, f.Name())
				fext := path.Ext(f.Name())
				fbasename := strings.TrimSuffix(f.Name(), fext)
				dziPath := path.Join(outcomeFolder, fbasename)

				defer func() {
					log.Printf("dzsave for %s, at %s", dziPath, time.Since(st))
				}()

				if _, err = execCmd("vips", "dzsave",
					fpath,
					dziPath,
					"--strip",
					"--suffix",
					fmt.Sprintf(".%s%s", c.TileFormat, c.TileSetting),
					fmt.Sprintf("--vips-concurrency=%d", c.MaxCpuCount),
					fmt.Sprintf("--tile-size=%s", c.TileSize),
					fmt.Sprintf("--overlap=%s", c.Overlap)); err != nil {
					panic(err)
				}
			})

		}
	}
	return nil
}
