package dzi

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/davidbyttow/govips/v2/vips"
)

func extractImage(filename, basename, _output string, c *Config) ([]*pageInfo, error) {
	pages := make([]*pageInfo, 1)

	ref, err := vips.LoadImageFromFile(filename, nil)
	if err != nil {
		return nil, err
	}

	var colorModel ColorMode

	switch ref.ColorSpace() {
	case vips.InterpretationSRGB, vips.InterpretationRGB, vips.InterpretationRGB16:
		colorModel = ColorModeRBG
		// Disable split channels for RGB images
		//c.SplitChannels = false
		//if err = ref.ToColorSpace(vips.InterpretationSRGB); err != nil {
		//	return nil, err
		//}
		break
	case vips.InterpretationCMYK:
		colorModel = ColorModeCMYK
		break
	default:
		return nil, errors.New("unsupported color space")
	}

	//log.Println(colorModel)

	info := &pageInfo{
		Prefix:     "page_1",
		PageNumber: 1,
		Width:      float64(ref.Width()),
		Height:     float64(ref.Height()),
		Unit:       "px",
		ColorMode:  colorModel,
		Swatches:   make([]*Swatch, 0),
		Dpi:        int(c.DefaultDPI),
	}
	output := path.Join(_output, info.Prefix)

	if err = os.MkdirAll(output, DefaultFolderPerm); err != nil {
		return nil, err
	}

	refRGB, err := ref.Copy()
	if err != nil {

	}
	defer func() {
		if refRGB != nil {
			refRGB.Close()
		}
		ref.Close()
	}()

	if colorModel == ColorModeCMYK {
		if err = refRGB.ToColorSpace(vips.InterpretationSRGB); err != nil {
			return nil, err
		}
	}

	//switch colorModel {
	//case ColorModeRBG:
	//	//if err = ref.TransformICCProfile(c.ICCProfileFilepath); err != nil {
	//	//	return nil, err
	//	//}
	//	//if err = ref.ToColorSpace(vips.InterpretationCMYK); err != nil {
	//	//	return nil, err
	//	//}
	//	break
	//case ColorModeCMYK:
	//	//	if refRGB, err = ref.Copy(); err != nil {
	//	//		return nil, err
	//	//	}
	//	//	if err = refRGB.ToColorSpace(vips.InterpretationSRGB); err != nil {
	//	//		return nil, err
	//	//	}
	//	break
	//	//default:
	//	//	return nil, errors.New("unsupported color space")
	//}

	//log.Println(refRGB.ColorSpace())

	rgbOutput := path.Join(output, fmt.Sprintf("%s.tiff", basename))

	if err = toTiff(refRGB, rgbOutput); err != nil {
		return nil, err
	}

	info.Swatches = append(info.Swatches, &Swatch{
		Filepath: rgbOutput,
		Name:     "Color",
		OpsName:  "Color",
		Type:     Final,
		NeedMate: false,
	})

	if c.SplitChannels {
		bands, err := ref.BandSplit()
		if err != nil {
			return nil, err
		}
		for idx, band := range bands {
			var swatchName string

			if colorModel == ColorModeCMYK {

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
			}
			if colorModel == ColorModeRBG {
				switch idx {
				case 0:
					swatchName = "Red"
				case 1:
					swatchName = "Green"
				case 2:
					swatchName = "Blue"
				case 3:
					swatchName = "Alpha"
				}
			}

			if err = band.Invert(); err != nil {
				return nil, err
			}

			outputPath := path.Join(output, fmt.Sprintf("%s(%s).tiff", basename, swatchName))
			if err = toTiff(band, outputPath); err != nil {
				return nil, err
			}

			info.Swatches = append(info.Swatches, &Swatch{
				Filepath: outputPath,
				Name:     swatchName,
				OpsName:  swatchName,
				RBG:      CMYK[strings.ToLower(swatchName)],
				Type:     CmykComponent,
				NeedMate: true,
			})
			band.Close()
		}
	}

	pages[0] = info
	return pages, nil
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
