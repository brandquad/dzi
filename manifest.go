package dzi

import (
	"database/sql/driver"
	"encoding/json"
	"math"
	"strconv"
)

type DziSize struct {
	Width  string `json:"width"`
	Height string `json:"height"`
	Units  string `json:"units"`
	//Dpi     string `json:"dpi"`

}

type Page struct {
	PageNum     int      `json:"pagenum"`
	Size        DziSize  `json:"size"`
	TextContent string   `json:"text_content"`
	Channels    []string `json:"channels"`
}

type Manifest struct {
	Version    string    `json:"version"`
	ID         string    `json:"id"`
	Timestamp  string    `json:"timestamp"`
	Source     string    `json:"source"`
	Filename   string    `json:"filename"`
	Basename   string    `json:"basename"`
	TileSize   string    `json:"tile_size"`
	CoverWidth string    `json:"cover_width"`
	Dpi        string    `json:"dpi"`
	Overlap    string    `json:"overlap"`
	Mode       string    `json:"mode"`
	Pages      []*Page   `json:"pages"`
	Swatches   []*Swatch `json:"swatches,omitempty"`
}

func (b *Manifest) Tilesize() int {
	if b.TileSize == "" {
		return 255
	}
	return int(b.toFloat(b.TileSize))
}

func (b *Manifest) toFloat(s string) float64 {
	if f, err := strconv.ParseFloat(s, 64); err != nil {
		return 0
	} else {
		return f
	}
}

func (b *Manifest) GetCoverSize() int {
	var s = b.toFloat(b.CoverWidth)
	return int(s)
}

func (b *Manifest) GetWidth(pageNum int) float64 {
	return b.toFloat(b.GetPageByNum(pageNum).Size.Width)
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

func (b *Manifest) GetWidthPixels(pageNum int) float64 {
	if b.GetPageByNum(pageNum).Size.Units == "px" {
		return b.GetWidth(pageNum)
	}
	return math.Ceil(b.toMM(b.GetPageByNum(pageNum).Size.Units, b.GetWidth(pageNum)) * (b.GetDPI(pageNum) / 25.4))
}

func (b *Manifest) GetHeightPixels(pageNum int) float64 {
	if b.GetPageByNum(pageNum).Size.Units == "px" {
		return b.GetHeight(pageNum)
	}
	return math.Ceil(b.toMM(b.GetPageByNum(pageNum).Size.Units, b.GetHeight(pageNum)) * (b.GetDPI(pageNum) / 25.4))
}

func (b *Manifest) GetHeight(pageNum int) float64 {
	return b.toFloat(b.GetPageByNum(pageNum).Size.Height)
}

func (b *Manifest) GetDPI(pageNum int) float64 {
	if b.GetPageByNum(pageNum).Size.Units == "px" {
		return 1
	}
	return b.toFloat(b.Dpi)
}

func (b *Manifest) GetPageByIndex(index int) *Page {
	return b.Pages[index]
}

func (b *Manifest) GetPageByNum(pageNum int) *Page {
	for _, page := range b.Pages {
		if page.PageNum == pageNum {
			return page
		}
	}
	return nil
}

func (b *Manifest) Scan(src interface{}) error {
	return JsonScan(src, b)
}
func (b Manifest) Value() (driver.Value, error) {
	return json.Marshal(b)
}
