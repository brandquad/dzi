package dzi

import (
	"encoding/xml"
	"fmt"
	"github.com/nerdtakula/poppler"
	"os"
	"path"
	"strings"
)

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

func extractPDF(filepath string, basename string, output string, resolution int) (*entryInfo, error) {

	outputResult := path.Join(output, fmt.Sprintf("%s.tiff", basename))

	if err := runGS(filepath, outputResult, resolution); err != nil {
		return nil, err
	}

	doc, err := poppler.NewFromFile(filepath, "")
	if err != nil {
		panic(err)
	}
	xmlString := doc.GetMetadata()
	var d pdfMeta
	decoder := xml.NewDecoder(strings.NewReader(xmlString))
	if err = decoder.Decode(&d); err != nil {
		return nil, err
	}

	info := &entryInfo{
		Width:     d.W,
		Height:    d.H,
		Unit:      d.Unit,
		ColorMode: ColorModeCMYK,
		Swatches:  make([]swatch, 0),
	}

	info.Width = d.W
	info.Height = d.H
	info.Unit = d.Unit

	swatchMap := make(map[string]swatch)
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

func runGS(filename string, output string, resolution int) error {
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
