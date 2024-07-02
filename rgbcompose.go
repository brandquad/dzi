package dzi

import (
	"github.com/davidbyttow/govips/v2/vips"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
)

func convertString(input string) string {
	// Regular expression to match anything inside parentheses
	re := regexp.MustCompile(`\(([^)]+)\)`)

	// Replace the matched text with an empty string, effectively removing it
	result := re.ReplaceAllString(input, "")

	return result
}

func rgbCompose(entries []*entryInfo, channelsPath string) error {
	for _, entry := range entries {

		var (
			ref        *vips.ImageRef
			resultPath string
			err        error
		)

		for _, swatch := range entry.Swatches {
			var refSecond *vips.ImageRef

			if ref == nil {
				// Open first channel
				ref, err = vips.LoadImageFromFile(swatch.Filepath, nil)
				if err != nil {
					return err
				}
				resultPath = convertString(swatch.Filepath)
				log.Println(resultPath)
			} else {
				// Open other channels
				refSecond, err = vips.LoadImageFromFile(swatch.Filepath, nil)
				if err != nil {
					return err
				}
			}
			if refSecond != nil {
				if err = ref.Composite(refSecond, vips.BlendModeMultiply, 0, 0); err != nil {
					return err
				}
				refSecond.Close()
			}

		}

		if ref != nil {
			var buffer []byte

			if err = ref.TransformICCProfile(path.Join("icc", "sRGB Profile.icc")); err != nil {
				return err
			}

			//if err = ref.OptimizeICCProfile(); err != nil {
			//	return err
			//}

			buffer, _, err = ref.ExportJpeg(&vips.JpegExportParams{
				Quality: 90,
			})

			if err != nil {
				return err
			}
			ext := path.Ext(resultPath)
			resultPath = strings.ReplaceAll(resultPath, ext, ".jpeg")
			if err = os.WriteFile(resultPath, buffer, DefaultFolderPerm); err != nil {
				return err
			}

			entry.Swatches = append(entry.Swatches, &Swatch{
				Filepath: resultPath,
				Name:     "Color",
				Type:     "final",
			})

			ref.Close()
		}

	}

	return nil

}
