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
func makeCovers(pages []*pageInfo, leadsRoot, coversRoot string, c Config) error {
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

	var err error
	//pagesFolders, err := os.ReadDir(dziRootPath)
	//if err != nil {
	//	return err
	//}
	for _, page := range pages {
		//if !pageFolder.IsDir() {
		//	continue
		//}

		//dziFolders, err := os.ReadDir(path.Join(dziRootPath, pageFolder.Name()))
		//if err != nil {
		//	return err
		//}
		for _, swatch := range page.Swatches {
			//if !dziFolder.IsDir() {
			//	continue
			//}

			//levelFoldersPath := path.Join(dziRootPath, pageFolder.Name(), dziFolder.Name())
			levelFoldersPath := swatch.DziColorPath
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

					leadPath, coverPath, err := collectLead(f.Path, leadsRoot, coversRoot, page.Prefix, tileSize, coverSize)
					if err != nil {
						return err
					}
					swatch.CoverPath = coverPath
					swatch.LeadPath = leadPath
					break
				}

			}
		}
	}

	return err

}

// collectLead from folder with images and construct result image.
func collectLead(folderPath, leadsRoot, coversRoot, prefix string, tileSize, coverSize int) (string, string, error) {

	pathComponents := strings.Split(folderPath, string(os.PathSeparator))
	filename := pathComponents[len(pathComponents)-2]
	filename = strings.TrimSuffix(filename, "_files")
	filename = strings.ReplaceAll(filename, " ", "_")

	leadPath := path.Join(leadsRoot, prefix, fmt.Sprintf("%s.png", filename))
	coverPath := path.Join(coversRoot, prefix, fmt.Sprintf("%s.png", filename))

	if err := os.MkdirAll(path.Dir(leadPath), DefaultFolderPerm); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(path.Dir(coverPath), DefaultFolderPerm); err != nil {
		return "", "", err
	}

	var tileRef *vips.ImageRef
	targetRef, err := createImage(1, 1, colorful.Color{R: 0, G: 0, B: 0})
	if err != nil {
		return "", "", err
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
		return "", "", err
	}

	for _, file := range files {

		// 0_3.webp -> [0,3]
		pairs := strings.Split(strings.TrimSuffix(file.Name(), path.Ext(file.Name())), "_")
		col, _ := strconv.Atoi(pairs[0])
		row, _ := strconv.Atoi(pairs[1])

		tileRef, err = vips.NewImageFromFile(path.Join(folderPath, file.Name()))
		if err != nil {
			return "", "", err
		}
		var x, y int
		x = col * tileSize
		y = row * tileSize

		if tileRef.Bands() > 3 {
			if err = tileRef.ToColorSpace(vips.InterpretationSRGB); err != nil {
				return "", "", err
			}
		}
		// Insert tile to target image
		if err = targetRef.Insert(tileRef, x, y, true, &vips.ColorRGBA{
			R: 0,
			G: 0,
			B: 0,
			A: 0,
		}); err != nil {
			return "", "", err
		}
	}

	// Export as PNG data
	buffer, _, err := targetRef.ExportPng(vips.NewPngExportParams())
	if err != nil {
		return "", "", err
	}

	if err = os.WriteFile(leadPath, buffer, 0777); err != nil {
		return "", "", err
	}

	// Make cover image based on lead image
	if err = targetRef.Thumbnail(coverSize, coverSize, vips.InterestingAll); err != nil {
		return "", "", err
	}
	buffer, _, err = targetRef.ExportPng(vips.NewPngExportParams())
	if err != nil {
		return "", "", err
	}

	// Write file to covers folder
	if err = os.WriteFile(coverPath, buffer, 0777); err != nil {
		return "", "", err
	}

	return leadPath, coverPath, nil
}
