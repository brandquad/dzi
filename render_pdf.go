package dzi

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alitto/pond"
)

var lineRe = regexp.MustCompile(`(?m)^(\d+).*\[(.*)\]`)

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

	// get read mediabox dimensions
	args = []string{
		"info", "-M", fileName, "1-9999",
	}
	buff, err = execCmd("mutool", args...)
	if err != nil {
		return nil, err
	}

	realDimensionsMap := make(map[int]muBox)

	for _, line := range strings.Split(string(buff), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		match := lineRe.FindAllStringSubmatch(line, -1)
		if match != nil {
			pageNumStr := match[0][1]
			pageNum, _ := strconv.Atoi(pageNumStr)
			dims := strings.Split(strings.TrimSpace(match[0][2]), " ")

			var r, l, t, b float64
			l, _ = strconv.ParseFloat(dims[0], 64)
			b, _ = strconv.ParseFloat(dims[1], 64)
			r, _ = strconv.ParseFloat(dims[2], 64)
			t, _ = strconv.ParseFloat(dims[3], 64)
			realDimensionsMap[pageNum] = muBox{
				B: b,
				L: l,
				R: r,
				T: t,
			}
		}
	}

	if len(realDimensionsMap) == 0 {
		// No results from mutool
		// Try to get dimensions from pdfinfo

		args = []string{
			fileName,
		}
		buff, err = execCmd("pdfinfo", args...)
		if err != nil {
			return nil, err
		}

		//re2 := regexp.MustCompile(`(\d+\.\d+)\s+x\s+(\d+\.\d+)`)
		re3 := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*x\s*(\d+(?:\.\d+)?)(?:\s*pts)?`)

		for _, line := range strings.Split(string(buff), "\n") {
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			if !strings.HasPrefix(line, "Page size:") {
				continue
			}

			matches := re3.FindStringSubmatch(line)

			if len(matches) < 3 {
				continue
			}

			//matches := re2.FindStringSubmatch(line)
			//
			//if len(matches) < 3 {
			//	continue
			//}

			pWidth, err1 := strconv.ParseFloat(matches[1], 64)
			pHeight, err2 := strconv.ParseFloat(matches[2], 64)
			if err1 != nil || err2 != nil {
				continue
			}

			for _, p := range mudoc.Pages {
				realDimensionsMap[p.PageNum] = muBox{
					B: pHeight,
					L: pWidth,
					R: 0.0,
					T: 0.0,
				}
			}

			break
		}

	}

	for idx, p := range mudoc.Pages {

		var ps = &pageSize{
			PageNum:  p.PageNum,
			WidthPt:  math.Abs(realDimensionsMap[p.PageNum].R - realDimensionsMap[p.PageNum].L),
			HeightPt: math.Abs(realDimensionsMap[p.PageNum].T - realDimensionsMap[p.PageNum].B),
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

	// Remove lines not starting with "Page "
	outputLines := strings.Split(string(buff), "\n")
	filteredLines := make([]string, 0)
	for _, line := range outputLines {
		if strings.HasPrefix(line, "Page ") {
			filteredLines = append(filteredLines, line)
		}
	}

	// Map spots to pages
	for _, page := range pages {
		for _, line := range filteredLines {
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

func renderPdf(fileName, outputPrefix, basename string, c *Config) ([]*pageSize, pageChannels, error) {

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

	var backupSpots = make(pageChannels)
	backupSpotsMutex := &sync.Mutex{}

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
			var spots channelsMap

			if splitChannels {
				outputFilepath := fmt.Sprintf("%s/%s.tiff", outputFolder, basename)
				if spots, err = callGS(fileName, outputFilepath, page, "tiffsep", c); err != nil {
					panic(err)
				}
				var overprint = c.Overprint
				if c.Overprint != "/simulate" {
					c.Overprint = ""
				}
				if _, err = callGS(fileName, outputFilepath, page, "tiff32nc", c); err != nil {
					panic(err)
				}
				c.Overprint = overprint
			} else {
				outputFilepath := fmt.Sprintf("%s/%s.png", outputFolder, basename)
				if spots, err = callGS(fileName, outputFilepath, page, "png16m", c); err != nil {
					panic(err)
				}
			}
			backupSpotsMutex.Lock()
			backupSpots[page.PageNum] = spots
			backupSpotsMutex.Unlock()

		})
	}

	pool.StopAndWait()

	if pool.FailedTasks() > 0 {
		return nil, nil, errors.New("error on PDF rendering")
	}
	return pages, backupSpots, nil
}
