package dzi

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	dzi "github.com/brandquad/dzi/colorutils"
	poppler2 "github.com/johbar/go-poppler"
)

const pt2mm = 2.8346456692913
const pt2cm = pt2mm * 10
const pt2in = 0.0138888889

func extractText(filepath string, pageNum int) (string, error) {
	var result []string
	buffer, err := execCmd("mutool", "draw", "-q", "-F", "stext.json", filepath, fmt.Sprintf("%d", pageNum))
	for _, line := range strings.Split(string(buffer), "\n") {
		if strings.HasPrefix(line, "warning:") {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, ""), err
}

func getPageInfo(doc *poppler2.Document, pageNum int) (*pageInfo, map[string]Swatch, error) {
	xmlString := doc.Info().Metadata

	var d pdfMeta
	var eg pdfEgMeta

	if strings.TrimSpace(xmlString) != "" {
		func() {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("Error extracting page info, %d", pageNum+1)
					xmlString = ""
				}
			}()

			decoder := xml.NewDecoder(strings.NewReader(xmlString))
			if err := decoder.Decode(&d); err != nil {
				panic(err)
			}
			decoder = xml.NewDecoder(strings.NewReader(xmlString))
			if err := decoder.Decode(&eg); err != nil {
				panic(err)
			}
		}()
	}

	swatchMap := make(map[string]Swatch)
	//if len(eg.Inks) > 0 && len(d.SwatchGroups) == 0 {
	if len(eg.Inks) > 0 {
		for _, e := range eg.Inks {
			sw := esko2swatch(e.Name, e.EGName, e.Type, e.Book, e.R, e.G, e.B)
			swatchMap[sw.Name] = sw
		}
	}

	if len(swatchMap) == 0 {
		if len(d.SwatchGroups) > 0 {
			for idx, s := range d.SwatchGroups {

				codePoints := []rune(s.SwatchName) // Output: [A r t i o s _ È „ O ‚ I ‡]
				decoded := ""
				for _, codePoint := range codePoints {
					if codePoint >= 0x0400 && codePoint <= 0x04FF { // Cyrillic range
						decoded += string(codePoint)
					} else {
						decoded += string(codePoint)
					}
				}

				var exists bool

				if !slices.Contains(d.PlateNames, s.SwatchName) {
					exists = true
				} else {
					for _, pn := range d.PlateNames {
						sName := strings.TrimSpace(s.SwatchName)
						pn = strings.TrimSpace(pn)
						exists = pn == sName || strings.HasPrefix(sName, pn) || strings.HasSuffix(sName, pn)
						if exists {
							break
						}
					}
				}

				if !exists {
					d.SwatchGroups = append(d.SwatchGroups[:idx], d.SwatchGroups[idx+1:]...)
					continue
				}

				var c []int
				switch strings.ToUpper(s.Mode) {
				case "LAB":
					c = dzi.Lab2rgb([]float64{s.L, s.A, s.B})
				case "RGB":
					c = []int{s.Red, s.Green, s.Blue}
				case "CMYK":
					c = dzi.Cmyk2rgb([]float64{
						s.Cyan,
						s.Magenta,
						s.Yellow,
						s.Black},
					)
				}
				if c != nil {
					swatchMap[s.SwatchName] = Swatch{
						Name: s.SwatchName,
						RBG:  fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2]),
						Type: SpotComponent,
					}
				}
			}
		}
	}

	return &pageInfo{
		Prefix:     fmt.Sprintf("page_%d", pageNum),
		PageNumber: pageNum,
		Width:      d.W,
		Height:     d.H,
		Unit:       d.Unit,
		ColorMode:  ColorModeCMYK,
		Swatches:   make([]*Swatch, 0),
	}, swatchMap, nil
}

func extractPDF(filePath, baseName, outputFolder string, c *Config) ([]*pageInfo, error) {

	// Render pages
	pagesSizes, spots, err := renderPdf(filePath, outputFolder, baseName, c)
	if err != nil {
		return nil, err
	}

	gopopDoc, err := poppler2.Open(filePath)
	if err != nil {
		return nil, err
	}

	pages := make([]*pageInfo, 0)
	totalPages := gopopDoc.GetNPages()

	for pageIndex := 1; pageIndex <= totalPages; pageIndex++ {

		log.Printf("Processing page %d from %d", pageIndex, totalPages)
		page, swatchMap, err := getPageInfo(gopopDoc, pageIndex)
		if err != nil {
			return nil, err
		}
		if c.ExtractText {
			textContent, err := extractText(filePath, pageIndex)
			if err != nil {
				return nil, err
			}
			page.TextContent = textContent
		}

		page, err = pageProcessing(outputFolder, page, swatchMap, spots[pageIndex])
		if err != nil {
			return nil, err
		}

		for _, ps := range pagesSizes {
			if ps.PageNum == pageIndex {
				page.Dpi = ps.Dpi
				page.Width = ps.WidthPt / pt2mm
				page.Height = ps.HeightPt / pt2mm
				page.Unit = "mm"
			}
		}

		pages = append(pages, page)
	}

	return pages, nil
}

func pageProcessing(outputFolder string, info *pageInfo, swatchMap map[string]Swatch, channels channelsMap) (*pageInfo, error) {
	//var spotsBackUpExists []string
	//
	//for k := range channels {
	//	spotsBackUpExists = append(spotsBackUpExists, k)
	//}

	//existsChannelFiles, err := os.ReadDir(path.Join(outputFolder, info.Prefix))
	//if err != nil {
	//	return nil, err
	//}

	for name, channel := range channels {
		//var name = channel.Name()
		//var filePath = path.Join(outputFolder, info.Prefix, channel.Name())

		//swatchName := matchSwatch(name)

		if _, err := os.Stat(channel.Filepath); errors.Is(err, os.ErrNotExist) {
			log.Printf("[-] Skipping file %s", channel.Filepath)
			continue
		}

		var rgbComponents = channel.RgbComponents
		if swatchMap != nil {
			if v, ok := swatchMap[name]; ok {
				rgbComponents = v.RBG
			}
		}

		swatchInfo := &Swatch{
			Filepath: channel.Filepath,
			OpsName:  channel.OpsName,
			Name:     name,
			NeedMate: true,
			RBG:      rgbComponents,
		}
		if name == "Color" {
			swatchInfo.Type = Final
			swatchInfo.NeedMate = false
		}

		//if v, ok := channels[name]; ok {
		//	swatchInfo.Type = SpotComponent
		//	swatchInfo.RBG = v.RgbComponents
		//	swatchInfo.Filepath = v.Filepath
		//	swatchInfo.Name = swatchName
		//} else {
		//
		//	if vl2, okl2 := swatchMap[swatchName]; !okl2 {
		//		swatchInfo.Type = CmykComponent
		//
		//		if vl3, ok3 := channels[swatchName]; ok3 {
		//			swatchInfo.Type = SpotComponent
		//			swatchInfo.RBG = vl3.RgbComponents
		//			swatchInfo.Filepath = vl3.Filepath
		//		} else {
		//			swatchInfo.RBG = CMYK[strings.ToLower(swatchName)]
		//		}
		//	} else {
		//		swatchInfo.Type = SpotComponent
		//		swatchInfo.RBG = vl2.RBG
		//	}
		//}
		//
		//if swatchName == "" {
		//	swatchInfo.Type = Final
		//	swatchInfo.Name = "Color"
		//	swatchInfo.NeedMate = false
		//}

		info.Swatches = append(info.Swatches, swatchInfo)
	}

	return info, nil
}
