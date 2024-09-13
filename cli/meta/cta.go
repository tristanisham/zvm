// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// CtaFatal prints an aesthetic CTA and exits with an error.
func CtaFatal(err error) {

	var style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#db0913")).
		Width(10).
		MarginTop(1).
		MarginBottom(1).
		Align(lipgloss.Center)
	fmt.Println(style.Render("Error"))
	log.Error(err)

	blueLink := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0000EE")).
		Bold(true).
		Underline(true)

	yellowText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fee12b"))

	fmt.Printf("\nIf you're experiencing a bug, run %s. If there's a new version of ZVM, we may have already fixed your bug in a new release :)\n", yellowText.Render("zvm upgrade"))
	fmt.Printf("Otherwise, please report this error as a GitHub issue.\n%s\n", blueLink.Render("https://github.com/tristanisham/zvm/issues/\n"))
	os.Exit(1)
}

// CtaUpgradeAvailable prints an aesthetic notice.
func CtaUpgradeAvailable(tag string) {
	var style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#6FA8DC")).
		Width(10).
		MarginTop(1).
		MarginBottom(1).
		Align(lipgloss.Center)
	fmt.Println(style.Render("Notice"))

	blueLink := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0000EE")).
		Bold(true).
		Underline(true)

	yellowText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fee12b"))

	fmt.Printf("\nZVM %s is available. You are currently on %s.\n\nRun %s or download the latest release at\n%s\n\n", blueLink.Render(tag), blueLink.Render(VERSION), yellowText.Render("zvm upgrade"), blueLink.Render("https://github.com/tristanisham/zvm/releases/latest"))

}
