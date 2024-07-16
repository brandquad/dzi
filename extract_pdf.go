package dzi

import (
	"encoding/xml"
	"errors"
	"fmt"
	poppler2 "github.com/johbar/go-poppler"
	"golang.org/x/text/encoding/charmap"
	"log"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
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
	var wPt, hPt float64
	p := doc.GetPage(pageNum - 1)
	wPt, hPt = p.Size()

	xmlString := doc.Info().Metadata
	var d pdfMeta
	var eg pdfEgMeta

	if strings.TrimSpace(xmlString) != "" {

		decoder := xml.NewDecoder(strings.NewReader(xmlString))
		if err := decoder.Decode(&d); err != nil {
			return nil, nil, err
		}
		decoder = xml.NewDecoder(strings.NewReader(xmlString))
		if err := decoder.Decode(&eg); err != nil {
			return nil, nil, err
		}
	}
	var egType, pdfType bool

	if wPt == 0 && hPt == 0 {

		if d.W > 0 && d.H > 0 {
			pdfType = true
			d.W = d.W
			d.H = d.H
		} else if eg.W > 0 && eg.H > 0 {
			egType = true

			d.W = eg.W
			d.H = eg.H

			if eg.Unit == "mm" {
				d.Unit = "Millimeters"
			} else {
				d.W /= pt2mm
				d.H /= pt2mm
				d.Unit = "Points"
			}
		} else {
			return nil, nil, errors.New("page size not defined")
		}
	} else {
		d.W = wPt
		d.H = hPt
	}

	swatchMap := make(map[string]Swatch)
	if len(eg.Inks) > 0 && len(d.SwatchGroups) == 0 {
		for _, e := range eg.Inks {
			sw := esko2swatch(e.Name, e.EGName, e.Type, e.Book, e.R, e.G, e.B)
			swatchMap[sw.Name] = sw
		}
	}

	if d.Unit == "" {
		d.Unit = "Millimeters"
	}

	if !egType && !pdfType {
		switch d.Unit {
		case "Millimeters":
			d.W /= pt2mm
			d.H /= pt2mm

		case "Centimeters":
			d.Unit = "cm"
			d.W /= pt2cm
			d.H /= pt2cm

		case "Inches":
			d.Unit = "in"
			d.W /= pt2in
			d.H /= pt2in

		case "Points":
			d.Unit = "mm"
			d.W /= pt2mm
			d.H /= pt2mm

		}
	}

	switch d.Unit {
	case "Millimeters":
		d.Unit = "mm"
	case "Centimeters":
		d.Unit = "cm"
	case "Inches":
		d.Unit = "in"
	case "Points":
		d.Unit = "mm"
	}

	if len(swatchMap) == 0 {
		for _, s := range d.SwatchGroups {
			var exists bool
			for _, pn := range d.PlateNames {
				sName := strings.TrimSpace(s.SwatchName)
				pn = strings.TrimSpace(pn)
				exists = pn == sName || strings.HasPrefix(sName, pn) || strings.HasSuffix(sName, pn)
				if exists {
					break
				}
			}

			if !exists {
				continue
			}

			var c []int
			switch strings.ToUpper(s.Mode) {
			case "LAB":
				c = lab2rgb([]float64{s.L, s.A, s.B})
			case "RGB":
				c = []int{s.Red, s.Green, s.Blue}
			case "CMYK":
				c = cmyk2rgb([]float64{
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

func extractPDF(filePath, baseName, outputFolder string, c Config) ([]*pageInfo, error) {

	// Render pages
	pagesSizes, err := renderPdf(filePath, outputFolder, baseName, c)
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

		if (page.Height == 0 || page.Width == 0) && len(pages) > 0 {
			page.Width = pages[0].Width
			page.Height = pages[0].Height
		}

		page, err = pageProcessing(outputFolder, page, swatchMap)
		if err != nil {
			return nil, err
		}

		for _, ps := range pagesSizes {
			if ps.PageNum == pageIndex {
				page.Dpi = ps.Dpi
			}
		}

		pages = append(pages, page)
	}

	return pages, nil
}

func pageProcessing(outputFolder string, info *pageInfo, swatchMap map[string]Swatch) (*pageInfo, error) {

	entries, err := os.ReadDir(path.Join(outputFolder, info.Prefix))
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		var name = entry.Name()
		var filePath = path.Join(outputFolder, info.Prefix, entry.Name())

		// Fix problem with cp-1251 in filenames
		name, err = url.QueryUnescape(name)
		if err != nil {
			return nil, err
		}
		dec := charmap.Windows1251.NewDecoder()
		if out, err := dec.String(name); err != nil {
			return nil, err
		} else {
			name = out

			newFilePath := path.Join(outputFolder, info.Prefix, name)
			if _, err := execCmd("mv", filePath, newFilePath); err != nil {
				return nil, err
			}
			filePath = newFilePath
		}

		swatchName := matchSwatch(name)

		if _, err = os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
			log.Printf("[-] Skipping file %s", filePath)
			continue
		}

		// Fix problem with equal spot and cmyk name (ex black1, yellow23)
		for _, cmykname := range []string{"black", "cyan", "yellow", "magenta"} {
			sw := strings.ToLower(swatchName)
			postfix := strings.TrimPrefix(sw, cmykname)
			if len(postfix) > 0 {
				if _, err = strconv.Atoi(postfix); err == nil {
					swatchName = strings.TrimSuffix(swatchName, postfix)
					break
				}
			}
		}

		swatchInfo := &Swatch{
			Filepath: filePath,
			Name:     swatchName,
			NeedMate: true,
		}
		if v, ok := swatchMap[swatchName]; !ok {
			swatchInfo.Type = CmykComponent
			swatchInfo.RBG = CMYK[strings.ToLower(swatchName)]
		} else {
			swatchInfo.Type = SpotComponent
			swatchInfo.RBG = v.RBG
		}

		if swatchName == "" {
			swatchInfo.Type = Final
			swatchInfo.Name = "Color"
			swatchInfo.NeedMate = false
		}

		info.Swatches = append(info.Swatches, swatchInfo)
	}

	return info, nil
}
