// Package info provides a registry for platform-specific "info" handlers
// and the report model every one of them fills.
//
// A handler does not print. It returns blocks — key/value sections, tables
// and plain lines — and the general/info controller renders the whole
// report in the format the user asked for. Printing from the handler was
// the previous shape, and it meant the platform half of `madock info`
// could only ever be text.
package info

// InfoContext provides all data a platform info handler needs.
type InfoContext struct {
	ProjectName string
	ProjectPath string
	ProjectConf map[string]string
	Service     string
}

// InfoHandler is implemented by each platform to handle the "info" command.
type InfoHandler interface {
	Collect(ctx *InfoContext) ([]Block, error)
}

// Item is one key/value pair of a section block.
type Item struct {
	Key   string
	Value string
}

// Column names one column of a table block. Key is the machine name used
// by the JSON and XML renderers; Title is what the text and Markdown
// renderers print.
type Column struct {
	Key   string
	Title string
}

// Block is one part of the report. Exactly one of Items, Rows or Lines is
// meant to be set; a block with none of them renders as an empty section.
type Block struct {
	// Key is the machine name — "project", "modules" — used as the JSON
	// field and the XML element. Blocks sharing a Key are grouped under it,
	// told apart by Name.
	Key string
	// Name distinguishes blocks that share a Key, such as one "scope" block
	// per configured scope.
	Name string
	// Title is the human heading for text and Markdown.
	Title string

	Items []Item

	Columns []Column
	Rows    [][]string
	// RowName is the XML element for one table row; defaults to "row".
	RowName string

	Lines []string
}

var handlers = map[string]InfoHandler{}

// Register adds a platform info handler to the registry.
func Register(platform string, handler InfoHandler) {
	handlers[platform] = handler
}

// Get returns the info handler for the given platform name.
func Get(platform string) (InfoHandler, bool) {
	h, ok := handlers[platform]
	return h, ok
}
