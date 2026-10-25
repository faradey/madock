package project

import (
	"os"
	"path/filepath"

	"github.com/faradey/madock/v4/src/helper/paths"
)

// madockCAFile is the installation's CA in every project's build context, for
// snippets/dockerfile/common/madock-ca to trust inside the application
// images. Always written, empty when the installation has no CA (nginx/ssl
// off, or not generated yet): a COPY of a file that may not exist would fail
// the build, and an empty one is skipped by the snippet.
const madockCAFile = "madockCA.crt"

// madockCA is the installation's CA, or "" without one; replaceable in tests.
var madockCA = func() string {
	data, err := os.ReadFile(filepath.Join(paths.CtxDir(), "madockCA.pem"))
	if err != nil {
		return ""
	}
	return string(data)
}

func init() {
	RegisterCtxFile(CtxFile{
		Name:   madockCAFile,
		Render: func(string) (string, bool) { return madockCA(), true },
	})
}
