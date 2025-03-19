package assets

import (
	_ "embed"
	"encoding/json"
	dzi "github.com/brandquad/dzi/colorutils"
	"log"
	"strings"
)

type pantoneColor struct {
	Name          string    `json:"name"`
	LabComponents []float64 `json:"components"`
}
type pantoneData struct {
	Colors []pantoneColor `json:"colors"`
}

//go:embed pantones.json
var pantonesData []byte

var Pantones map[string][]int

func init() {
	var j pantoneData
	if err := json.Unmarshal(pantonesData, &j); err != nil {
		log.Fatal(err)
	}
	Pantones = make(map[string][]int)
	for _, color := range j.Colors {
		Pantones[strings.ToLower(color.Name)] = dzi.Lab2rgb(color.LabComponents)
	}
}
