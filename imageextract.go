package dzi

import (
	"errors"
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"os"
	"path"
	"strings"
)

func extractImage(filename, basename, output, iccPath string) (*entryInfo, error) {
	ref, err := vips.LoadImageFromFile(filename, nil)
	if err != nil {
		return nil, err
	}

	info := &entryInfo{
		Width:     float64(ref.Width()),
		Height:    float64(ref.Height()),
		Unit:      "px",
		ColorMode: ColorModeCMYK,
		Swatches:  make([]swatch, 0),
	}

	var refRGB *vips.ImageRef

	defer func() {
		if refRGB != nil {
			refRGB.Close()
		}
		ref.Close()
	}()

	switch ref.ColorSpace() {
	case vips.InterpretationSRGB, vips.InterpretationRGB, vips.InterpretationRGB16:
		if refRGB, err = ref.Copy(); err != nil {
			return nil, err
		}

		if err = refRGB.ToColorSpace(vips.InterpretationSRGB); err != nil {
			return nil, err
		}

		if err = ref.TransformICCProfile(iccPath); err != nil {
			return nil, err
		}
		if err = ref.ToColorSpace(vips.InterpretationCMYK); err != nil {
			return nil, err
		}
		break
	case vips.InterpretationCMYK:
		if refRGB, err = ref.Copy(); err != nil {
			return nil, err
		}
		if err = refRGB.ToColorSpace(vips.InterpretationSRGB); err != nil {
			return nil, err
		}

		break
	default:
		return nil, errors.New("unsupported color space")
	}
	rgbOutput := path.Join(output, fmt.Sprintf("%s.tiff", basename))
	if err = toTiff(refRGB, rgbOutput); err != nil {
		return nil, err
	}
	info.Swatches = append(info.Swatches, swatch{
		Filepath: rgbOutput,
		Name:     "Color",
		Type:     Final,
		NeedMate: false,
	})

	bands, err := ref.BandSplit()
	if err != nil {
		return nil, err
	}
	for idx, band := range bands {
		var swatchName string
		switch idx {
		case 0:
			swatchName = "Cyan"
		case 1:
			swatchName = "Magenta"
		case 2:
			swatchName = "Yellow"
		case 3:
			swatchName = "Black"
		case 4:
			swatchName = "Alpha"
		}

		if err = band.Invert(); err != nil {
			return nil, err
		}

		outputPath := path.Join(output, fmt.Sprintf("%s(%s).tiff", basename, swatchName))
		if err = toTiff(band, outputPath); err != nil {
			return nil, err
		}

		info.Swatches = append(info.Swatches, swatch{
			Filepath: outputPath,
			Name:     swatchName,
			RBG:      CMYK[strings.ToLower(swatchName)],
			Type:     CmykComponent,
			NeedMate: true,
		})
		band.Close()
	}

	return info, nil
}

func toTiff(ref *vips.ImageRef, output string) error {
	buffer, _, err := ref.ExportTiff(&vips.TiffExportParams{
		StripMetadata: true,
	})
	if err != nil {
		return err
	}

	return os.WriteFile(output, buffer, 0644)
}

func toPng(ref *vips.ImageRef, output string) error {
	buffer, _, err := ref.ExportPng(vips.NewPngExportParams())
	if err != nil {
		return err
	}

	return os.WriteFile(output, buffer, 0644)
}
