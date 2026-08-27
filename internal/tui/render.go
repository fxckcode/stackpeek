package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	labelStyle  = lipgloss.NewStyle().Faint(true)
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

func render(m Model) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%s%s\n", headerStyle.Render("Stackpeek"), labelStyle.Render("Root: "), m.root)
	if m.fatal != nil {
		fmt.Fprintf(&b, "%s\n", errStyle.Render("Fatal: "+m.fatal.Error()))
		return b.String()
	}
	fmt.Fprintf(&b, "%s\n", labelStyle.Render("Findings"))
	if len(m.findings) == 0 {
		b.WriteString("no technologies detected\n")
	} else {
		for _, f := range m.findings {
			fmt.Fprintf(&b, "  %s  %s\n", f.Technology, labelStyle.Render("("+f.Signal+")"))
		}
	}
	if len(m.diagnostics) == 0 {
		return b.String()
	}
	fmt.Fprintf(&b, "%s\n", labelStyle.Render("Diagnostics"))
	for _, d := range m.diagnostics {
		fmt.Fprintf(&b, "  %s [%s] %s\n", d.Path, d.Code, d.Message)
	}
	return b.String()
}
