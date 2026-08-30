package fmtc

import (
	"fmt"

	"github.com/faradey/madock/v4/src/helper/cli/color"
)

// What is left of a print-helper set that nothing used.
//
// The file offered eighteen functions and fifteen icon constants. One function
// was called, from one place; the constants were referenced only by the
// seventeen functions that were not. madock-pro, the only other consumer of this
// module, uses nine functions from this package and none of them from here.
//
// Removed rather than kept in case somebody wants them, because "in case" is how
// it got here: a set written whole, against needs that never arrived, and read
// since as if it were in use. Anything genuinely needed comes back in a commit
// that says who needs it.
const iconArrow = "→"

// PrintKeyValue prints an indented `key: value` line.
func PrintKeyValue(key, value string) {
	fmt.Printf("  %s%s%s %s: %s\n", color.Gray, iconArrow, color.Reset, key, value)
}
