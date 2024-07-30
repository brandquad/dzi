package dzi

import (
	"archive/zip"
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/lucasb-eyer/go-colorful"
	"log"
	"os"
	"path"
	"slices"
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
		Num   int
		Path  string
		Files []string
	}

	tileSize, _ := strconv.Atoi(c.TileSize)
	coverSize, _ := strconv.Atoi(c.CoverHeight)

	var archive *zip.ReadCloser
	var err error
	for _, page := range pages {
		for _, swatch := range page.Swatches {
			archive, err = zip.OpenReader(swatch.DziColorPath)
			if err != nil {
				return err
			}

			var folders = make([]folderStruct, 0)
			for _, file := range archive.File {

				dir := path.Dir(file.Name)
				_p := strings.Split(dir, string(os.PathSeparator))
				dir = _p[len(_p)-1]
				folderNum, _ := strconv.Atoi(dir)

				idx := slices.IndexFunc(folders, func(s folderStruct) bool {
					return s.Num == folderNum
				})

				if idx == -1 {
					folderPath := strings.Join(_p, string(os.PathSeparator))
					files := make([]string, 0)
					for _, f := range archive.File {
						if strings.HasSuffix(f.Name, c.TileFormat) && strings.HasPrefix(f.Name, folderPath+string(os.PathSeparator)) {
							files = append(files, f.Name)
						}
					}

					folders = append(folders, folderStruct{
						Num:   folderNum,
						Path:  folderPath,
						Files: files,
					})
				}
			}
			// Sort folders by level (Small -> Big)
			sort.Slice(folders, func(i, j int) bool {
				return folders[i].Num < folders[j].Num
			})
			// Reverse slice
			//for i, j := 0, len(folders)-1; i < j; i, j = i+1, j-1 {
			//	folders[i], folders[j] = folders[j], folders[i]
			//}

			// Check each level
			for _, f := range folders {
				maxWidth := len(f.Files) * tileSize
				if maxWidth >= 2000 {

					leadPath, coverPath, err := collectLead(archive, f.Files, f.Path, leadsRoot, coversRoot, page.Prefix, tileSize, coverSize)
					if err != nil {
						return err
					}
					swatch.CoverPath = coverPath
					swatch.LeadPath = leadPath
					break
				}

			}
			archive.Close()
		}
	}

	return err

}

// collectLead from folder with images and construct result image.
func collectLead(archive *zip.ReadCloser, files []string, folderPath, leadsRoot, coversRoot, prefix string, tileSize, coverSize int) (string, string, error) {

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

	for _, file := range files {

		pairS1 := strings.Split(file, "/")
		pairS2 := pairS1[len(pairS1)-1]

		// 0_3.webp -> [0,3]
		pairs := strings.Split(strings.TrimSuffix(pairS2, path.Ext(file)), "_")
		col, _ := strconv.Atoi(pairs[0])
		row, _ := strconv.Atoi(pairs[1])

		fp, err := archive.Open(file)
		if err != nil {
			return "", "", err
		}

		tileRef, err = vips.NewImageFromReader(fp)
		if err != nil {
			return "", "", err
		}
		var x, y int
		x = col * tileSize
		y = row * tileSize

		if tileRef.HasAlpha() {
			// Remove alpha channel
			if err := tileRef.ExtractBand(0, tileRef.Bands()-1); err != nil {
				return "", "", err
			}
		}
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
