package dzi

import (
	"encoding/xml"
	"fmt"
	poppler2 "github.com/johbar/go-poppler"
	"github.com/nerdtakula/poppler"
	"os"
	"path"
	"strings"
)

const pt2mm = 2.8346456692913
const pt2cm = pt2mm * 10
const pt2in = 0.0138888889

type pdfEgMeta struct {
	Unit string `xml:"RDF>Description>units"`
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

func esko2swatch(name, egname, egtype, book string, nr, ng, nb float64) swatch {
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

	return swatch{
		Filepath: "",
		Name:     swatchName,
		RBG:      fmt.Sprintf("#%02x%02x%02x", int(R), int(G), int(B)),
		Type:     swatchType,
		NeedMate: true,
	}
}

type pdfMeta struct {
	W            float64
	H            float64
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

func extractPDF(filepath string, basename string, output string, resolution int) (*entryInfo, error) {

	outputResult := path.Join(output, fmt.Sprintf("%s.tiff", basename))

	gopopDoc, err := poppler2.Open(filepath)
	if err != nil {
		return nil, err
	}
	var wPt, hPt float64
	for i := 0; i < gopopDoc.GetNPages(); i++ {
		p := gopopDoc.GetPage(i)
		wPt, hPt = p.Size()
		break
	}

	doc, err := poppler.NewFromFile(filepath, "")
	if err != nil {
		panic(err)
	}

	xmlString := doc.GetMetadata()
	var d pdfMeta
	var eg pdfEgMeta

	decoder := xml.NewDecoder(strings.NewReader(xmlString))
	if err = decoder.Decode(&d); err != nil {
		return nil, err
	}
	decoder = xml.NewDecoder(strings.NewReader(xmlString))

	if err = decoder.Decode(&eg); err != nil {
		return nil, err
	}

	swatchMap := make(map[string]swatch)
	if len(eg.Inks) > 0 {

		if eg.Unit == "mm" {
			d.Unit = "Millimeters"
		} else {
			d.Unit = "Points"
		}

		for _, e := range eg.Inks {
			sw := esko2swatch(e.Name, e.EGName, e.Type, e.Book, e.R, e.G, e.B)
			swatchMap[sw.Name] = sw
		}
	}

	if d.Unit == "" {
		d.Unit = "Millimeters"
	}

	if d.Unit == "Millimeters" {
		d.Unit = "mm"
		d.W = wPt / pt2mm
		d.H = hPt / pt2mm
	}

	if d.Unit == "Centimeters" {
		d.Unit = "cm"
		d.W = wPt / pt2cm
		d.H = hPt / pt2cm
	}

	if d.Unit == "Inches" {
		d.Unit = "in"
		d.W = wPt / pt2in
		d.H = hPt / pt2in
	}

	if d.Unit == "Points" {
		d.Unit = "mm"
		d.W = wPt / pt2mm
		d.H = hPt / pt2mm
	}

	var maxSpots = 0
	for _, sw := range swatchMap {
		if sw.Type == SpotComponent {
			maxSpots++
		}
	}

	if err = runGS(filepath, outputResult, resolution, maxSpots); err != nil {
		return nil, err
	}

	info := &entryInfo{
		Width:     d.W,
		Height:    d.H,
		Unit:      d.Unit,
		ColorMode: ColorModeCMYK,
		Swatches:  make([]swatch, 0),
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
			swatchMap[s.SwatchName] = swatch{
				Name: s.SwatchName,
				RBG:  fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2]),
				Type: SpotComponent,
			}
		}
	}

	entries, err := os.ReadDir(output)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		name := entry.Name()
		swatchName := matchSwatch(name)

		swatchInfo := swatch{
			Filepath: path.Join(output, entry.Name()),
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

func runGS(filename string, output string, resolution, maxSpots int) error {
	args := []string{
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
		"-dFirstPage=1",
		"-dLastPage=1",
		fmt.Sprintf("-dMaxSpots=%d", maxSpots),
		"-sDEVICE=tiffsep",
		fmt.Sprintf("-sOutputFile=%s", output),
		fmt.Sprintf("-r%d", resolution),
		filename,
	}

	if _, err := execCmd("gs", args...); err != nil {
		return err
	}
	return nil
}
