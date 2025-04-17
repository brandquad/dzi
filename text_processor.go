package dzi

import (
	"encoding/json"
	"fmt"
	"strings"
)

type textBBox struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}
type textBlocks struct {
	Type string   `json:"type"`
	Text string   `json:"text"`
	BBox textBBox `json:"bbox"`
}

type textPages struct {
	PageNum int          `json:"page_num"`
	Blocks  []textBlocks `json:"blocks"`
}

// Structs to parse raw json output from mutool
type rawFont struct {
	Name   string `json:"name"`
	Family string `json:"family"`
	Weight string `json:"weight"`
	Style  string `json:"style"`
	Size   int    `json:"size"`
}
type rawLine struct {
	WMode int      `json:"wmode"`
	BBox  textBBox `json:"bbox"`
	Font  rawFont  `json:"font"`
	X     float64  `json:"x"`
	Y     float64  `json:"y"`
	Text  string   `json:"text"`
}

type rawBlock struct {
	Type  string    `json:"type"`
	BBox  textBBox  `json:"bbox"`
	Lines []rawLine `json:"lines"`
}

type rawPage struct {
	Blocks []rawBlock `json:"blocks"`
}

type rawResult struct {
	Pages []rawPage `json:"pages"`
	File  string    `json:"file"`
}

func TextExtractor(filepath string, pagenum int) ([]textPages, error) {
	var result []textPages
	buffer, err := execCmd("mutool", "draw", "-q", "-F", "stext.json", filepath, fmt.Sprintf("%d", pagenum))
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(string(buffer), "warning:") {
		strBuffer := strings.Split(string(buffer), "\n")
		buffer = []byte(strings.Join(strBuffer[1:], ""))
	}

	var raw rawResult
	err = json.Unmarshal(buffer, &raw)
	if err != nil {
		return nil, err
	}

	for pn, o := range raw.Pages {
		page := textPages{
			PageNum: pn,
			Blocks:  make([]textBlocks, 0),
		}

		for _, b := range o.Blocks {
			text := []string{}

			for _, l := range b.Lines {
				text = append(text, l.Text)
			}

			page.Blocks = append(page.Blocks, textBlocks{
				Type: b.Type,
				Text: strings.Join(text, "\n"),
				BBox: b.BBox,
			})
		}

		result = append(result, page)

	}

	return result, nil

}
