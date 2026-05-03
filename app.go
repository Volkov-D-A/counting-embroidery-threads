package main

import (
	"context"

	"counting-embroidery-threads/internal/dmc"
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

// ImportThreadFilePath imports a known TXT report path without opening a dialog.
func (a *App) ImportThreadFilePath(path string, skeinLengthMeters float64) (*threadcalc.ImportResult, error) {
	return threadcalc.ImportFile(path, skeinLengthMeters)
}

// RecalculateThreadCodes applies user code corrections to the current result.
func (a *App) RecalculateThreadCodes(result *threadcalc.ImportResult, corrections []threadcalc.CodeCorrection, skeinLengthMeters float64) (*threadcalc.ImportResult, error) {
	return threadcalc.RecalculateWithCorrections(result, corrections, skeinLengthMeters)
}

// GetBuiltinPalette returns the bundled DMC palette for browsing in the UI.
func (a *App) GetBuiltinPalette() ([]dmc.PaletteEntry, error) {
	return dmc.LoadPaletteEntries()
}

// GetPalette returns the bundled DMC palette with user entries overlaid.
func (a *App) GetPalette() ([]dmc.PaletteEntry, error) {
	return dmc.LoadEffectivePaletteEntries()
}

// GetUserPalettePath returns the JSON file path used for user palette overrides.
func (a *App) GetUserPalettePath() (string, error) {
	return dmc.UserPalettePath()
}

// SaveUserPaletteEntry creates or replaces one user palette entry.
func (a *App) SaveUserPaletteEntry(entry dmc.PaletteEntry) ([]dmc.PaletteEntry, error) {
	return dmc.SaveUserPaletteEntry(entry)
}

// DeleteUserPaletteEntry removes one user palette entry.
func (a *App) DeleteUserPaletteEntry(code string) ([]dmc.PaletteEntry, error) {
	return dmc.DeleteUserPaletteEntry(code)
}

// GetTransformationSettings returns saved report transformation settings.
func (a *App) GetTransformationSettings() (*threadcalc.TransformationSettings, error) {
	return threadcalc.LoadTransformationSettings()
}

// GetTransformationSettingsPath returns the JSON file path used for transformation settings.
func (a *App) GetTransformationSettingsPath() (string, error) {
	return threadcalc.TransformationSettingsPath()
}

// SaveTransformationSettings stores report transformation settings.
func (a *App) SaveTransformationSettings(settings *threadcalc.TransformationSettings) (*threadcalc.TransformationSettings, error) {
	return threadcalc.SaveTransformationSettings(settings)
}
