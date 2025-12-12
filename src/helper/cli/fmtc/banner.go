package fmtc

import (
	"fmt"

	"github.com/faradey/madock/src/helper/cli/color"
)

// Banner displays a styled ASCII banner for setup
func Banner(title, subtitle string) {
	width := 44

	// Top border
	fmt.Printf("%s╔", color.Cyan)
	for i := 0; i < width; i++ {
		fmt.Print("═")
	}
	fmt.Printf("╗%s\n", color.Reset)

	// Empty line
	fmt.Printf("%s║%s", color.Cyan, color.Reset)
	for i := 0; i < width; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("%s║%s\n", color.Cyan, color.Reset)

	// Title line (centered)
	titlePadding := (width - len(title)) / 2
	fmt.Printf("%s║%s", color.Cyan, color.Reset)
	for i := 0; i < titlePadding; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("%s%s%s", color.Green, title, color.Reset)
	for i := 0; i < width-titlePadding-len(title); i++ {
		fmt.Print(" ")
	}
	fmt.Printf("%s║%s\n", color.Cyan, color.Reset)

	// Subtitle line (centered)
	subtitlePadding := (width - len(subtitle)) / 2
	fmt.Printf("%s║%s", color.Cyan, color.Reset)
	for i := 0; i < subtitlePadding; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("%s%s%s", color.Gray, subtitle, color.Reset)
	for i := 0; i < width-subtitlePadding-len(subtitle); i++ {
		fmt.Print(" ")
	}
	fmt.Printf("%s║%s\n", color.Cyan, color.Reset)

	// Empty line
	fmt.Printf("%s║%s", color.Cyan, color.Reset)
	for i := 0; i < width; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("%s║%s\n", color.Cyan, color.Reset)

	// Bottom border
	fmt.Printf("%s╚", color.Cyan)
	for i := 0; i < width; i++ {
		fmt.Print("═")
	}
	fmt.Printf("╝%s\n", color.Reset)
}

// BannerSimple displays a simpler single-line banner
func BannerSimple(title string) {
	fmt.Printf("\n%s══════ %s%s%s ══════%s\n\n",
		color.Cyan,
		color.Green,
		title,
		color.Cyan,
		color.Reset,
	)
}
