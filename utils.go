package dzi_processing

import (
	"errors"
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func syncToS3(assetId int, tmp string, c IncomeConfig) error {
	log.Println("Copy to S3:", c.S3Host, c.S3Bucket)
	var err error

	os.Setenv("MC_NO_COLOR", "1")
	aliasName := fmt.Sprintf("mediaquad%d", assetId)
	to := fmt.Sprintf("%s/%s/%d", aliasName, c.S3Bucket, assetId)
	from := fmt.Sprintf("%s/", tmp)

	log.Println("To:", to)
	log.Println("From:", from)
	log.Println("MC Alias:", aliasName)

	if minioOutput, err := execCmd("mc",
		"alias",
		"set",
		aliasName,
		c.S3Host,
		c.S3Key,
		c.S3Secret); err != nil {
		return err
	} else {
		log.Println(string(minioOutput))
	}

	if _, err = execCmd("mc", "cp", "-r", from, to, "--quiet"); err != nil {
		return err
	}

	defer func() {
		if aliasName != "" {
			log.Println("Remove alias:", aliasName)
			if _, err := execCmd("mc", "alias", "rm", aliasName); err != nil {
				log.Printf("Error removing mc alias: %v", err)
			}
		}
	}()
	log.Println("Done")
	return nil
}

func execCmd(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s: %w", output, err)
	}
	return output, nil
}

func extractField(s, f string) (string, bool) {
	if strings.HasPrefix(s, f) {
		return strings.TrimSpace(strings.TrimPrefix(s, f)), true
	}
	return "", false
}

func downloadFileTemporary(link string) (*os.File, error) {
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

func createImage(w, h int, c []float64) (*vips.ImageRef, error) {
	imageRef, err := vips.Black(w, h)
	if err != nil {
		return nil, err
	}
	err = imageRef.ToColorSpace(vips.InterpretationSRGB)
	if err != nil {
		return nil, err
	}

	err = imageRef.Linear([]float64{0, 0, 0}, c)
	if err != nil {
		return nil, err
	}

	return imageRef, nil

}
