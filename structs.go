package dzi

import (
	"path"
	"path/filepath"
	"strings"
)

var rgbColorModes = []string{"srgb", "rgb", "rgb16"}
var allColorModes = append([]string{"cmyk"}, rgbColorModes...)

type SwatchType string
type ColorMode string

const (
	CmykComponent SwatchType = "CmykComponent"
	SpotComponent SwatchType = "SpotComponent"
	Final         SwatchType = "Final"
)

const (
	ColorModeCMYK  ColorMode = "CMYK"
	ColorModeCMYKA ColorMode = "CMYKA"
	ColorModeRBG   ColorMode = "RBG"
	ColorModeRBGA  ColorMode = "RBGA"
)

type Swatch struct {
	Filepath string     `json:"-"`
	DziPath  string     `json:"dzi_path"`
	Name     string     `json:"name"`
	RBG      string     `json:"rgb"`
	Type     SwatchType `json:"type"`
	NeedMate bool       `json:"need_mate"`
}

func (s Swatch) Basename() string {
	return filepath.Base(s.Filepath)
}

func (s Swatch) Filename() string {
	ext := path.Ext(s.Basename())
	return strings.TrimSuffix(s.Basename(), ext)
}

type pageInfo struct {
	Prefix      string
	PageNumber  int
	Width       float64
	Height      float64
	ColorMode   ColorMode
	Unit        string
	Swatches    []*Swatch
	TextContent string
	Dpi         int
}

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
