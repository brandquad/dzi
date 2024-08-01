package dzi

import (
	"errors"
	"fmt"
	"github.com/alitto/pond"
	"time"

	//"github.com/alitto/pond"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/lucasb-eyer/go-colorful"
	"log"
	"os"
	"path"
)

func processSwatch(page *pageInfo, swatch *Swatch, colorizedFolder, bwFolder string) error {
	st := time.Now()

	log.Printf("[>] Colorize %s page %d", swatch.Name, page.PageNumber)
	var mateRef *vips.ImageRef
	ref, err := vips.LoadImageFromFile(swatch.Filepath, nil)
	if err != nil {
		return err
	}

	defer func() {
		log.Printf("[<] Colorize %s page %d, at %s", swatch.Name, page.PageNumber, time.Since(st))
		if mateRef != nil {
			mateRef.Close()
		}
		if ref != nil {
			ref.Close()
		}
	}()

	// Make copy to B-W channels folder
	if err = cp(swatch.Filepath, path.Join(bwFolder, swatch.Basename())); err != nil {
		return err
	}
	//log.Printf("Make copy %s", time.Since(st))

	if swatch.NeedMate {

		rgbMateColor, err := colorful.Hex(swatch.RBG)
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

		if err = os.Remove(swatch.Filepath); err != nil {
			return err
		}

		outputFilepath := path.Join(colorizedFolder, fmt.Sprintf("%s.png", swatch.Filename()))
		if err = toPng(mateRef, outputFilepath); err != nil {
			return err
		}

		swatch.Filepath = outputFilepath

		if ref, err = mateRef.Copy(); err != nil {
			return err
		}
		mateRef.Close()

	} else {
		if err = os.Remove(path.Join(bwFolder, swatch.Basename())); err != nil {
			return err
		}
		log.Printf("[-] No mate for %s, skipped", swatch.Filepath)
	}

	return nil
}

func prepareFolders(page *pageInfo, folderPrefix ...string) ([]string, error) {
	folders := make([]string, len(folderPrefix))
	for idx, prefix := range folderPrefix {
		folderPath := path.Join(prefix, page.Prefix)
		if err := os.MkdirAll(folderPath, DefaultFolderPerm); err != nil {
			return nil, err
		}
		folders[idx] = folderPath
	}
	return folders, nil
}

func colorize(pages []*pageInfo, _outputColorized, _outputBw, _leads1000, _covers string, c Config) error {
	st := time.Now()
	var ref, mateRef *vips.ImageRef
	//var err error

	defer func() {
		log.Println("[*] Total colorize time: ", time.Since(st))
		if ref != nil {
			ref.Close()
		}
		if mateRef != nil {
			mateRef.Close()
		}
	}()

	panicHandler := func(p interface{}) {
		fmt.Printf("[!] Task panicked: %v", p)
	}
	//pool := pond.New(c.MaxCpuCount, 1000, pond.MinWorkers(c.MaxCpuCount), pond.PanicHandler(panicHandler))
	pool := pond.New(1, 1000, pond.MinWorkers(1), pond.PanicHandler(panicHandler))

	for _, page := range pages {

		// Output paths
		folders, err := prepareFolders(page, _outputColorized, _outputBw, _leads1000, _covers)
		if err != nil {
			return err
		}
		colorizedFolder, bwFolder := folders[0], folders[1]

		for _, swatch := range page.Swatches {
			pool.Submit(func() {
				if err := processSwatch(page, swatch, colorizedFolder, bwFolder); err != nil {
					panic(err)
				}
			})

		}
	}

	pool.StopAndWait()
	if pool.FailedTasks() > 0 {
		return errors.New("error on colorize")
	}

	return nil
}
