package dzi

import (
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/lucasb-eyer/go-colorful"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func makeCovers(dziRootPath, leadsRoot, coversRoot string, c Config) error {
	st := time.Now()
	defer func() {
		log.Println("[<] Make covers, at", time.Since(st))
	}()

	type ff struct {
		Num  int
		Name string
		Path string
	}

	tileSize, _ := strconv.Atoi(c.TileSize)
	coverSize, _ := strconv.Atoi(c.CoverHeight)

	pagesFolders, err := os.ReadDir(dziRootPath)
	if err != nil {
		return err
	}
	for _, pageFolder := range pagesFolders {
		if !pageFolder.IsDir() {
			continue
		}

		dziFolders, err := os.ReadDir(path.Join(dziRootPath, pageFolder.Name()))
		if err != nil {
			return err
		}
		for _, dziFolder := range dziFolders {
			if !dziFolder.IsDir() {
				continue
			}
			levelFoldersPath := path.Join(dziRootPath, pageFolder.Name(), dziFolder.Name())
			levelFolders, err := os.ReadDir(levelFoldersPath)
			if err != nil {
				return err
			}
			var finalFolders = make([]ff, 0)
			for _, levelFolder := range levelFolders {
				if !levelFolder.IsDir() {
					continue
				}

				finalFolderNum, _ := strconv.Atoi(levelFolder.Name())
				finalFolders = append(finalFolders, ff{
					Num:  finalFolderNum,
					Name: levelFolder.Name(),
					Path: path.Join(levelFoldersPath, levelFolder.Name()),
				})

			}

			sort.Slice(finalFolders, func(i, j int) bool {
				return finalFolders[i].Num < finalFolders[j].Num
			})
			for i, j := 0, len(finalFolders)-1; i < j; i, j = i+1, j-1 {
				finalFolders[i], finalFolders[j] = finalFolders[j], finalFolders[i]
			}

			for _, f := range finalFolders {
				files, err := filepath.Glob(path.Join(f.Path, "*_0.webp"))
				if err != nil {
					return err
				}
				maxWidth := len(files) * tileSize
				if maxWidth <= 2000 {
					if err := collectLead(f.Path, leadsRoot, coversRoot, path.Join(pageFolder.Name(), dziFolder.Name()), tileSize, coverSize); err != nil {
						return err
					}
					break
				}

			}
		}

	}

	return err

}

func collectLead(folderPath, leadsRoot, coversRoot, relpath string, tileSize, coverSize int) error {
	var tile *vips.ImageRef
	ref, err := createImage(1, 1, colorful.Color{
		R: 0,
		G: 0,
		B: 0,
	})
	if err != nil {
		return err
	}
	defer func() {
		if ref != nil {
			ref.Close()
		}
		if tile != nil {
			tile.Close()
		}
	}()

	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}
	for _, file := range files {
		pairs := strings.Split(strings.TrimSuffix(file.Name(), path.Ext(file.Name())), "_")
		col, _ := strconv.Atoi(pairs[0])
		row, _ := strconv.Atoi(pairs[1])

		tile, err = vips.NewImageFromFile(path.Join(folderPath, file.Name()))
		if err != nil {
			return err
		}
		var x, y int
		x = col * tileSize
		y = row * tileSize

		if err := ref.Insert(tile, x, y, true, &vips.ColorRGBA{
			R: 0,
			G: 0,
			B: 0,
			A: 0,
		}); err != nil {
			return err
		}
	}

	buffer, _, err := ref.ExportPng(vips.NewPngExportParams())
	if err != nil {
		return err
	}

	finalLeadPath := path.Join(leadsRoot, strings.TrimSuffix(relpath, "_files"))
	if err := os.WriteFile(fmt.Sprintf("%s.png", finalLeadPath), buffer, 0777); err != nil {
		return err
	}

	if err := ref.Thumbnail(coverSize, coverSize, vips.InterestingAll); err != nil {
		return err
	}
	buffer, _, err = ref.ExportPng(vips.NewPngExportParams())
	if err != nil {
		return err
	}

	finalCoverPath := path.Join(coversRoot, strings.TrimSuffix(relpath, "_files"))
	if err := os.WriteFile(fmt.Sprintf("%s.png", finalCoverPath), buffer, 0777); err != nil {
		return err
	}

	return nil
}
