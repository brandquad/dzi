package dzi

import (
	"fmt"
	"log"
	"regexp"
)

func extractOutputPath(log string) (string, error) {
	re := regexp.MustCompile(`->\s+(.+?\.(pdf|PDF))\s`)
	matches := re.FindStringSubmatch(log)
	if len(matches) < 2 {
		return "", fmt.Errorf("не удалось найти путь к PDF в строке")
	}
	return matches[1], nil
}

func convertPPTX(originalFilepath, basename string, c *Config) (string, error) {

	log.Println("Processing as PPTX file")

	args := []string{
		"--headless",
		"--convert-to",
		"pdf",
		originalFilepath,
	}

	result, err := execCmd(c.LibreOfficePath, args...)
	if err != nil {
		return "", err
	}

	return extractOutputPath(string(result))
}
