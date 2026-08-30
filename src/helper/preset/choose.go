package preset

import (
	"fmt"

	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
)

// selectPreset is the interactive chooser, indirected so a test can answer it.
//
// The question this asks a person is the only part of Choose that cannot run
// unattended, and it is not the part worth testing. What is worth testing is
// everything around it: that "custom" is offered, that it is offered last, and
// that choosing it means no preset rather than the last one in the list.
var selectPreset = fmtc.SelectPreset

// Choose asks which preset to use and returns it, or nil when the person picks
// the custom option.
//
// Six platforms carried this block, and it was identical in all six down to the
// blank line — every difference between the copies was the title of the
// question. That is what makes it worth sharing and what made it easy to get
// wrong: appending the custom entry, comparing the chosen index against the
// length of the real presets, and treating anything at or past that length as
// "no preset" are three steps that look like bookkeeping and decide whether a
// person who asked to configure by hand gets asked any questions at all.
func Choose(presets []Preset, title string) *Preset {
	options := make([]fmtc.PresetOption, 0, len(presets)+1)
	for _, p := range presets {
		options = append(options, fmtc.PresetOption{
			Name:        p.Name,
			Description: p.Description,
			IsCustom:    false,
		})
	}
	options = append(options, fmtc.PresetOption{
		Name:        CustomPreset.Name,
		Description: CustomPreset.Description,
		IsCustom:    true,
	})

	fmt.Println("")
	fmtc.TitleLn(title)

	chosen := selectPreset("Configuration", options)
	if chosen < 0 || chosen >= len(presets) {
		return nil
	}

	fmt.Println("")
	fmtc.SuccessIconLn(fmt.Sprintf("Using preset: %s", presets[chosen].Name))

	return &presets[chosen]
}
