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

// makeCovers function to construct preview images through DZI tiles
func makeCovers(dziRootPath, leadsRoot, coversRoot string, c Config) error {
	st := time.Now()
	log.Println("[>] Make covers")
	defer func() {
		log.Println("[<] Make covers, at", time.Since(st))
	}()

	type folderStruct struct {
		Num  int
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
			var finalFolders = make([]folderStruct, 0)
			for _, levelFolder := range levelFolders {
				if !levelFolder.IsDir() {
					continue
				}

				finalFolderNum, _ := strconv.Atoi(levelFolder.Name())
				finalFolders = append(finalFolders, folderStruct{
					Num:  finalFolderNum,
					Path: path.Join(levelFoldersPath, levelFolder.Name()),
				})

			}
			// Sort folders by level (Small -> Big)
			sort.Slice(finalFolders, func(i, j int) bool {
				return finalFolders[i].Num < finalFolders[j].Num
			})
			// Reverse slice
			for i, j := 0, len(finalFolders)-1; i < j; i, j = i+1, j-1 {
				finalFolders[i], finalFolders[j] = finalFolders[j], finalFolders[i]
			}

			// Check each level
			for _, f := range finalFolders {
				files, err := filepath.Glob(path.Join(f.Path, fmt.Sprintf("*_0.%s", c.TileFormat)))
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

// collectLead from folder with images and construct result image.
func collectLead(folderPath, leadsRoot, coversRoot, relPath string, tileSize, coverSize int) error {

	var tileRef *vips.ImageRef
	targetRef, err := createImage(1, 1, colorful.Color{R: 0, G: 0, B: 0})
	if err != nil {
		return err
	}

	defer func() {
		if targetRef != nil {
			targetRef.Close()
		}
		if tileRef != nil {
			tileRef.Close()
		}
	}()

	// Loop by file and insert each to targetRef image
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}
	for _, file := range files {
		// 0_3.webp -> [0,3]
		pairs := strings.Split(strings.TrimSuffix(file.Name(), path.Ext(file.Name())), "_")
		col, _ := strconv.Atoi(pairs[0])
		row, _ := strconv.Atoi(pairs[1])

		tileRef, err = vips.NewImageFromFile(path.Join(folderPath, file.Name()))
		if err != nil {
			return err
		}
		var x, y int
		x = col * tileSize
		y = row * tileSize

		if tileRef.Bands() > 3 {
			if err = tileRef.ToColorSpace(vips.InterpretationSRGB); err != nil {
				return err
			}
		}
		// Insert tile to target image
		if err := targetRef.Insert(tileRef, x, y, true, &vips.ColorRGBA{
			R: 0,
			G: 0,
			B: 0,
			A: 0,
		}); err != nil {
			return err
		}
	}

	// Export as PNG data
	buffer, _, err := targetRef.ExportPng(vips.NewPngExportParams())
	if err != nil {
		return err
	}

	// Write file to leads folder
	finalLeadPath := path.Join(leadsRoot, strings.TrimSuffix(relPath, "_files"))
	if err := os.WriteFile(fmt.Sprintf("%s.png", finalLeadPath), buffer, 0777); err != nil {
		return err
	}

	// Make cover image based on lead image
	if err := targetRef.Thumbnail(coverSize, coverSize, vips.InterestingAll); err != nil {
		return err
	}
	buffer, _, err = targetRef.ExportPng(vips.NewPngExportParams())
	if err != nil {
		return err
	}

	// Write file to covers folder
	finalCoverPath := path.Join(coversRoot, strings.TrimSuffix(relPath, "_files"))
	if err := os.WriteFile(fmt.Sprintf("%s.png", finalCoverPath), buffer, 0777); err != nil {
		return err
	}

	return nil
}
