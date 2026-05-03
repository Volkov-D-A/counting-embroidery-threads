package threadcalc

import (
	"math"
	"path/filepath"
	"testing"

	"counting-embroidery-threads/internal/dmc"
)

func TestImportExampleFiles(t *testing.T) {
	tests := []struct {
		name       string
		rows       int
		beadRows   int
		minResults int
	}{
		{name: "Заморские гости расход.TXT", rows: 101, beadRows: 0, minResults: 60},
		{name: "Лола.TXT", rows: 24, beadRows: 0, minResults: 18},
		{name: "Целитель для разбора.TXT", rows: 41, beadRows: 0, minResults: 30},
		{name: "Семплер Цифры.TXT", rows: 14, beadRows: 7, minResults: 14},
		{name: "Анемоны расход.TXT", rows: 78, beadRows: 0, minResults: 42},
		{name: "Блюдо с фруктами_расход.TXT", rows: 80, beadRows: 0, minResults: 43},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ImportFile(filepath.Join("..", "..", "examle", test.name), DefaultSkeinLengthMeters)
			if err != nil {
				t.Fatal(err)
			}
			if result.RowsImported != test.rows {
				t.Fatalf("RowsImported = %d, want %d", result.RowsImported, test.rows)
			}
			if result.BeadRowsIgnored != test.beadRows {
				t.Fatalf("BeadRowsIgnored = %d, want %d", result.BeadRowsIgnored, test.beadRows)
			}
			if len(result.Items) < test.minResults {
				t.Fatalf("got %d result items, want at least %d", len(result.Items), test.minResults)
			}
		})
	}
}

func TestImportResultUsesEmptySlices(t *testing.T) {
	result, err := ImportFile(filepath.Join("..", "..", "examle", "Анемоны расход.TXT"), DefaultSkeinLengthMeters)
	if err != nil {
		t.Fatal(err)
	}
	if result.Items == nil {
		t.Fatal("Items is nil, want empty or populated slice")
	}
	if result.Warnings == nil {
		t.Fatal("Warnings is nil, want empty slice")
	}

	item := requireItem(t, result, "310")
	if item.Notes == nil {
		t.Fatal("ThreadResult.Notes is nil, want empty slice")
	}
}

func TestImportUsesTotalMetersAndMergesDuplicates(t *testing.T) {
	result, err := ImportFile(filepath.Join("..", "..", "examle", "Лола.TXT"), DefaultSkeinLengthMeters)
	if err != nil {
		t.Fatal(err)
	}

	black := requireItem(t, result, "310")
	if black.Meters != 5.5 {
		t.Fatalf("310 meters = %.2f, want 5.50", black.Meters)
	}
	if black.Skeins != 1 {
		t.Fatalf("310 skeins = %d, want 1", black.Skeins)
	}

	white := requireItem(t, result, "B5200")
	if white.Meters != 3.55 {
		t.Fatalf("B5200 meters = %.2f, want 3.55", white.Meters)
	}
	if white.ColorHex != "#FFFFFF" {
		t.Fatalf("B5200 color = %s, want #FFFFFF", white.ColorHex)
	}
}

func TestRecalculateWithCodeCorrectionsMergesTargetCode(t *testing.T) {
	result, err := ImportFile(filepath.Join("..", "..", "examle", "Целитель для разбора.TXT"), DefaultSkeinLengthMeters)
	if err != nil {
		t.Fatal(err)
	}
	originalBlack := requireItem(t, result, "310")
	originalDarkBlueberry := requireItem(t, result, "32")

	recalculated, err := RecalculateWithCorrections(result, []CodeCorrection{
		{From: "32", To: "310"},
	}, DefaultSkeinLengthMeters)
	if err != nil {
		t.Fatal(err)
	}

	if itemExists(recalculated, "32") {
		t.Fatal("32 should be merged into 310 after correction")
	}

	black := requireItem(t, recalculated, "310")
	wantMeters := roundMeters(originalBlack.Meters + originalDarkBlueberry.Meters)
	if black.Meters != wantMeters {
		t.Fatalf("310 meters = %.2f, want %.2f", black.Meters, wantMeters)
	}
	wantSkeins := int(math.Ceil(wantMeters / DefaultSkeinLengthMeters))
	if black.Skeins != wantSkeins {
		t.Fatalf("310 skeins = %d, want %d", black.Skeins, wantSkeins)
	}
	if !containsNote(black.Notes, "исправлено: 32 -> 310") {
		t.Fatalf("correction note not found: %v", black.Notes)
	}
	if recalculated.TotalMeters != result.TotalMeters {
		t.Fatalf("TotalMeters = %.2f, want %.2f", recalculated.TotalMeters, result.TotalMeters)
	}
	if recalculated.TotalSkeins >= result.TotalSkeins {
		t.Fatalf("TotalSkeins = %d, want less than original %d after merge", recalculated.TotalSkeins, result.TotalSkeins)
	}
}

func TestRecalculateWithCodeCorrectionsNormalizesTargetCode(t *testing.T) {
	source := &ImportResult{
		SkeinLengthMeters: DefaultSkeinLengthMeters,
		Items: []ThreadResult{
			{Code: "BAD", Meters: 1.2, Skeins: 1, Notes: []string{"цвет не найден в палитре"}},
		},
		Warnings: []string{"Для кода BAD не найден цвет палитры"},
	}

	recalculated, err := RecalculateWithCorrections(source, []CodeCorrection{
		{From: "BAD", To: "е321(k29)"},
	}, DefaultSkeinLengthMeters)
	if err != nil {
		t.Fatal(err)
	}

	item := requireItem(t, recalculated, "E321(K29)")
	if item.ColorHex != "#BD1136" {
		t.Fatalf("E321(K29) color = %s, want #BD1136", item.ColorHex)
	}
	if !containsNote(item.Notes, "исправлено: BAD -> E321(K29)") {
		t.Fatalf("correction note not found: %v", item.Notes)
	}
	if len(recalculated.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", recalculated.Warnings)
	}
}

func TestCyrillicENormalizedAndPaletteUsesSpecialtyThenBaseDMC(t *testing.T) {
	result, err := ImportFile(filepath.Join("..", "..", "examle", "Целитель для разбора.TXT"), DefaultSkeinLengthMeters)
	if err != nil {
		t.Fatal(err)
	}

	darkBlueberry := requireItem(t, result, "32")
	if darkBlueberry.ColorHex != "#4D2E8A" {
		t.Fatalf("32 color = %s, want #4D2E8A", darkBlueberry.ColorHex)
	}
	if !darkBlueberry.PaletteFound {
		t.Fatal("32 should be found in DMC palette")
	}

	item := requireItem(t, result, "E321(K29)")
	if item.ColorHex != "#BD1136" {
		t.Fatalf("E321(K29) color = %s, want #BD1136", item.ColorHex)
	}

	lightEffects := requireItem(t, result, "E130(K17)")
	if lightEffects.ColorHex != "#FFFFFF" {
		t.Fatalf("E130(K17) color = %s, want #FFFFFF", lightEffects.ColorHex)
	}
	if !lightEffects.PaletteFound {
		t.Fatal("E130(K17) should be found in DMC Light Effects palette")
	}

	regularFallback := requireItem(t, result, "E797(K34)")
	if regularFallback.ColorHex != "#13477D" {
		t.Fatalf("E797(K34) fallback color = %s, want #13477D", regularFallback.ColorHex)
	}
	if !containsNote(regularFallback.Notes, "цвет взят из 797") {
		t.Fatalf("expected fallback note for E797(K34), got %v", regularFallback.Notes)
	}
}

func TestBlendsWithThreeOrMorePartsStillUseHalfTotal(t *testing.T) {
	palette := dmc.Palette{
		"111": "#111111",
		"222": "#222222",
		"333": "#333333",
	}
	report := "Thread lengths\nThread (DMC)\tDescription\tStitches\tBackstitches\tLength(m)\tBacks\tTotal\n111+222+333\t+\t1\t0\t10.0\t0.0\t10.0\n"

	result := parseAndCalculate(report, palette, DefaultSkeinLengthMeters)

	for _, code := range []string{"111", "222", "333"} {
		item := requireItem(t, result, code)
		if item.Meters != 5 {
			t.Fatalf("%s meters = %.2f, want 5.00", code, item.Meters)
		}
	}
}

func TestSingleDigitNewDMCCodeIsZeroPadded(t *testing.T) {
	palette := dmc.Palette{
		"01": "#E3E3E6",
	}
	report := "Thread lengths\nThread (DMC)\tDescription\tStitches\tBackstitches\tLength(m)\tBacks\tTotal\n1\tWhite Tin\t1\t0\t1.0\t0.0\t1.0\n"

	result := parseAndCalculate(report, palette, DefaultSkeinLengthMeters)
	item := requireItem(t, result, "01")
	if item.ColorHex != "#E3E3E6" {
		t.Fatalf("01 color = %s, want #E3E3E6", item.ColorHex)
	}
}

func TestLetterCodesFromPaletteAreRecognized(t *testing.T) {
	result, err := ImportFile(filepath.Join("..", "..", "examle", "Анемоны расход.TXT"), DefaultSkeinLengthMeters)
	if err != nil {
		t.Fatal(err)
	}

	white := requireItem(t, result, "WHITE")
	if white.ColorHex != "#FCFBF8" {
		t.Fatalf("WHITE color = %s, want #FCFBF8", white.ColorHex)
	}

	ecru := requireItem(t, result, "ECRU")
	if ecru.ColorHex != "#F0EADA" {
		t.Fatalf("ECRU color = %s, want #F0EADA", ecru.ColorHex)
	}

	for _, warning := range result.Warnings {
		if warning == "Для кода WHITE не найден цвет палитры" || warning == "Для кода ECRU не найден цвет палитры" {
			t.Fatalf("unexpected palette warning: %s", warning)
		}
	}
}

func TestBlendPartCanBeStoredInDescription(t *testing.T) {
	result, err := ImportFile(filepath.Join("..", "..", "examle", "Блюдо с фруктами_расход.TXT"), DefaultSkeinLengthMeters)
	if err != nil {
		t.Fatal(err)
	}

	ecru := requireItem(t, result, "ECRU")
	if ecru.Meters != 1.65 {
		t.Fatalf("ECRU meters = %.2f, want 1.65", ecru.Meters)
	}
}

func requireItem(t *testing.T, result *ImportResult, code string) ThreadResult {
	t.Helper()
	for _, item := range result.Items {
		if item.Code == code {
			return item
		}
	}
	t.Fatalf("item %s not found in result", code)
	return ThreadResult{}
}

func itemExists(result *ImportResult, code string) bool {
	for _, item := range result.Items {
		if item.Code == code {
			return true
		}
	}
	return false
}

func containsNote(notes []string, expected string) bool {
	for _, note := range notes {
		if note == expected {
			return true
		}
	}
	return false
}
