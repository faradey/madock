package attr

type Arguments struct {
}

type ArgumentsWithArgs struct {
	Arguments
	Args []string
}

var Options struct {
	Path          string `long:"path" description:"Path to file on server (from Magento root)"`
	Global        bool   `long:"global" description:"Global"`
	ImagesOnly    bool   `long:"images-only" description:"Sync images only"`
	Compress      bool   `long:"compress" description:"Compress images"`
	Name          string `long:"name" description:"Parameter name"`
	Value         string `long:"value" description:"Parameter value"`
	Args          []string
	Download      bool     `long:"download" description:"Download Magento from repository"`
	Install       bool     `long:"install" description:"Install Magento"`
	Force         bool     `long:"force" short:"f" description:"Force"`
	File          string   `long:"file" description:"File path"`
	Title         string   `long:"title" description:"Title"`
	Host          string   `long:"host" description:"Host"`
	WithVolumes   bool     `long:"with-volumes" description:"With Volumes"`
	WithChown     bool     `long:"with-chown" description:"With Chown"`
	SampleData    bool     `long:"sample-data" description:"sample-data"`
	DBServiceName string   `long:"service-name" description:"DB service name"`
	IgnoreTable   []string `long:"ignore-table" description:"Ignore db table"`
}
