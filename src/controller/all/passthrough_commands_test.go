package all

import (
	"sort"
	"testing"

	"github.com/faradey/madock/v4/src/command"
)

// TestPassThroughCommands pins which commands are allowed to swallow `--help`.
//
// The dispatcher answers `madock <command> --help` for everything else, and this
// flag is the opt-out — for commands that hand their arguments to another
// program, where `--help` is a request for composer's help, not madock's.
//
// It has to be a pinned list because the failure it guards is invisible. Before
// the dispatcher check existed, help was each command's own job: it happened as
// a side effect of calling the argument parser, and a command that never called
// one ran instead. `install` never called one — `madock install --help` on an
// installed project printed the assembled `bin/magento setup:install …` command
// with the admin password in it and ran it over the live database. Eight madock
// commands were in that state on 2026-08-20, and over fifty in madock-pro.
//
// Marking a command here puts it back in that state. That is right for the ones
// listed and wrong for everything else, so it should cost a deliberate edit in
// two places rather than one line nobody reviews.
func TestPassThroughCommands(t *testing.T) {
	want := []string{
		"bc",
		"bigcommerce",
		"claude",
		"cli",
		"composer",
		"m",
		"magento",
		"medusa",
		"mftf",
		"n98",
		"node",
		"prestashop",
		"ps",
		"saleor",
		"shopify",
		"shopify:web",
		"shopify:web:frontend",
		"shopware",
		"shopware:bin",
		"shopware:cli",
		"shopware:consume",
		"spree",
		"sw",
		"sw:b",
		"sw:c",
		"sw:cli",
		"sy",
		"sy:w",
		"sy:w:f",
		"sylius",
		"wp",
	}

	var got []string
	for _, def := range command.GetAll() {
		if def.PassThrough {
			got = append(got, def.Aliases...)
		}
	}
	sort.Strings(got)

	if len(got) != len(want) {
		t.Fatalf("pass-through commands = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pass-through commands = %v, want %v", got, want)
		}
	}
}

// TestEveryCommandExplainsItself is the other half of the same change.
//
// `--help` now prints the registered Help line, so a command registered without
// one answers with its name and a blank line — which reads as "this command has
// no options" rather than "nobody wrote this down". The help index already
// skipped such a command silently (`madock help` filters on a non-empty Help),
// so an unhelped command was invisible in both places at once.
func TestEveryCommandExplainsItself(t *testing.T) {
	for _, def := range command.GetAll() {
		if def.Help == "" {
			t.Errorf("%v is registered with no Help, so `--help` has nothing to print", def.Aliases)
		}
	}
}

// TestDestructiveCommandsAnswerHelp names the ones where getting this wrong is
// not a cosmetic problem.
//
// Each of these either destroys data or takes an environment down, and none of
// them forwards its arguments anywhere, so none may ever be marked pass-through.
// `install` is the one that was.
func TestDestructiveCommandsAnswerHelp(t *testing.T) {
	for _, alias := range []string{
		"install",
		"stop",
		"restart",
		"prune",
		"project:remove",
		"db:import",
		"snapshot:restore",
		"rebuild",
	} {
		def, ok := command.Get(alias)
		if !ok {
			t.Errorf("%s is not registered — if it was renamed, this list has to follow", alias)
			continue
		}
		if def.PassThrough {
			t.Errorf("%s is marked pass-through, so `madock %s --help` runs it", alias, alias)
		}
	}
}
