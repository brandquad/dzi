package main

import (
	"github.com/brandquad/dzi"
	"log"
)

func main() {
	filepath := "/Users/konstantinshishkin/Downloads/ME-1222.pdf"
	blocks, err := dzi.TextExtractor(filepath, 1)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(blocks)
}
