package dockerassets

import (
	"io/fs"
	"strings"
	"testing"
)

// TestTemplateTagsAreBalanced checks every embedded template for well-formed
// conditional tags, because the engine that reads them cannot complain.
//
// processConditionals (src/helper/configs/config.go) locates the closing tag by
// counting openings. One unbalanced <<<if — including one written inside a
// comment — makes findMatchingEndif return -1, and the loop then abandons the
// whole file with every conditional left unresolved. Nothing errors: the
// generated config is simply written out with raw <<<iffalse>>> in it and every
// branch kept, which nginx either rejects or, worse, accepts as duplicate
// server blocks.
//
// Bash here-strings (`read a b c <<< "$x"`) are legal template content and are
// not tags, so only the three tag forms are counted.
func TestTemplateTagsAreBalanced(t *testing.T) {
	err := fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		b, readErr := fs.ReadFile(FS, path)
		if readErr != nil {
			return readErr
		}
		body := string(b)

		opens := strings.Count(body, "<<<if")
		closes := strings.Count(body, "<<<endif>>>")
		if opens != closes {
			t.Errorf("%s: %d <<<if against %d <<<endif>>> — processConditionals will abandon the file and leave every conditional unresolved", path, opens, closes)
		}

		// An opening tag has to be terminated, or the engine gives up at the
		// same place for a different reason.
		for _, part := range strings.Split(body, "<<<if")[1:] {
			if !strings.Contains(part, ">>>") {
				t.Errorf("%s: an <<<if is never closed by >>>", path)
			}
		}

		if strings.Count(body, "<<<else>>>") > opens {
			t.Errorf("%s: more <<<else>>> than <<<if — an else outside a conditional is silently kept in the output", path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded templates: %v", err)
	}
}
