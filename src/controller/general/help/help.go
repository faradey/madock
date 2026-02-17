package help

import (
	"fmt"
	"sort"
	"strings"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"help"},
		Handler:  Execute,
		Help:     "Show help",
		Category: "general",
	})
}

func Execute() {
	attr.Parse(new(arg_struct.ControllerGeneralHelp))

	fmtc.WarningLn("Usage:")
	tab()
	fmt.Println("command [arguments]")

	// Collect and deduplicate commands
	defs := command.GetAll()

	type entry struct {
		primary string
		aliases string
		help    string
	}

	var entries []entry
	for _, def := range defs {
		if len(def.Aliases) == 0 || def.Help == "" {
			continue
		}
		primary := def.Aliases[0]
		aliasStr := ""
		if len(def.Aliases) > 1 {
			aliasStr = strings.Join(def.Aliases[1:], ", ")
		}
		entries = append(entries, entry{
			primary: primary,
			aliases: aliasStr,
			help:    def.Help,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].primary < entries[j].primary
	})

	// Find max command width for alignment
	maxWidth := 0
	for _, e := range entries {
		w := len(e.primary)
		if e.aliases != "" {
			w += len(e.aliases) + 3 // " (aliases)"
		}
		if w > maxWidth {
			maxWidth = w
		}
	}

	fmtc.Warning("Available commands:")
	for _, e := range entries {
		tabln()
		tab()
		name := e.primary
		if e.aliases != "" {
			name += " (" + e.aliases + ")"
		}
		fmtc.Success(name)
		// Pad to align descriptions
		padding := max(maxWidth-len(name)+2, 2)
		fmt.Print(strings.Repeat(" ", padding))
		fmt.Println(e.help)
	}

	fmt.Println("")
}

func tab() {
	fmt.Print("\t")
}

func tabln() {
	fmt.Println("\t")
}
