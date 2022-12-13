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
	Hosts      bool   `long:"hosts" description:"Website Hosts and codes"`
	Args       []string
	Download   bool `long:"download" description:"Download Magento from repository"`
	Install    bool `long:"install" description:"Install Magento"`
}

func ParseAttributes() {
	if len(os.Args) > 2 {
		np := flags.NewParser(&Options, flags.HelpFlag|0|flags.PassDoubleDash)
		Options.Args, _ = np.Parse()
	}
}
