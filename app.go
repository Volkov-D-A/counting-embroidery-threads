package main

import (
	"context"

	"counting-embroidery-threads/internal/threadcalc"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ImportThreadFile opens a TXT report and returns aggregated DMC usage.
func (a *App) ImportThreadFile(skeinLengthMeters float64) (*threadcalc.ImportResult, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Выберите TXT-файл расхода нитей",
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "TXT отчеты",
				Pattern:     "*.txt;*.TXT",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return &threadcalc.ImportResult{
			Cancelled:         true,
			SkeinLengthMeters: threadcalc.DefaultSkeinLengthMeters,
			Items:             []threadcalc.ThreadResult{},
			Warnings:          []string{},
		}, nil
	}

	return threadcalc.ImportFile(path, skeinLengthMeters)
}

// RecalculateThreadCodes applies user code corrections to the current result.
func (a *App) RecalculateThreadCodes(result *threadcalc.ImportResult, corrections []threadcalc.CodeCorrection, skeinLengthMeters float64) (*threadcalc.ImportResult, error) {
	return threadcalc.RecalculateWithCorrections(result, corrections, skeinLengthMeters)
}
