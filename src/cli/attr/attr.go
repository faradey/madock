package attr

import (
	"os"

	flags "github.com/jessevdk/go-flags"
)

var Options struct {
	Path       string `long:"path" description:"Path to file on server (from Magento root)"`
	Global     bool   `long:"global" description:"Global"`
	ImagesOnly bool   `long:"images-only" description:"Sync images only"`
	Compress   bool   `long:"compress" description:"Compress images"`
}

func ParseAttributes() {
	if len(os.Args) > 2 {
		flags.ParseArgs(&Options, os.Args)
	}
}
