package dzi

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/brandquad/dzi/assets"
	dzi "github.com/brandquad/dzi/colorutils"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/lucasb-eyer/go-colorful"
	"golang.org/x/text/encoding/charmap"
)

var defaultDecoder = charmap.Windows1251.NewDecoder()
var re = regexp.MustCompile(`\"(?P<name>.*)\" .* ink = (?P<cmyk>.*) CMYK`)

func syncToS3(assetId int, tmp string, c *Config) error {
	st := time.Now()
	log.Println("[>] Copy to S3:", c.S3Host, c.S3Bucket)

	var err error

	if err = os.Setenv("MC_NO_COLOR", "1"); err != nil {
		return err
	}

	aliasName := fmt.Sprintf("mediaquad%d", assetId)
	to := fmt.Sprintf("%s/%s/%d", aliasName, c.S3Bucket, assetId)
	from := fmt.Sprintf("%s/", tmp)

	// Copy to s3 through mc commands

	// Set alias
	if _, err := execCmd("mc",
		"alias",
		"set",
		aliasName,
		c.S3Host,
		c.S3Key,
		c.S3Secret); err != nil {
		return err
	}

	// Call mc cp command
	if _, err = execCmd("mc", "cp", "-r", from, to, "--quiet"); err != nil {
		return err
	}

	defer func() {

		if aliasName != "" {
			// Remove alias
			if _, err := execCmd("mc", "alias", "rm", aliasName); err != nil {
				log.Printf("[!] Error removing mc alias: %v", err)
			}
		}

		log.Printf("[<] Copy to S3, at %s", time.Since(st))
	}()

	return nil
}

func cp(from, to string) error {
	dir, _ := path.Split(to)
	if err := os.MkdirAll(dir, DefaultFolderPerm); err != nil {
		return err
	}
	_, err := execCmd("cp", from, to)
	return err
}

func execCmd(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s: %w", output, err)
	}
	return output, nil
}

// field - provides functionality to extract specific fields from a string.
func field(s, f string) string {
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, f) {
			return strings.TrimSpace(strings.TrimPrefix(l, f))
		}
	}
	return ""
}

// downloadFileTemporary get url to file and return file object after downloading
func downloadFileTemporary(link string) (*os.File, error) {
	st := time.Now()
	log.Println("[>] Downloading file temporary")
	defer func() {
		log.Printf("[<] Downloading %s in %s...", link, time.Since(st))
	}()

	p := strings.Split(link, ".")

	resp, err := http.Get(link)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	defer func() {
		resp.Body.Close()
	}()

	file, err := os.CreateTemp("", fmt.Sprintf("tmpfile-*.%s", p[len(p)-1]))
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buf)
		if err != nil {
			if err == io.EOF {
				file.Write(buf[:n])
				break
			}
			return nil, err
		}
		if n > 0 {
			_, err = file.Write(buf[:n])
			if err != nil {
				file.Close()
				os.Remove(file.Name())
				return nil, err
			}
		}
	}
	return file, file.Sync()
}

// createImage return empty vips image with a certain width, height and background color
func createImage(w, h int, c colorful.Color) (*vips.ImageRef, error) {

	var cR, cG, cB uint8 = c.RGB255()
	color := []float64{float64(cR), float64(cG), float64(cB)}

	imageRef, err := vips.Black(w, h)
	if err != nil {
		return nil, err
	}
	err = imageRef.ToColorSpace(vips.InterpretationSRGB)
	if err != nil {
		return nil, err
	}

	err = imageRef.Linear([]float64{0, 0, 0}, color)
	if err != nil {
		return nil, err
	}

	return imageRef, nil
}

type channelsMap map[string]*channelFile
type pageChannels map[int]channelsMap

type channelFile struct {
	RgbComponents string
	Filepath      string
	OpsName       string
	IsColor       bool
}

// callGS just run ghostscript
func callGS(filename, output string, page *pageSize, device string, c *Config) (channelsMap, error) {
	log.Printf("[!] Effective DPI for page %d is %d, dOverprint is %s, device is %s", page.PageNum, page.Dpi, c.Overprint, device)
	var (
		overprint string
		dUsePDFX3 string = "-dUsePDFX3Profile=0"
	)
	if c.Overprint != "" {
		overprint = fmt.Sprintf("-dOverprint=%s", c.Overprint)
	}
	if c.UsePDFX3 {
		dUsePDFX3 = "-dUsePDFX3Profile=1"
	}

	maxBitmap := "-dMaxBitmap=500000000"
	maxSpots := ""
	if page.Spots != nil {
		maxSpots = fmt.Sprintf("-dMaxSpots=%d", len(page.Spots))
	}

	printSpotCmyk := "-dPrintSpotCMYK"

	if device == "tiff32nc" {
		maxBitmap = ""
		maxSpots = ""
		printSpotCmyk = ""
	}

	args := []string{
		"-q",
		"-dBATCH",
		"-dNOPAUSE",
		maxBitmap,
		"-dSAFER",
		"-dSubsetFonts=true",
		printSpotCmyk,
		"-dAlignToPixels=1",
		"-dGridFitTT=0",
		"-dTextAlphaBits=4",
		dUsePDFX3,
		fmt.Sprintf("-dGraphicsAlphaBits=%d", c.GraphicsAlphaBits),
		overprint,
		maxSpots,
		fmt.Sprintf("-dFirstPage=%d", page.PageNum),
		fmt.Sprintf("-dLastPage=%d", page.PageNum),
		fmt.Sprintf("-r%d", page.Dpi),
		fmt.Sprintf("-sOutputFile=%s", output),
		fmt.Sprintf("-sDEVICE=%s", device),
		filename,
	}

	args = slices.DeleteFunc(args, func(x string) bool {
		return len(x) == 0
	})

	cmdOut, err := execCmd("gs", args...)
	if err != nil {
		return nil, err
	}

	var spots = make(channelsMap)

	if device == "tiff32nc" {
		return spots, nil
	}

	files, err := os.ReadDir(path.Dir(output))
	if err != nil {
		return nil, err
	}
	baseName := path.Base(output)
	cleanBaseName := strings.TrimSuffix(baseName, path.Ext(baseName))

	// Fix ME-83 and ME-85
	for _, file := range files {
		spotFile := &channelFile{}

		if file.Name() == baseName {
			spotFile.OpsName = "Color"
			spotFile.IsColor = true
			spotFile.Filepath = filepath.Join(path.Dir(output), file.Name())
			spots["Color"] = spotFile
			continue
		}

		fileExt := path.Ext(file.Name())
		spotName := strings.TrimSuffix(file.Name(), fileExt)
		spotName = strings.TrimPrefix(spotName, cleanBaseName)
		spotName = strings.TrimPrefix(spotName, "(")
		spotName = strings.TrimSuffix(spotName, ")")

		if strings.Contains(spotName, "%") {
			// Need decode
			spotNameUnescape, _ := url.QueryUnescape(spotName)
			if len(spotNameUnescape) > 0 {
				if !utf8.Valid([]byte(spotNameUnescape)) {
					out, _ := defaultDecoder.Bytes([]byte(spotNameUnescape))
					spotName = string(out)
				} else {
					spotName = spotNameUnescape
				}
			} else {
				spotName = spotNameUnescape
			}

			spotNameForFile := spotName
			if strings.Contains(spotNameForFile, "/") {
				spotNameForFile = strings.Replace(spotNameForFile, "/", "-", -1)
			}

			// Restore file
			fileName := path.Join(path.Dir(output), fmt.Sprintf("%s(%s)%s", cleanBaseName, spotNameForFile, fileExt))
			oldFileName := path.Join(path.Dir(output), file.Name())
			if _, err := execCmd("mv", oldFileName, fileName); err != nil {
				return nil, err
			}
			spotFile.OpsName = spotNameForFile
			spotFile.Filepath = fileName
		} else {
			spotFile.Filepath = path.Join(path.Dir(output), file.Name())
			spotFile.OpsName = spotName
		}
		if v, ok := CMYK[strings.ToLower(spotName)]; ok {
			spotFile.RgbComponents = v
		}

		spots[spotName] = spotFile
	}

	for _, line := range strings.Split(string(cmdOut), "\n") {
		if strings.HasPrefix(line, "%%SeparationColor:") {

			paramsMap := make(map[string]string)
			line = strings.TrimPrefix(line, "%%SeparationColor: ")

			match := re.FindStringSubmatch(line)
			for i, name := range re.SubexpNames() {
				if i > 0 && i <= len(match) {
					paramsMap[name] = match[i]
				}
			}

			spotName := paramsMap["name"]
			spotNameBytes := []byte(spotName)

			if !utf8.Valid(spotNameBytes) {
				o, e := defaultDecoder.Bytes([]byte(spotName))
				if e == nil {
					spotName = string((o))
				}
			}

			if v, ok := assets.Pantones[strings.ToLower(spotName)]; ok {
				spots[spotName].RgbComponents = rgb2hex(v)
				continue
			}

			components := strings.Split(paramsMap["cmyk"], " ")

			_c, _ := strconv.ParseFloat(components[0], 64)
			_m, _ := strconv.ParseFloat(components[1], 64)
			_y, _ := strconv.ParseFloat(components[2], 64)
			_k, _ := strconv.ParseFloat(components[3], 64)

			_c = _c * 100.0 / 32760.0
			_m = _m * 100.0 / 32760.0
			_y = _y * 100.0 / 32760.0
			_k = _k * 100.0 / 32760.0

			spots[spotName].RgbComponents = rgb2hex(dzi.Cmyk2rgb([]float64{_c, _m, _y, _k}))
		}
	}
	return spots, err
}

func rgb2hex(rgb []int) string {
	return fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2])
}

// esko2swatch convert Esko metadata to Swatch
func esko2swatch(name, egname, egtype, book string, nr, ng, nb float64) Swatch {
	var swatchName = name
	if egtype == "pantone" {
		switch book {
		case "pms1000c":
			swatchName = fmt.Sprintf("PANTONE %s C", egname)
		case "pms1000u":
			swatchName = fmt.Sprintf("PANTONE %s U", egname)
		case "pms1000m":
			swatchName = fmt.Sprintf("PANTONE %s M", egname)
		case "goec":
			swatchName = fmt.Sprintf("PANTONE %s C", egname)
		case "goeu":
			swatchName = fmt.Sprintf("PANTONE %s U", egname)
		case "pmetc":
			swatchName = fmt.Sprintf("PANTONE %s C", egname)
		case "ppasc":
			swatchName = fmt.Sprintf("PANTONE %s C", egname)
		case "ppasu":
			swatchName = fmt.Sprintf("PANTONE %s U", egname)

		default:
			swatchName = name
		}
	}

	var swatchType SwatchType
	switch egtype {
	case "process":
		swatchType = CmykComponent
	case "pantone", "designer":
		swatchType = SpotComponent
	}

	var R = 255 * nr / 1
	var G = 255 * ng / 1
	var B = 255 * nb / 1

	return Swatch{
		Filepath: "",
		Name:     swatchName,
		RBG:      fmt.Sprintf("#%02x%02x%02x", int(math.Round(R)), int(math.Round(G)), int(math.Round(B))),
		Type:     swatchType,
		NeedMate: true,
	}
}
