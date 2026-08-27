package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"stackpeek/internal/detector"
)

func TestView_StableHeaderAndRoot(t *testing.T) {
	view := NewModel(detector.ScanResult{
		Root:     "/repo/demo",
		Findings: []detector.Finding{{Technology: "Node.js", Signal: "package.json"}},
	}).View()
	if !strings.Contains(view, "Stackpeek") || !strings.Contains(view, "/repo/demo") {
		t.Fatalf("missing header/root:\n%s", view)
	}
}

func TestView_SortedFindingsAndSources(t *testing.T) {
	view := NewModel(detector.ScanResult{
		Root: "/r",
		Findings: []detector.Finding{
			{Technology: "Python", Signal: "requirements.txt"},
			{Technology: "Docker", Signal: "Dockerfile"},
			{Technology: "Node.js", Signal: "package.json"},
		},
	}).View()
	docker, node, python := strings.Index(view, "Docker"), strings.Index(view, "Node.js"), strings.Index(view, "Python")
	if docker < 0 || node < 0 || python < 0 || !(docker < node && node < python) {
		t.Fatalf("findings missing/unsorted: d=%d n=%d p=%d\n%s", docker, node, python, view)
	}
	for _, sig := range []string{"Dockerfile", "package.json", "requirements.txt"} {
		if !strings.Contains(view, sig) {
			t.Fatalf("missing signal %q:\n%s", sig, view)
		}
	}
}

func TestView_Diagnostics(t *testing.T) {
	view := NewModel(detector.ScanResult{
		Root:        "/r",
		Findings:    []detector.Finding{{Technology: "Node.js", Signal: "package.json"}},
		Diagnostics: []detector.Diagnostic{{Path: "composer.json", Code: "parse", Message: "invalid json"}},
	}).View()
	for _, want := range []string{"composer.json", "parse", "invalid json"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing diagnostic %q:\n%s", want, view)
		}
	}
}

func TestView_EmptyFindings(t *testing.T) {
	view := NewModel(detector.ScanResult{Root: "/empty", Findings: []detector.Finding{}}).View()
	if !strings.Contains(strings.ToLower(view), "no technologies detected") {
		t.Fatalf("missing empty-state:\n%s", view)
	}
}

func TestView_FatalState(t *testing.T) {
	view := NewModel(detector.ScanResult{
		Root:  "/missing",
		Fatal: errors.New(`stat root "/missing": no such file or directory`),
	}).View()
	if !strings.Contains(view, "stat root") {
		t.Fatalf("missing fatal:\n%s", view)
	}
}

func TestUpdate_QuitKeys(t *testing.T) {
	cases := []struct {
		name string
		msg  tea.Msg
	}{
		{"q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{"Q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}}},
		{"ctrl-c", tea.KeyMsg{Type: tea.KeyCtrlC}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, cmd := NewModel(detector.ScanResult{Root: "/r"}).Update(tc.msg)
			if cmd == nil {
				t.Fatal("expected quit command")
			}
			msg := cmd()
			if _, ok := msg.(tea.QuitMsg); !ok {
				t.Fatalf("expected tea.QuitMsg, got %T", msg)
			}
		})
	}
}

func TestNewModel_DoesNotMutateInput(t *testing.T) {
	in := detector.ScanResult{
		Root:        "/r",
		Findings:    []detector.Finding{{Technology: "Zulu", Signal: "z"}, {Technology: "Alpha", Signal: "a"}},
		Diagnostics: []detector.Diagnostic{{Path: "b", Code: "c", Message: "m"}},
	}
	orig := in.Findings[0].Technology
	_ = NewModel(in).View()
	if in.Findings[0].Technology != orig {
		t.Fatalf("input mutated: got %q want %q", in.Findings[0].Technology, orig)
	}
}
