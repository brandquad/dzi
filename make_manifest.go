package dzi

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

func makeManifest(pages []*pageInfo, assetId int, c Config, url, basename, filename string) (*Manifest, error) {
	st := time.Now()
	log.Println("[>] Make manifest.json")
	defer func() {
		log.Printf("[<] Make manifest.json, at %s", time.Since(st))
	}()

	swatches := make([]*Swatch, 0)
	manifestPages := make([]*Page, 0)

	for _, page := range pages {
		var channelsArr = make([]string, 0)

		for _, s := range page.Swatches {

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
		if page.Unit == "px" {
			wStr = fmt.Sprintf("%d", int(page.Width))
			hStr = fmt.Sprintf("%d", int(page.Height))
		} else {
			wStr = fmt.Sprintf("%f", page.Width)
			hStr = fmt.Sprintf("%f", page.Height)
		}

		manifestPages = append(manifestPages, &Page{
			PageNum: page.PageNumber,
			Size: DziSize{
				Width:  wStr,
				Height: hStr,
				Units:  page.Unit,
				Dpi:    strconv.Itoa(page.Dpi),
			},
			Channels:    channelsArr,
			TextContent: page.TextContent,
		})

	}

	var manifest = &Manifest{
		Version:       "3",
		ID:            strconv.Itoa(assetId),
		Timestamp:     time.Now().Format("2006-01-02 15:04:05"),
		Source:        url,
		Filename:      filename,
		Basename:      basename,
		TileSize:      c.TileSize,
		TileFormat:    c.TileFormat,
		CoverHeight:   c.CoverHeight,
		Overlap:       c.Overlap,
		Mode:          "CMYK",
		Pages:         manifestPages,
		Swatches:      swatches,
		SplitChannels: c.SplitChannels,
	}

	return manifest, nil
}
