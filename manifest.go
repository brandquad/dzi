package dzi

import (
	"database/sql/driver"
	"encoding/json"
	"math"
	"strconv"
)

type DziSize struct {
	Width      string `json:"width"`
	Height     string `json:"height"`
	CoverWidth string `json:"cover_width"`
	Units      string `json:"units"`
	Dpi        string `json:"dpi"`
	Overlap    string `json:"overlap"`
	TileSize   string `json:"tile_size"`
}

type Manifest struct {
	ID        string   `json:"id"`
	Timestamp string   `json:"timestamp"`
	Source    string   `json:"source"`
	Filename  string   `json:"filename"`
	Basename  string   `json:"basename"`
	Mode      string   `json:"mode"`
	Size      DziSize  `json:"size"`
	Channels  []string `json:"channels"`
	Swatches  []swatch `json:"swatches,omitempty"`
}

func (b *Manifest) TileSize() int {
	if b.Size.TileSize == "" {
		return 255
	}
	return int(b.toFloat(b.Size.TileSize))
}

func (b *Manifest) toFloat(s string) float64 {
	if f, err := strconv.ParseFloat(s, 64); err != nil {
		return 0
	} else {
		return f
	}
}

func (b *Manifest) GetCoverSize() int {
	var s = b.toFloat(b.Size.CoverWidth)
	return int(s)
}

func (b *Manifest) GetWidth() float64 {
	return b.toFloat(b.Size.Width)
}

func (b *Manifest) toMM(unit string, x float64) float64 {
	switch unit {
	case "pts":
		return x * 0.3527777778
	case "pt":
		return x * 0.3527777778
	case "mm":
		return x
	case "Millimeters":
		return x
	case "cm":
		return x * 100
	case "in":
		return x * 25.4
	}
	return x

}

func (b *Manifest) GetWidthPixels() float64 {
	if b.Size.Units == "px" {
		return b.GetWidth()
	}
	return math.Ceil(b.toMM(b.Size.Units, b.GetWidth()) * (b.GetDPI() / 25.4))
}

func (b *Manifest) GetHeightPixels() float64 {
	if b.Size.Units == "px" {
		return b.GetHeight()
	}
	return math.Ceil(b.toMM(b.Size.Units, b.GetHeight()) * (b.GetDPI() / 25.4))
}

func (b *Manifest) GetHeight() float64 {
	return b.toFloat(b.Size.Height)
}

func (b *Manifest) GetDPI() float64 {
	if b.Size.Units == "px" {
		return 1
	}
	return b.toFloat(b.Size.Dpi)
}

func (b *Manifest) Scan(src interface{}) error {
	return JsonScan(src, b)
}
func (b Manifest) Value() (driver.Value, error) {
	return json.Marshal(b)
}
