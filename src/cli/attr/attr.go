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
	Name       string `long:"name" description:"Parameter name"`
	Value      string `long:"value" description:"Parameter value"`
	Args       []string
	Download   bool `long:"download" description:"Download Magento from repository"`
	Install    bool `long:"install" description:"Install Magento"`
	Force      bool `long:"forse" short:"f" description:"Install Magento"`
}

func ParseAttributes() {
	if len(os.Args) > 2 {
		args := os.Args[2:]
		Options.Args, _ = flags.NewParser(&Options, flags.HelpFlag|0|flags.PassDoubleDash).ParseArgs(args)
	}
}
