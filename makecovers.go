package dzi

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
)

func makeCovers(dziRootPath string, c Config) error {
	//pageMap := make(map[string]string)
	type ff struct {
		Num  int
		Name string
		Path string
	}

	tileSize, _ := strconv.Atoi(c.TileSize)

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
					log.Println(maxWidth, f.Path)
					if err := collectLead(f.Path); err != nil {
						return err
					}
					break
				}

			}
		}

	}

	//err := filepath.Walk(dziRootPath, func(pageFolder string, info os.FileInfo, err error) error {
	//	if pageFolder != dziRootPath && info.IsDir() {
	//
	//		err = filepath.Walk(pageFolder, func(dziFileFolder string, subInfo os.FileInfo, err error) error {
	//
	//			if dziFileFolder != pageFolder && subInfo.IsDir() && strings.HasSuffix(subInfo.Name(), "_files") {
	//
	//				var folders = make([]ff, 0)
	//				err = filepath.Walk(dziRootPath, func(levelFolder string, subSubInfo os.FileInfo, err error) error {
	//
	//					if levelFolder != dziFileFolder && subSubInfo.IsDir() {
	//						finalFolders, err := os.ReadDir(levelFolder)
	//						if err != nil {
	//							return err
	//						}
	//						for _, f := range finalFolders {
	//							if f.IsDir() {
	//								finalFolderNum, _ := strconv.Atoi(f.Name())
	//								folders = append(folders, ff{
	//									Num:  finalFolderNum,
	//									Name: f.Name(),
	//									Path: path.Join(levelFolder, f.Name()),
	//								})
	//							}
	//						}
	//						sort.Slice(folders, func(i, j int) bool {
	//							return folders[i].Num < folders[j].Num
	//						})
	//						for i, j := 0, len(folders)-1; i < j; i, j = i+1, j-1 {
	//							folders[i], folders[j] = folders[j], folders[i]
	//						}
	//						for _, f := range folders {
	//							files, err := filepath.Glob(path.Join(f.Path, "*_0.webp"))
	//							if err != nil {
	//								return err
	//							}
	//							maxWidth := len(files) * tileSize
	//							if maxWidth <= 2000 {
	//								log.Println(maxWidth, f.Path)
	//								if err := collectLead(f.Path); err != nil {
	//									return err
	//								}
	//								break
	//							}
	//
	//						}
	//					}
	//					return nil
	//				})
	//			}
	//
	//			return nil
	//		})
	//		if err != nil {
	//			return err
	//		}
	//	}
	//	return nil
	//})

	return err

}

func collectLead(folderPath string) error {
	//var matrix [][]string
	//files, err := filepath.Glob(path.Join(folderPath, "*.webp"))

	return nil
}
