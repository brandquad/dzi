package dzi

import (
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/alitto/pond"
	"log"
	"math"
	"os"
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

	type muBox struct {
		B float64 `xml:"b,attr"`
		L float64 `xml:"l,attr"`
		R float64 `xml:"r,attr"`
		T float64 `xml:"t,attr"`
	}
	type muRotate struct {
		Rotate float64 `xml:"v,attr"`
	}
	type muPage struct {
		PageNum  int      `xml:"pagenum,attr"`
		MediaBox muBox    `xml:"MediaBox"`
		Rotate   muRotate `xml:"Rotate"`
	}
	type muDoc struct {
		Pages []muPage `xml:"page"`
	}

	args := []string{
		"pages",
		fileName,
	}

	// Use pdftk to extract pages counter and pages dimensions
	buff, err := execCmd("mutool", args...)
	if err != nil {
		return nil, err
	}
	o := string(buff)
	if strings.HasPrefix(o, fileName) {
		o = strings.TrimPrefix(o, fileName+":")
	}
	output := fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?><xml>%s</xml>", o)

	var mudoc muDoc
	err = xml.Unmarshal([]byte(output), &mudoc)
	if err != nil {
		return nil, err
	}
	pages := make([]*pageSize, len(mudoc.Pages))

	for idx, p := range mudoc.Pages {

		var ps = &pageSize{
			PageNum:  p.PageNum,
			WidthPt:  p.MediaBox.R + math.Abs(p.MediaBox.L),
			HeightPt: p.MediaBox.T + math.Abs(p.MediaBox.B),
			Rotate:   p.Rotate.Rotate,
		}

		if ps.Rotate == 90.0 {
			_t := ps.HeightPt
			ps.HeightPt = ps.WidthPt
			ps.WidthPt = _t
		}

		dpi := c.DefaultDPI

		// Convert PostScript points to Inches
		widthInches := ps.WidthPt * pt2in
		heightInches := ps.HeightPt * pt2in

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

			// Fix ME-67. Extreme broken PDF size
			if widthInches*dpi/3 > float64(c.MaxSizePixels) || heightInches*dpi/3 > float64(c.MaxSizePixels) {
				dpi /= 3
			}

		}
		if int(dpi) > c.MaxResolution {
			dpi = float64(c.MaxResolution)
		}

		if needRecalculate {
			widthPx = widthInches * dpi
			heightPx = heightInches * dpi
		}

		ps.Dpi = int(dpi)
		ps.WidthPx = int(widthPx)
		ps.HeightPx = int(heightPx)
		ps.WidthInch = widthInches
		ps.HeightInch = heightInches

		pages[idx] = ps
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

	outputLines := strings.Split(string(buff), "\n")
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

func renderPdf(fileName, outputPrefix, basename string, c *Config) ([]*pageSize, map[string][]int, error) {
	st := time.Now()
	defer func() {
		log.Println("[*] Total render time:", time.Since(st))
	}()
	pages, err := getPagesDimensions(fileName, c)
	if err != nil {
		return nil, nil, err
	}

	log.Println("[!] Pages count:", len(pages))

	splitChannels := c.SplitChannels

	panicHandler := func(p interface{}) {
		fmt.Printf("[!] Task panicked: %v", p)
	}

	pool := pond.New(c.MaxCpuCount, len(pages), pond.MinWorkers(c.MaxCpuCount), pond.PanicHandler(panicHandler))

	var backupSpots = make(map[string][]int)

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
			var (
				localBackupSpots map[string][]int
				err              error
			)
			if splitChannels {
				outputFilepath := fmt.Sprintf("%s/%s.tiff", outputFolder, basename)
				if localBackupSpots, err = callGS(fileName, outputFilepath, page, "tiffsep"); err != nil {
					panic(err)
				}
				if _, err = callGS(fileName, outputFilepath, page, "tiff32nc"); err != nil {
					panic(err)
				}
			} else {
				outputFilepath := fmt.Sprintf("%s/%s.png", outputFolder, basename)
				if localBackupSpots, err = callGS(fileName, outputFilepath, page, "png16m"); err != nil {
					panic(err)
				}
			}
			for k, v := range localBackupSpots {
				if _, ok := backupSpots[k]; !ok {
					backupSpots[k] = v
				}
			}
		})
	}

	pool.StopAndWait()

	if pool.FailedTasks() > 0 {
		return nil, nil, errors.New("error on PDF rendering")
	}
	return pages, backupSpots, nil
}
