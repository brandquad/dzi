package dzi

import (
	"errors"
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/lucasb-eyer/go-colorful"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

// cmyk2rgb converts a CMYK color value to RGB
func cmyk2rgb(cmyk []float64) []int {
	var r, g, b float64
	r = 255.0 * (1 - cmyk[0]/100) * (1 - cmyk[3]/100)
	g = 255.0 * (1 - cmyk[1]/100) * (1 - cmyk[3]/100)
	b = 255.0 * (1 - cmyk[2]/100) * (1 - cmyk[3]/100)
	return []int{int(math.Ceil(r)), int(math.Ceil(g)), int(math.Ceil(b))}
}

// lab2rgb converts a LAB color value to RGB
func lab2rgb(lab []float64) []int {
	var y float64 = (lab[0] + 16) / 116
	var x float64 = lab[1]/500 + y
	var z float64 = y - lab[2]/200
	var r, g, b float64

	if x*x*x > 0.008856 {
		x = 0.95047 * x * x * x
	} else {
		x = 0.95047 * ((x - 16/116) / 7.787)
	}
	if y*y*y > 0.008856 {
		y = 1.00000 * (y * y * y)
	} else {
		y = 1.00000 * ((y - 16/116) / 7.787)
	}
	if z*z*z > 0.008856 {
		z = 1.08883 * (z * z * z)
	} else {
		z = 1.08883 * ((z - 16/116) / 7.787)
	}

	r = x*3.2406 + y*-1.5372 + z*-0.4986
	g = x*-0.9689 + y*1.8758 + z*0.0415
	b = x*0.0557 + y*-0.2040 + z*1.0570

	if r > 0.0031308 {
		r = 1.055*math.Pow(r, 1/2.4) - 0.055
	} else {
		r = 12.92 * r
	}

	if g > 0.0031308 {
		g = 1.055*math.Pow(g, 1/2.4) - 0.055
	} else {
		g = 12.92 * g
	}
	if b > 0.0031308 {
		b = 1.055*math.Pow(b, 1/2.4) - 0.055
	} else {
		b = 12.92 * b
	}

	return []int{
		int(math.Ceil(math.Max(0, math.Min(1, r)) * 255)),
		int(math.Ceil(math.Max(0, math.Min(1, g)) * 255)),
		int(math.Ceil(math.Max(0, math.Min(1, b)) * 255)),
	}
}

func syncToS3(assetId int, tmp string, c Config) error {
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

// callGS just run ghostscript
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
		fmt.Sprintf("-r%d", page.Dpi),
		fmt.Sprintf("-dDEVICEWIDTHPOINTS=%.02f", page.WidthPt),
		fmt.Sprintf("-dDEVIDEHEIGHTPOINTS=%.02f", page.HeightPt),
		fmt.Sprintf("-sOutputFile=%s", output),
		fmt.Sprintf("-sDEVICE=%s", device),
		filename,
	}

	_, err := execCmd("gs", args...)
	return err
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
		RBG:      fmt.Sprintf("#%02x%02x%02x", int(R), int(G), int(B)),
		Type:     swatchType,
		NeedMate: true,
	}
}
