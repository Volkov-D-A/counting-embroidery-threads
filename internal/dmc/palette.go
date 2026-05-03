package dmc

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"counting-embroidery-threads/internal/dmccode"
)

//go:embed floss_adrianj.csv floss_2017.csv floss_light_effects.csv
var paletteFS embed.FS
var hexColor = regexp.MustCompile(`^#[0-9A-F]{6}$`)

const userPaletteFilename = "user_palette.json"

// Color describes a display colour from the bundled DMC palette.
type Color struct {
	Hex  string
	Name string
}

// Palette maps DMC codes to display colours.
type Palette map[string]Color

// PaletteEntry is a single display row from the bundled DMC palette.
type PaletteEntry struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Hex    string `json:"hex"`
	Source string `json:"source,omitempty"`
}

type userPaletteFile struct {
	Colors []PaletteEntry `json:"colors"`
}

// LoadPalette loads the bundled open DMC palette.
func LoadPalette() (Palette, error) {
	palette := Palette{}
	for _, filename := range []string{"floss_adrianj.csv", "floss_2017.csv", "floss_light_effects.csv"} {
		if err := loadCSVPalette(palette, filename); err != nil {
			return nil, err
		}
	}
	if color, ok := palette["WHITE"]; ok {
		palette["BLANC"] = color
	}
	if color, ok := palette["ECRU"]; ok {
		palette["ECRUT"] = color
	}

	return palette, nil
}

// LoadEffectivePalette loads the bundled palette with user entries overlaid.
func LoadEffectivePalette() (Palette, error) {
	palette, err := LoadPalette()
	if err != nil {
		return nil, err
	}
	userEntries, err := LoadUserPaletteEntries()
	if err != nil {
		return nil, err
	}
	for _, entry := range userEntries {
		palette[entry.Code] = Color{
			Hex:  entry.Hex,
			Name: entry.Name,
		}
	}
	return palette, nil
}

// LoadPaletteEntries loads the bundled palette as a sorted list for the UI.
func LoadPaletteEntries() ([]PaletteEntry, error) {
	palette, err := LoadPalette()
	if err != nil {
		return nil, err
	}

	return paletteEntries(palette, "built-in"), nil
}

// LoadEffectivePaletteEntries loads the bundled palette with user entries overlaid.
func LoadEffectivePaletteEntries() ([]PaletteEntry, error) {
	bundled, err := LoadPalette()
	if err != nil {
		return nil, err
	}
	userEntries, err := LoadUserPaletteEntries()
	if err != nil {
		return nil, err
	}

	entriesByCode := map[string]PaletteEntry{}
	for code, color := range bundled {
		entriesByCode[code] = PaletteEntry{
			Code:   code,
			Name:   color.Name,
			Hex:    color.Hex,
			Source: "built-in",
		}
	}
	for _, entry := range userEntries {
		entry.Source = "user"
		entriesByCode[entry.Code] = entry
	}

	entries := make([]PaletteEntry, 0, len(entriesByCode))
	for _, entry := range entriesByCode {
		entries = append(entries, entry)
	}
	sortPaletteEntries(entries)
	return entries, nil
}

// UserPalettePath returns the JSON file path used for user palette overrides.
func UserPalettePath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(executable), userPaletteFilename), nil
}

// LoadUserPaletteEntries loads user palette entries from the file near the executable.
func LoadUserPaletteEntries() ([]PaletteEntry, error) {
	path, err := UserPalettePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []PaletteEntry{}, nil
	}
	if err != nil {
		return nil, err
	}

	var file userPaletteFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return normalizeUserPaletteEntries(file.Colors)
}

// SaveUserPaletteEntry creates or replaces one user palette entry.
func SaveUserPaletteEntry(entry PaletteEntry) ([]PaletteEntry, error) {
	entry, err := normalizeUserPaletteEntry(entry)
	if err != nil {
		return nil, err
	}

	entries, err := LoadUserPaletteEntries()
	if err != nil {
		return nil, err
	}

	replaced := false
	for index, current := range entries {
		if current.Code == entry.Code {
			entries[index] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		entries = append(entries, entry)
	}

	return writeUserPaletteEntries(entries)
}

// DeleteUserPaletteEntry removes a user palette entry by normalized code.
func DeleteUserPaletteEntry(code string) ([]PaletteEntry, error) {
	normalizedCode := dmccode.Normalize(code)
	entries, err := LoadUserPaletteEntries()
	if err != nil {
		return nil, err
	}

	next := entries[:0]
	for _, entry := range entries {
		if entry.Code != normalizedCode {
			next = append(next, entry)
		}
	}
	return writeUserPaletteEntries(next)
}

func paletteEntries(palette Palette, source string) []PaletteEntry {
	entries := make([]PaletteEntry, 0, len(palette))
	for code, color := range palette {
		entries = append(entries, PaletteEntry{
			Code:   code,
			Name:   color.Name,
			Hex:    color.Hex,
			Source: source,
		})
	}

	sortPaletteEntries(entries)
	return entries
}

func writeUserPaletteEntries(entries []PaletteEntry) ([]PaletteEntry, error) {
	entries, err := normalizeUserPaletteEntries(entries)
	if err != nil {
		return nil, err
	}

	path, err := UserPalettePath()
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(userPaletteFile{Colors: entries}, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, err
	}
	return entries, nil
}

func normalizeUserPaletteEntries(entries []PaletteEntry) ([]PaletteEntry, error) {
	entriesByCode := map[string]PaletteEntry{}
	for _, entry := range entries {
		normalized, err := normalizeUserPaletteEntry(entry)
		if err != nil {
			return nil, err
		}
		entriesByCode[normalized.Code] = normalized
	}

	normalizedEntries := make([]PaletteEntry, 0, len(entriesByCode))
	for _, entry := range entriesByCode {
		normalizedEntries = append(normalizedEntries, entry)
	}
	sortPaletteEntries(normalizedEntries)
	return normalizedEntries, nil
}

func normalizeUserPaletteEntry(entry PaletteEntry) (PaletteEntry, error) {
	code := dmccode.Normalize(entry.Code)
	if code == "" {
		return PaletteEntry{}, fmt.Errorf("код цвета не указан")
	}

	hex := normalizeHex(entry.Hex)
	if !hexColor.MatchString(hex) {
		return PaletteEntry{}, fmt.Errorf("HEX для %s должен быть в формате #RRGGBB", code)
	}

	return PaletteEntry{
		Code:   code,
		Name:   strings.TrimSpace(entry.Name),
		Hex:    hex,
		Source: "user",
	}, nil
}

func normalizeHex(hex string) string {
	hex = strings.ToUpper(strings.TrimSpace(hex))
	if hex != "" && !strings.HasPrefix(hex, "#") {
		hex = "#" + hex
	}
	return hex
}

func loadCSVPalette(palette Palette, filename string) error {
	file, err := paletteFS.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return err
	}
	indexes := map[string]int{}
	for i, name := range header {
		indexes[strings.TrimSpace(name)] = i
	}

	required := []string{"Floss#", "Description", "Red", "Green", "Blue", "RGB code"}
	for _, name := range required {
		if _, ok := indexes[name]; !ok {
			return fmt.Errorf("%s: palette column %q not found", filename, name)
		}
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		code := strings.ToUpper(strings.TrimSpace(record[indexes["Floss#"]]))
		if code == "" {
			continue
		}

		hex := strings.ToUpper(strings.TrimSpace(record[indexes["RGB code"]]))
		if len(hex) != 6 {
			red, green, blue, err := parseRGB(record[indexes["Red"]], record[indexes["Green"]], record[indexes["Blue"]])
			if err != nil {
				return fmt.Errorf("%s palette row %q: %w", filename, code, err)
			}
			hex = fmt.Sprintf("%02X%02X%02X", red, green, blue)
		}

		palette[code] = Color{
			Hex:  "#" + hex,
			Name: strings.TrimSpace(record[indexes["Description"]]),
		}
	}

	return nil
}

func lessPaletteCode(left, right string) bool {
	leftNumber, leftOK := firstNumberValue(left)
	rightNumber, rightOK := firstNumberValue(right)
	if leftOK && rightOK && leftNumber != rightNumber {
		return leftNumber < rightNumber
	}
	if leftOK != rightOK {
		return leftOK
	}
	return left < right
}

func sortPaletteEntries(entries []PaletteEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return lessPaletteCode(entries[i].Code, entries[j].Code)
	})
}

func firstNumberValue(code string) (int, bool) {
	for index, r := range code {
		if r < '0' || r > '9' {
			continue
		}

		end := index
		for end < len(code) && code[end] >= '0' && code[end] <= '9' {
			end++
		}

		value, err := strconv.Atoi(code[index:end])
		return value, err == nil
	}
	return 0, false
}

func parseRGB(redValue, greenValue, blueValue string) (int, int, int, error) {
	red, err := strconv.Atoi(strings.TrimSpace(redValue))
	if err != nil {
		return 0, 0, 0, err
	}
	green, err := strconv.Atoi(strings.TrimSpace(greenValue))
	if err != nil {
		return 0, 0, 0, err
	}
	blue, err := strconv.Atoi(strings.TrimSpace(blueValue))
	if err != nil {
		return 0, 0, 0, err
	}
	return red, green, blue, nil
}
