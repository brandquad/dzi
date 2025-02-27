package main

import (
	"flag"
	"github.com/brandquad/dzi"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/kelseyhightower/envconfig"
	"log"
	"os"
	"slices"
	"strconv"
)

type Config struct {
	S3Host             string  `envconfig:"DZI_S3_HOST" required:"true"`
	S3Key              string  `envconfig:"DZI_S3_KEY" required:"true"`
	S3Secret           string  `envconfig:"DZI_S3_SECRET" required:"true"`
	S3Bucket           string  `envconfig:"DZI_BUCKET" required:"true" default:"dzi"`
	TileSize           string  `envconfig:"DZI_TILE_SIZE" default:"1024"`
	Overlap            string  `envconfig:"DZI_OVERLAP" default:"1"`
	Resolution         int     `envconfig:"DZI_RESOLUTION" default:"600"`
	MinResolution      int     `envconfig:"DZI_MIN_RESOLUTION" default:"200"`
	MaxResolution      int     `envconfig:"DZI_MAX_RESOLUTION" default:"1600"`
	CoverHeight        string  `envconfig:"DZI_COVER_H" default:"300"`
	DebugMode          bool    `envconfig:"DZI_DEBUG" default:"false"`
	SplitChannels      bool    `envconfig:"DZI_SPLIT_CHANNELS" default:"true"`
	Overprint          string  `envconfig:"DZI_OVERPRINT" default:"/enable"`
	HookUrl            string  `envconfig:"HOOK_URL"`
	CopyChannelsToS3   bool    `envconfig:"DZI_COPY_CHANNELS" default:"false"`
	MaxCpuCount        int     `envconfig:"MAX_CPU_COUNT" default:"4"`
	MaxSizePixels      float64 `envconfig:"MAX_SIZE_PIXELS" default:"15000"`
	ExtractText        bool    `envconfig:"DZI_EXTRACT_TEXT" default:"true"`
	TileFormat         string  `envconfig:"DZI_TILE_FORMAT" default:"png"`
	TileSetting        string  `envconfig:"DZI_TILE_SETTING" default:""`
	ICCProfileFilepath string  `envconfig:"ICC_PROFILE_PATH" default:"./icc/sRGB_Profile.icc"`
	GraphicsAlphaBits  int     `envconfig:"GRAPHICS_ALPHA_BITS" default:"4"`
}

func (c Config) MakeDziConfig() *dzi.Config {
	if !slices.Contains([]string{dzi.OverprintEnabled, dzi.OverprintSimulate, dzi.OverprintDisable}, c.Overprint) {
		log.Fatalln("overprint not correct")
	}
	return &dzi.Config{
		S3Host:             c.S3Host,
		S3Key:              c.S3Key,
		S3Secret:           c.S3Secret,
		S3Bucket:           c.S3Bucket,
		TileSize:           c.TileSize,
		Overlap:            c.Overlap,
		Resolution:         c.Resolution,
		CoverHeight:        c.CoverHeight,
		ICCProfileFilepath: c.ICCProfileFilepath,
		SplitChannels:      c.SplitChannels,
		DebugMode:          c.DebugMode,
		CopyChannelsToS3:   c.CopyChannelsToS3,
		Overprint:          c.Overprint,
		DefaultDPI:         float64(c.Resolution),
		MinResolution:      c.MinResolution,
		MaxResolution:      c.MaxResolution,
		MaxSizePixels:      c.MaxSizePixels,
		MaxCpuCount:        c.MaxCpuCount,
		ExtractText:        c.ExtractText,
		TileFormat:         c.TileFormat,
		TileSetting:        c.TileSetting,
		GraphicsAlphaBits:  c.GraphicsAlphaBits,
	}
}

func main() {

	//st := time.Now()

	var c Config
	if err := envconfig.Process("", &c); err != nil {
		log.Fatalln(err)
	}
	//c.DebugMode = true
	//c.ICCProfileFilepath = "./icc/sRGB_Profile.icc"

	vips.LoggingSettings(func(messageDomain string, verbosity vips.LogLevel, message string) {}, vips.LogLevelInfo)
	vips.Startup(&vips.Config{
		ConcurrencyLevel: c.MaxCpuCount,
	})
	defer vips.Shutdown()

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
	//buffer, _ := json.Marshal(manifest)
	//os.WriteFile("manifest.json", buffer, 0644)

	//log.Println("Total time:", time.Since(st))
}
