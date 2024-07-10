package main

import (
	"fmt"
	"github.com/alitto/pond"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const defaultDpi = 600.0
const maxSizePixels = 10000
const pt2in = 0.0138888889
const maxCpuNum = 4

func field(s, f string) string {
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, f) {
			return strings.TrimSpace(strings.TrimPrefix(l, f))
		}
	}
	return ""
}

func execCmd(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s: %w", output, err)
	}
	return output, nil
}

type pageSize struct {
	PageNum int

	WidthPt  float64
	HeightPt float64

	Width  float64
	Height float64

	WidthPx  int
	HeightPx int

	Dpi    float64
	Rotate float64
}

func getPagesDimensions(fileName string) ([]pageSize, error) {
	pages := make([]pageSize, 0)

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
			dpiForPage := defaultDpi

			widthInches := widthFloat * pt2in
			heightInches := heightFloat * pt2in

			widthPx := widthInches * dpiForPage
			heightPx := heightInches * dpiForPage

			var recalc bool
			if widthPx > maxSizePixels {
				dpiForPage = maxSizePixels / widthInches
				recalc = true
			}
			if heightPx > maxSizePixels {
				heightPx = maxSizePixels / widthInches
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
			}
			break
		}

		if ps != nil {
			pages = append(pages, *ps)
		}
	}

	return pages, nil
}

func process(fileName string) ([]pageSize, error) {
	pages, err := getPagesDimensions(fileName)
	if err != nil {
		return nil, err
	}

	return pages, nil
}

func main() {
	const pathToFiles = "/Users/konstantinshishkin/Downloads/test/59.pdf"
	pages, err := process(pathToFiles)
	if err != nil {
		log.Fatal(err)
	}

	outputPath := "_tmp/sss/"
	if _, err := os.ReadDir(outputPath); err == nil {
		os.RemoveAll(outputPath)
	}

	os.MkdirAll(outputPath, 0777)

	pool := pond.New(maxCpuNum, len(pages), pond.MinWorkers(maxCpuNum))
	log.Println("MaxCpuNum:", maxCpuNum)
	log.Println("PagesCount:", len(pages))

	for _, page := range pages {
		pool.Submit(func() {
			log.Printf("Running render page number #%d", page.PageNum)

			outputFilepath := fmt.Sprintf("%s/%d.tiff", outputPath, page.PageNum)
			if err := gs(pathToFiles, outputFilepath, page.PageNum, int(page.Dpi), true); err != nil {
				log.Fatal(err)
			}
		})
	}

	pool.StopAndWait()
}

func gs(filename, output string, page, resolution int, splitChannels bool) error {
	var device = "tiffsep"
	if !splitChannels {
		device = "jpeg"
	}

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
		"-dMaxSpots=59",
		fmt.Sprintf("-dFirstPage=%d", page),
		fmt.Sprintf("-dLastPage=%d", page),
		fmt.Sprintf("-r%d", resolution),
		fmt.Sprintf("-sOutputFile=%s", output),
		fmt.Sprintf("-sDEVICE=%s", device),
		filename,
	}

	_, err := execCmd("gs", args...)
	return err
}
