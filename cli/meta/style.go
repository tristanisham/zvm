package meta

import "github.com/charmbracelet/lipgloss"

var TableHeaderStyle = lipgloss.NewStyle().Bold(true)

var TableVersionHeaderStyle = TableHeaderStyle.Width(12)

var TableInstalledHeaderStyle = TableHeaderStyle.Width(12)

var TableInstalledVersionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#24ae32"))

var TableRemoteVersionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#007fff"))

var TableLocalVersionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f5c542"))

var TableOutdatedVersionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f95d5d"))
