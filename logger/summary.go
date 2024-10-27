package logger

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#874BFD")).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B4BEFE")).
			Italic(true)

	logFileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#89B4FA")).
			Padding(1)
)

func PrintLogSummary(loggers []*RateLimitedLogger) {
	var allUpdatedFiles []string
	for _, logger := range loggers {
		allUpdatedFiles = append(allUpdatedFiles, logger.GetUpdatedLogFiles()...)
	}

	if len(allUpdatedFiles) == 0 {
		fmt.Println(subtitleStyle.Render("No logs were updated during this session."))
		return
	}

	summary := strings.Builder{}
	summary.WriteString(titleStyle.Render("Log Summary") + "\n\n")
	summary.WriteString(subtitleStyle.Render("The following log files were updated:") + "\n\n")

	for _, file := range allUpdatedFiles {
		// Use the full path instead of just the filename
		summary.WriteString(logFileStyle.Render("  • "+file) + "\n")
	}

	fmt.Println(borderStyle.Render(summary.String()))
}
