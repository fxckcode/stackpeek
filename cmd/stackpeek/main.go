package main

import (
	"context"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"stackpeek/internal/detector"
	"stackpeek/internal/tui"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, launchTUI)) }

func launchTUI(m tui.Model) error {
	_, err := tea.NewProgram(m).Run()
	return err
}

func run(args []string, stdout, stderr io.Writer, launch func(tui.Model) error) int {
	_ = stdout
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: stackpeek [path]")
		return 1
	}
	path := "."
	if len(args) == 1 {
		path = args[0]
	}
	ctx := context.Background()
	abs, err := detector.ValidateRoot(ctx, path)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	fsys, err := detector.OpenRoot(ctx, abs)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	result := detector.DefaultRegistry().Scan(ctx, abs, fsys)
	if err := launch(tui.NewModel(result)); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}
