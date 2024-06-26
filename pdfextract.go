package dzi

import (
	"encoding/xml"
	"errors"
	"fmt"
	poppler2 "github.com/johbar/go-poppler"
	"log"
	"os"
	"path"
	"strings"
)

const pt2mm = 2.8346456692913
const pt2cm = pt2mm * 10
const pt2in = 0.0138888889

type pdfEgMeta struct {
	Unit string  `xml:"RDF>Description>units"`
	W    float64 `xml:"RDF>Description>vsize"`
	H    float64 `xml:"RDF>Description>hsize"`
	Inks []struct {
		Name   string  `xml:"name"`
		Type   string  `xml:"type"`
		Book   string  `xml:"book"`
		EGName string  `xml:"egname"`
		R      float64 `xml:"r"`
		G      float64 `xml:"g"`
		B      float64 `xml:"b"`
	} `xml:"RDF>Description>inks>Seq>li"`
}

func esko2swatch(name, egname, egtype, book string, nr, ng, nb float64) Swatch {
	var swatchName = name
	if egtype == "pantone" {
		switch book {
		case "pms1000c":
			swatchName = fmt.Sprintf("PANTONE %s C", egname)
		case "pms1000u":
			swatchName = fmt.Sprintf("PANTONE %s U", egname)
		case "pms1000m":
			swatchName = fmt.Sprintf("PANTONE %s M", egname)
		case "goec":
			swatchName = fmt.Sprintf("PANTONE %s C", egname)
		case "goeu":
			swatchName = fmt.Sprintf("PANTONE %s U", egname)
		case "pmetc":
			swatchName = fmt.Sprintf("PANTONE %s C", egname)
		case "ppasc":
			swatchName = fmt.Sprintf("PANTONE %s C", egname)
		case "ppasu":
			swatchName = fmt.Sprintf("PANTONE %s U", egname)

		default:
			swatchName = name
		}
	}

	var swatchType SwatchType
	switch egtype {
	case "process":
		swatchType = CmykComponent
	case "pantone", "designer":
		swatchType = SpotComponent
	}

	var R = 255 * nr / 1
	var G = 255 * ng / 1
	var B = 255 * nb / 1

	return Swatch{
		Filepath: "",
		Name:     swatchName,
		RBG:      fmt.Sprintf("#%02x%02x%02x", int(R), int(G), int(B)),
		Type:     swatchType,
		NeedMate: true,
	}
}

type pdfMeta struct {
	W            float64  `xml:"RDF>Description>MaxPageSize>w"`
	H            float64  `xml:"RDF>Description>MaxPageSize>h"`
	Unit         string   `xml:"RDF>Description>MaxPageSize>unit"`
	PlateNames   []string `xml:"RDF>Description>PlateNames>Seq>li"`
	SwatchGroups []struct {
		SwatchName string  `xml:"swatchName"`
		Type       string  `xml:"type"`
		Mode       string  `xml:"mode"`
		L          float64 `xml:"L"`
		A          float64 `xml:"A"`
		B          float64 `xml:"B"`
		Cyan       float64 `xml:"cyan"`
		Magenta    float64 `xml:"magenta"`
		Yellow     float64 `xml:"yellow"`
		Black      float64 `xml:"black"`
		Red        int     `xml:"red"`
		Green      int     `xml:"green"`
		Blue       int     `xml:"blue"`
	} `xml:"RDF>Description>SwatchGroups>Seq>li>Colorants>Seq>li"`
}

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

func getEntryInfo(doc *poppler2.Document, pageNum int) (*entryInfo, map[string]Swatch, error) {
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

	return &entryInfo{
		Prefix:     fmt.Sprintf("page_%d", pageNum),
		PageNumber: pageNum,
		Width:      d.W,
		Height:     d.H,
		Unit:       d.Unit,
		ColorMode:  ColorModeCMYK,
		Swatches:   make([]Swatch, 0),
	}, swatchMap, nil
}

func pageProcessing(filepath, output, basename string, pageNum int, info *entryInfo, swatchMap map[string]Swatch, resolution int, splitChannels bool) (*entryInfo, error) {
	maxSpots := 0
	for _, swatch := range swatchMap {
		if swatch.Type == SpotComponent {
			maxSpots++
		}
	}
	if splitChannels {
		if err := runGS(filepath, path.Join(output, info.Prefix, fmt.Sprintf("%s.tiff", basename)), pageNum, resolution, maxSpots, true); err != nil {
			return nil, err
		}
	}
	if err := runGS(filepath, path.Join(output, info.Prefix, fmt.Sprintf("%s.jpeg", basename)), pageNum, resolution, maxSpots, false); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(path.Join(output, info.Prefix))
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		name := entry.Name()
		swatchName := matchSwatch(name)

		swatchInfo := Swatch{
			Filepath: path.Join(output, info.Prefix, entry.Name()),
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

func extractPDF(filepath string, basename string, output string, resolution int, splitChannels bool) ([]*entryInfo, error) {
	//gopopDoc, err := poppler1.NewFromFile(filepath, "")
	gopopDoc, err := poppler2.Open(filepath)
	if err != nil {
		return nil, err
	}

	pages := make([]*entryInfo, 0)
	totalPages := gopopDoc.GetNPages()
	//totalPages := gopopDoc.GetTotalPages()
	for pageIndex := 1; pageIndex <= totalPages; pageIndex++ {
		log.Printf("Processing page %d from %d", pageIndex, totalPages)
		info, swatchMap, err := getEntryInfo(gopopDoc, pageIndex)
		if err != nil {
			return nil, err
		}

		if err := os.MkdirAll(path.Join(output, info.Prefix), DefaultFolderPerm); err != nil {
			return nil, err
		}

		textContent, err := extractText(filepath, pageIndex)
		if err != nil {
			return nil, err
		}
		info.TextContent = textContent

		if (info.Height == 0 || info.Width == 0) && len(pages) > 0 {
			info.Width = pages[0].Width
			info.Height = pages[0].Height
		}

		info, err = pageProcessing(filepath, output, basename, pageIndex, info, swatchMap, resolution, splitChannels)
		if err != nil {
			return nil, err
		}
		pages = append(pages, info)

	}

	return pages, nil
}

func runGS(filename string, output string, pageNum, resolution, maxSpots int, splitChannels bool) error {
	var args []string
	if splitChannels {
		args = []string{
			"-q",
			"-dBATCH",
			"-dNOPAUSE",
			"-dSAFER",
			"-dSubsetFonts=true",
			"-dMaxBitmap=500000000",
			"-dAlignToPixels=0",
			"-dGridFitTT=2",
			"-dTextAlphaBits=4",
			"-dGraphicsAlphaBits=4",
			fmt.Sprintf("-dMaxSpots=%d", maxSpots),
			fmt.Sprintf("-dFirstPage=%d", pageNum),
			fmt.Sprintf("-dLastPage=%d", pageNum),
			"-sDEVICE=tiffsep",
			fmt.Sprintf("-r%d", resolution),
			fmt.Sprintf("-sOutputFile=%s", output),
			filename,
		}
	} else {
		args = []string{
			"-q",
			"-dBATCH",
			"-dNOPAUSE",
			"-dSAFER",
			"-dSubsetFonts=true",
			"-dMaxBitmap=500000000",
			"-dAlignToPixels=0",
			"-dGridFitTT=2",
			"-dTextAlphaBits=4",
			"-dGraphicsAlphaBits=4",
			fmt.Sprintf("-dMaxSpots=%d", maxSpots),
			fmt.Sprintf("-dFirstPage=%d", pageNum),
			fmt.Sprintf("-dLastPage=%d", pageNum),
			"-sDEVICE=jpeg",
			fmt.Sprintf("-r%d", resolution),
			fmt.Sprintf("-sOutputFile=%s", output),
			filename,
		}
	}

	if _, err := execCmd("gs", args...); err != nil {
		return err
	}

	if splitChannels {
		return os.Remove(output)
	}
	return nil
}
