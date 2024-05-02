package dzi

import (
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/lucasb-eyer/go-colorful"
	"log"
	"os"
	"path"
	"time"
)

func colorize(e *entryInfo, outputColorized, outputBw, leads1000, covers, iccProfile string, coverWidth int) error {
	var ref, mateRef *vips.ImageRef
	var err error

	defer func() {
		if ref != nil {
			ref.Close()
		}
		if mateRef != nil {
			mateRef.Close()
		}
	}()

	for _, entry := range e.Swatches {

		log.Println("Colorize:", entry.Name)

		ref, err = vips.LoadImageFromFile(entry.Filepath, nil)
		if err != nil {
			return err
		}

		if err = cp(entry.Filepath, path.Join(outputBw, entry.Basename())); err != nil {
			return err
		}

		if entry.NeedMate {

			rgbMateColor, err := colorful.Hex(entry.RBG)
			if err != nil {
				return err
			}

			if err = ref.ToColorSpace(vips.InterpretationSRGB); err != nil {
				return err
			}

			mateRef, err = createImage(ref.Width(), ref.Height(), rgbMateColor)
			if err != nil {
				return err
			}

			if err = mateRef.Composite(ref, vips.BlendModeScreen, 0, 0); err != nil {
				return err
			}
			if err = os.Remove(entry.Filepath); err != nil {
				return err
			}

			outputFilepath := path.Join(outputColorized, fmt.Sprintf("%s.png", entry.Filename()))
			if err = toPng(mateRef, outputFilepath); err != nil {
				return err
			}
			entry.Filepath = outputFilepath

			if ref, err = mateRef.Copy(); err != nil {
				return err
			}
			mateRef.Close()

		}

		log.Println("Covers for:", entry.Name)
		leads1000Path := path.Join(leads1000, fmt.Sprintf("%s.png", entry.Filename()))
		coverPath := path.Join(covers, fmt.Sprintf("%s.png", entry.Filename()))
		if ref.ColorSpace() == vips.InterpretationCMYK {
			if err = ref.TransformICCProfile(iccProfile); err != nil {
				return err
			}
			if err = ref.ToColorSpace(vips.InterpretationSRGB); err != nil {
				return err
			}
		}

		st := time.Now()

		if err = ref.Thumbnail(1000, 1000, vips.InterestingAll); err != nil {
			return err
		}

		log.Println("Covers 1000 for:", entry.Name, time.Since(st).Microseconds())

		if err = toPng(ref, leads1000Path); err != nil {
			return err
		}

		log.Println("toPNG 1000 for:", entry.Name, time.Since(st).Microseconds())

		if err = ref.Thumbnail(coverWidth, coverWidth, vips.InterestingAll); err != nil {
			return err
		}

		log.Println("Covers 200 for:", entry.Name, time.Since(st).Microseconds())

		if err = toPng(ref, coverPath); err != nil {
			return err
		}

		log.Println("toPNG 200 for:", entry.Name, time.Since(st).Microseconds())

	}

	return nil
}
