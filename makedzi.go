package dzi

import (
	"fmt"
	"os"
	"path"
	"strings"
)

func makeDZI(info []*pageInfo, income string, outcome string, c Config) error {

	for _, entry := range info {

		sourceFolder := path.Join(income, entry.Prefix)
		outcomeFolder := path.Join(outcome, entry.Prefix)

		os.MkdirAll(outcomeFolder, DefaultFolderPerm)

		files, err := os.ReadDir(sourceFolder)
		if err != nil {
			return err
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}

			fpath := path.Join(sourceFolder, f.Name())
			fext := path.Ext(f.Name())
			fbasename := strings.TrimSuffix(f.Name(), fext)
			dziPath := path.Join(outcomeFolder, fbasename)

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
	}
	return nil
}
