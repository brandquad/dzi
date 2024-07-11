package dzi

import (
	"errors"
	"fmt"
	"github.com/alitto/pond"
	"log"
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

	Width  float64
	Height float64

	WidthPx  int
	HeightPx int

	Dpi    float64
	Rotate float64
}

func field(s, f string) string {
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, f) {
			return strings.TrimSpace(strings.TrimPrefix(l, f))
		}
	}
	return ""
}

func getPagesDimensions(fileName string, c Config) ([]*pageSize, error) {
	pages := make([]*pageSize, 0)

	args := []string{
		fileName,
		"dump_data",
	}

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

			rotate := outputLines[pageStartLine+1]
			rotate = strings.TrimSpace(field(rotate, "PageMediaRotation:"))
			dimensions := outputLines[pageStartLine+3]
			dimensions = strings.TrimSpace(field(dimensions, "PageMediaDimensions:"))
			dimensions = strings.ReplaceAll(dimensions, ",", "")
			rotateFloat, err := strconv.ParseFloat(strings.TrimSpace(rotate), 64)
			if err != nil {
				return nil, err
			}
			dimensionsPair := strings.Split(dimensions, " ")
			widthFloat, err := strconv.ParseFloat(strings.TrimSpace(dimensionsPair[0]), 64)
			heightFloat, err := strconv.ParseFloat(strings.TrimSpace(dimensionsPair[1]), 64)
			dpiForPage := c.DefaultDPI

			widthInches := widthFloat * pt2in
			heightInches := heightFloat * pt2in

			widthPx := widthInches * dpiForPage
			heightPx := heightInches * dpiForPage

			var recalc bool
			if widthPx > c.MaxSizePixels {
				dpiForPage = c.MaxSizePixels / widthInches
				recalc = true
			}
			if heightPx > c.MaxSizePixels {
				heightPx = c.MaxSizePixels / widthInches
				recalc = true
			}

			if recalc {
				widthPx = widthInches * dpiForPage
				heightPx = heightInches * dpiForPage
			}
			ps = &pageSize{
				PageNum:  i,
				WidthPt:  widthFloat,
				HeightPt: heightFloat,
				Width:    widthInches,
				Height:   heightInches,
				WidthPx:  int(widthPx),
				HeightPx: int(heightPx),
				Dpi:      dpiForPage,
				Rotate:   rotateFloat,
				Spots:    []string{"Cyan", "Magenta", "Yellow", "Black"},
			}
			break
		}

		if ps != nil {
			pages = append(pages, ps)
		}
	}

	// Extract spots
	args = []string{
		"-q",
		"-dNODISPLAY",
		"-dNOSAFER",
		fmt.Sprintf("-sFile=%s", fileName),
		"-dDumpFontsNeeded=false",
		"info.ps",
	}

	buff, err = execCmd("gs", args...)
	if err != nil {
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

func callGS(filename, output string, page *pageSize, device string) error {

	args := []string{
		"-q",
		"-dBATCH",
		"-dNOPAUSE",
		"-dSAFER",
		"-dSubsetFonts=true",
		"-dMaxBitmap=500000000",
		"-dAlignToPixels=0",
		"-dGridFitTT=2",
		"-dTextAlphaBits=4",
		"-dGraphicsAlphaBits=4",
		fmt.Sprintf("-dMaxSpots=%d", len(page.Spots)),
		fmt.Sprintf("-dFirstPage=%d", page.PageNum),
		fmt.Sprintf("-dLastPage=%d", page.PageNum),
		fmt.Sprintf("-r%d", int(page.Dpi)),
		fmt.Sprintf("-dDEVICEWIDTHPOINTS=%.02f", page.WidthPt),
		fmt.Sprintf("-dDEVIDEHEIGHTPOINTS=%.02f", page.HeightPt),
		fmt.Sprintf("-sOutputFile=%s", output),
		fmt.Sprintf("-sDEVICE=%s", device),
		filename,
	}

	_, err := execCmd("gs", args...)
	return err
}

func renderPdf(fileName, outputPrefix, basename string, c Config) error {
	st := time.Now()
	defer func() {
		log.Println("[*] Total render time:", time.Since(st))
	}()
	pages, err := getPagesDimensions(fileName, c)
	if err != nil {
		return err
	}

	log.Println("MaxCpuNum:", c.MaxCpuCount)
	log.Println("PagesCount:", len(pages))

	panicHandler := func(p interface{}) {
		fmt.Printf("Task panicked: %v", p)
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

			if c.SplitChannels {
				outputFilepath := fmt.Sprintf("%s/%s.tiff", outputFolder, basename)
				if err := callGS(fileName, outputFilepath, page, "tiffsep"); err != nil {
					panic(err)
				}
			}

			//outputFilepath := fmt.Sprintf("%s/%s.jpeg", outputFolder, basename)
			//if err := callGS(fileName, outputFilepath, page, "jpeg"); err != nil {
			//	panic(err)
			//}

		})
	}

	pool.StopAndWait()

	if pool.FailedTasks() > 0 {
		return errors.New("error on PDF rendering")
	}
	return nil
}
