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
	Filepath string
	Name     string
	RBG      string
	Type     SwatchType
	NeedMate bool
}

func (s Swatch) Basename() string {
	return filepath.Base(s.Filepath)
}

func (s Swatch) Filename() string {
	ext := path.Ext(s.Basename())
	return strings.TrimSuffix(s.Basename(), ext)
}

type entryInfo struct {
	Prefix      string
	PageNumber  int
	Width       float64
	Height      float64
	ColorMode   ColorMode
	Unit        string
	Swatches    []Swatch
	TextContent string
}
