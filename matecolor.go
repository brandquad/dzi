package dzi

import (
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/lucasb-eyer/go-colorful"
	"log"
	"os"
	"path"
	"strconv"
)

func colorize(e []*entryInfo, _outputColorized, _outputBw, _leads1000, _covers, iccProfile string, coverWidth int) error {
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

	for _, _e := range e {

		outputColorized := path.Join(_outputColorized, _e.Prefix)
		outputBw := path.Join(_outputBw, _e.Prefix)
		leads1000 := path.Join(_leads1000, _e.Prefix)
		covers := path.Join(_covers, _e.Prefix)

		if err = os.MkdirAll(outputColorized, DefaultFolderPerm); err != nil {
			return err
		}
		if err = os.MkdirAll(outputBw, DefaultFolderPerm); err != nil {
			return err
		}
		if err = os.MkdirAll(leads1000, DefaultFolderPerm); err != nil {
			return err
		}
		if err = os.MkdirAll(covers, DefaultFolderPerm); err != nil {
			return err
		}

		for _, entry := range _e.Swatches {

			log.Printf("Colorize channel %s at page %d", entry.Name, _e.PageNumber)

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

			} else {
				if err = os.Remove(path.Join(outputBw, entry.Basename())); err != nil {
					return err
				}
			}

			leads1000Path := path.Join(leads1000, fmt.Sprintf("%s.png", entry.Filename()))
			coverPath := path.Join(covers, fmt.Sprintf("%s.png", entry.Filename()))

			if _, err = execCmd("vips", "thumbnail", entry.Filepath, leads1000Path, "1000"); err != nil {
				return err
			}
			if _, err = execCmd("vips", "thumbnail", entry.Filepath, coverPath, strconv.Itoa(coverWidth)); err != nil {
				return err
			}
		}
	}

	return nil
}
