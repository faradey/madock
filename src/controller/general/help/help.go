package help

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"help"},
		Handler:  Execute,
		Help:     "Show help",
		Category: "general",
		ArgsType: new(arg_struct.ControllerGeneralHelp),
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralHelp)).(*arg_struct.ControllerGeneralHelp)

	if args.Command != "" {
		showCommandHelp(args.Command)
		return
	}

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
	fmtc.WarningLn("Use 'madock help <command>' for more information about a command.")
}

func showCommandHelp(name string) {
	def, ok := command.Get(name)
	if !ok {
		fmtc.ErrorLn("Unknown command: " + name)
		os.Exit(1)
	}

	primary := def.Aliases[0]
	fmt.Println()
	fmtc.WarningLn("Command: " + primary)
	fmt.Println("  " + def.Help)

	if len(def.Aliases) > 1 {
		fmt.Println()
		fmtc.WarningLn("Aliases:")
		fmt.Println("  " + strings.Join(def.Aliases[1:], ", "))
	}

	if def.ArgsType != nil {
		fmt.Println()
		fmtc.WarningLn("Arguments:")
		p, err := arg.NewParser(arg.Config{
			Program:   "madock " + primary,
			IgnoreEnv: true,
		}, def.ArgsType)
		if err != nil {
			fmtc.ErrorLn("Failed to parse arguments: " + err.Error())
			return
		}
		p.WriteHelp(os.Stdout)
	}

	fmt.Println()
}

func tab() {
	fmt.Print("\t")
}

func tabln() {
	fmt.Println("\t")
}
