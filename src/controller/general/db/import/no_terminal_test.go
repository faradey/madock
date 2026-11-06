package _import

import (
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/dbtarget"
)

func withoutATerminal(t *testing.T) {
	t.Helper()
	prev := isTerminal
	isTerminal = func() bool { return false }
	t.Cleanup(func() { isTerminal = prev })
}

// A retry with errors skipped is a choice: without anybody to make it, the
// import has to fail and leave what it did not do visible.
func TestADuplicateRowWithoutATerminalIsNotRetried(t *testing.T) {
	withoutATerminal(t)
	retried := false

	err := handleDuplicateEntry(func(bool, bool) (string, error) {
		retried = true
		return "", nil
	})

	if retried {
		t.Error("the import was run again with nobody to ask")
	}
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Errorf("err = %v, want a refusal naming --force", err)
	}
}

func TestAGtidConflictWithoutATerminalResetsNothing(t *testing.T) {
	withoutATerminal(t)
	var ran []string

	err := handleGtidConflict(dbtarget.Target{},
		func(q string) (string, error) { ran = append(ran, q); return "", nil },
		func(bool, bool) (string, error) { ran = append(ran, "import"); return "", nil })

	if len(ran) > 0 {
		t.Errorf("ran %v with nobody to ask", ran)
	}
	if err == nil || !strings.Contains(err.Error(), "--strip-gtid") || !strings.Contains(err.Error(), "--reset-gtid") {
		t.Errorf("err = %v, want a refusal naming --strip-gtid and --reset-gtid", err)
	}
}
