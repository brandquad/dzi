package dzi

import (
	"errors"
	"fmt"
	"github.com/alitto/pond"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

type pageSize struct {
	PageNum int

	Spots []string

	WidthPt  float64
	HeightPt float64

	WidthInch  float64
	HeightInch float64

	WidthPx  int
	HeightPx int

	Dpi    int
	Rotate float64
}

// getPagesDimensions collect pages dimensions and spots colors from PDF file
func getPagesDimensions(fileName string, c *Config) ([]*pageSize, error) {
	pages := make([]*pageSize, 0)

	args := []string{
		fileName,
		"dump_data",
	}

	// Use pdftk to extract pages counter and pages dimensions
	buff, err := execCmd("pdftk", args...)
	if err != nil {
		return nil, err
	}

	output := string(buff)

	maxPages, err := strconv.Atoi(strings.TrimSpace(field(output, "NumberOfPages:")))
	outputLines := strings.Split(output, "\n")

	for i := 1; i <= maxPages; i++ {

		var ps *pageSize = nil
		var pageStartLine int = 0
		for lineNum, line := range outputLines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, fmt.Sprintf("PageMediaNumber: %d", i)) {
				pageStartLine = lineNum
				continue
			}
			if pageStartLine == 0 {
				continue
			}
			// Get page rotation ( but, not used)
			rotate := outputLines[pageStartLine+1]
			rotate = strings.TrimSpace(field(rotate, "PageMediaRotation:"))
			rotateFloat, err := strconv.ParseFloat(strings.TrimSpace(rotate), 64)
			if err != nil {
				return nil, err
			}

			// Get page dimensions
			dimensions := outputLines[pageStartLine+3]
			dimensions = strings.TrimSpace(field(dimensions, "PageMediaDimensions:"))
			dimensions = strings.ReplaceAll(dimensions, ",", "")
			dimensionsPair := strings.Split(dimensions, " ")

			// Convert string to float64
			widthFloat, err := strconv.ParseFloat(strings.TrimSpace(dimensionsPair[0]), 64)
			if err != nil {
				return nil, err
			}
			heightFloat, err := strconv.ParseFloat(strings.TrimSpace(dimensionsPair[1]), 64)
			if err != nil {
				return nil, err
			}

			dpi := c.DefaultDPI

			// Convert PostScript points to Inches
			widthInches := widthFloat * pt2in
			heightInches := heightFloat * pt2in

			// Convert Inches to pixels
			widthPx := widthInches * dpi
			heightPx := heightInches * dpi

			// Recalculate Dpi value based on max size in pixels
			var needRecalculate bool
			if widthPx > c.MaxSizePixels {
				dpi = c.MaxSizePixels / widthInches
				needRecalculate = true
			}
			if heightPx > c.MaxSizePixels {
				dpi = c.MaxSizePixels / heightInches
				needRecalculate = true
			}

			if !needRecalculate && widthPx < c.MaxSizePixels {
				dpi = c.MaxSizePixels / widthInches
				needRecalculate = true
			}

			if !needRecalculate && heightPx < c.MaxSizePixels {
				dpi = c.MaxSizePixels / heightInches
				needRecalculate = true
			}

			if int(dpi) < c.MinResolution {
				dpi = float64(c.MinResolution)
			}
			if int(dpi) > c.MaxResolution {
				dpi = float64(c.MaxResolution)
			}

			if needRecalculate {
				widthPx = widthInches * dpi
				heightPx = heightInches * dpi
			}
			ps = &pageSize{
				PageNum:    i,
				WidthPt:    widthFloat,
				HeightPt:   heightFloat,
				WidthInch:  widthInches,
				HeightInch: heightInches,
				WidthPx:    int(math.Ceil(widthPx)),
				HeightPx:   int(math.Ceil(heightPx)),
				Dpi:        int(math.Ceil(dpi)),
				Rotate:     rotateFloat,
				Spots:      []string{"Cyan", "Magenta", "Yellow", "Black"},
			}
			break
		}

		if ps != nil {
			pages = append(pages, ps)
		}
	}

	// Extract colors spots over ghostscript and info.ps script file
	args = []string{
		"-q",
		"-dNODISPLAY",
		"-dNOSAFER",
		fmt.Sprintf("-sFile=%s", fileName),
		"-dDumpFontsNeeded=false",
		"info.ps",
	}
	buff, err = execCmd("gs", args...)
	if err != nil && len(buff) == 0 {
		return nil, err
	}

	outputLines = strings.Split(string(buff), "\n")
	for _, page := range pages {
		for _, line := range outputLines {
			if strings.HasPrefix(line, fmt.Sprintf("Page %d", page.PageNum)) {
				triplet := strings.Split(line, "\t")
				spotsStr := triplet[2][1 : len(triplet[2])-1]
				if spotsStr != "" {
					for _, spot := range strings.Split(spotsStr, " #@ ") {
						spot = strings.TrimSpace(spot)
						if spot != "" {
							page.Spots = append(page.Spots, spot)
						}
					}
				}
			}
		}
	}

	return pages, nil
}

func renderPdf(fileName, outputPrefix, basename string, c *Config) ([]*pageSize, error) {
	st := time.Now()
	defer func() {
		log.Println("[*] Total render time:", time.Since(st))
	}()
	pages, err := getPagesDimensions(fileName, c)
	if err != nil {
		return nil, err
	}

	log.Println("[!] Pages count:", len(pages))

	splitChannels := c.SplitChannels
	for _, page := range pages {
		aspect := float64(page.WidthPx) / float64(page.HeightPx)

		if (aspect < 0.4 || aspect > 1.6) && page.Dpi < 200 {
			splitChannels = false
			c.SplitChannels = false
			log.Println("[!] Split channels disabled. It is non standard PDF format")
			break
		}
	}

	panicHandler := func(p interface{}) {
		fmt.Printf("[!] Task panicked: %v", p)
	}

	pool := pond.New(c.MaxCpuCount, len(pages), pond.MinWorkers(c.MaxCpuCount), pond.PanicHandler(panicHandler))

	for _, page := range pages {
		pool.Submit(func() {
			st := time.Now()
			log.Printf("[>] Render page #%d", page.PageNum)
			defer func() {
				log.Printf("[<] Render page #%d, at %s", page.PageNum, time.Since(st))
			}()

			outputFolder := fmt.Sprintf("%s/page_%d", outputPrefix, page.PageNum)
			if err := os.MkdirAll(outputFolder, DefaultFolderPerm); err != nil {
				panic(err)
			}

			if splitChannels {
				outputFilepath := fmt.Sprintf("%s/%s.tiff", outputFolder, basename)
				if err := callGS(fileName, outputFilepath, page, "tiffsep"); err != nil {
					panic(err)
				}
			} else {
				outputFilepath := fmt.Sprintf("%s/%s.png", outputFolder, basename)
				if err := callGS(fileName, outputFilepath, page, "png16m"); err != nil {
					panic(err)
				}
			}

		})
	}

	pool.StopAndWait()

	if pool.FailedTasks() > 0 {
		return nil, errors.New("error on PDF rendering")
	}
	return pages, nil
}
