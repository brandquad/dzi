package main

import (
	"flag"
	"github.com/brandquad/dzi"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/kelseyhightower/envconfig"
	"log"
	"os"
	"strconv"
)

type Config struct {
	S3Host             string `envconfig:"DZI_S3_HOST" required:"true"`
	S3Key              string `envconfig:"DZI_S3_KEY" required:"true"`
	S3Secret           string `envconfig:"DZI_S3_SECRET" required:"true"`
	S3Bucket           string `envconfig:"DZI_BUCKET" required:"true" default:"dzi"`
	TileSize           string `envconfig:"DZI_TILE_SIZE" default:"255"`
	Overlap            string `envconfig:"DZI_OVERLAP" default:"1"`
	Resolution         string `envconfig:"DZI_RESOLUTION" default:"600"`
	CoverWidth         string `envconfig:"DZI_COVER_W" default:"200"`
	DebugMode          bool   `envconfig:"DZI_DEBUG" default:"false"`
	HookUrl            string `envconfig:"HOOK_URL"`
	ICCProfileFilepath string
}

func (c Config) MakeDziConfig() dzi.Config {
	return dzi.Config{
		S3Host:             c.S3Host,
		S3Key:              c.S3Key,
		S3Secret:           c.S3Secret,
		S3Bucket:           c.S3Bucket,
		TileSize:           c.TileSize,
		Overlap:            c.Overlap,
		Resolution:         c.Resolution,
		CoverWidth:         c.CoverWidth,
		ICCProfileFilepath: c.ICCProfileFilepath,
		DebugMode:          false,
	}
}

func main() {
	vips.LoggingSettings(func(messageDomain string, verbosity vips.LogLevel, message string) {}, vips.LogLevelInfo)
	vips.Startup(nil)
	defer vips.Shutdown()

	var c Config
	if err := envconfig.Process("", &c); err != nil {
		log.Fatalln(err)
	}
	c.DebugMode = true
	c.ICCProfileFilepath = "./icc/CoatedGRACoL2006.icc"

	flag.Parse()

	if flag.NArg() < 2 {
		log.Fatalln("URL and AssetId are required as arguments")
	}

	url := flag.Arg(0)
	assetId, err := strconv.Atoi(flag.Arg(1))

	if err != nil {
		log.Fatalf("Failed to convert AssetId to integer: %v\n", err)
	}
	config := c.MakeDziConfig()
	_, config.DebugMode = os.LookupEnv("DEBUG")
	manifest, err := dzi.Processing(url, assetId, config)

	if err != nil {
		log.Fatalln(err)
	}

	log.Println(manifest)
}
