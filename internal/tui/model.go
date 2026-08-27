package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"stackpeek/internal/detector"
)

// Model is a Bubble Tea summary view over an immutable scan snapshot.
type Model struct {
	root        string
	findings    []detector.Finding
	diagnostics []detector.Diagnostic
	fatal       error
}

// NewModel copies result so callers retain ownership of their ScanResult.
func NewModel(result detector.ScanResult) Model {
	return Model{
		root:        result.Root,
		findings:    detector.AggregateFindings(result.Findings),
		diagnostics: detector.AggregateDiagnostics(result.Diagnostics),
		fatal:       result.Fatal,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model; q/Q/Ctrl-C quit.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		if k.Type == tea.KeyCtrlC || (k.Type == tea.KeyRunes && len(k.Runes) == 1 && (k.Runes[0] == 'q' || k.Runes[0] == 'Q')) {
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string { return render(m) }
