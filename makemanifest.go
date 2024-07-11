package dzi

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

func makeManifest(info []*pageInfo, assetId int, c Config, url, basename, filename string) (*Manifest, error) {
	log.Println("Make manifest.json")
	swatches := make([]*Swatch, 0)
	pages := make([]*Page, 0)

	for _, entry := range info {
		var channelsArr = make([]string, 0)

		for _, s := range entry.Swatches {

			var needAppend bool = true
			for _, sd := range swatches {
				if sd.Name == s.Name {
					needAppend = false
				}
			}

			if needAppend {
				swatches = append(swatches, s)
			}

			if s.Type != Final {
				channelsArr = append(channelsArr, s.Name)
			}
		}

		var wStr, hStr string
		if entry.Unit == "px" {
			wStr = fmt.Sprintf("%d", int(entry.Width))
			hStr = fmt.Sprintf("%d", int(entry.Height))
		} else {
			wStr = fmt.Sprintf("%f", entry.Width)
			hStr = fmt.Sprintf("%f", entry.Height)
		}

		pages = append(pages, &Page{
			PageNum: entry.PageNumber,
			Size: DziSize{
				Width:  wStr,
				Height: hStr,
				Units:  entry.Unit,
			},
			Channels:    channelsArr,
			TextContent: entry.TextContent,
		})

	}

	var manifest *Manifest = &Manifest{
		Version:       "2",
		ID:            strconv.Itoa(assetId),
		Timestamp:     time.Now().Format("2006-01-02 15:04:05"),
		Source:        url,
		Filename:      filename,
		Basename:      basename,
		TileSize:      c.TileSize,
		CoverHeight:   c.CoverHeight,
		Dpi:           c.Resolution,
		Overlap:       c.Overlap,
		Mode:          "CMYK",
		Pages:         pages,
		Swatches:      swatches,
		SplitChannels: c.SplitChannels,
	}

	return manifest, nil
}
