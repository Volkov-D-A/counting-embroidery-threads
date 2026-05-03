package dmc

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

//go:embed floss_adrianj.csv floss_2017.csv floss_light_effects.csv
var paletteFS embed.FS

// Palette maps DMC codes to display HEX colours.
type Palette map[string]string

// LoadPalette loads the bundled open DMC palette.
func LoadPalette() (Palette, error) {
	palette := Palette{}
	for _, filename := range []string{"floss_adrianj.csv", "floss_2017.csv", "floss_light_effects.csv"} {
		if err := loadCSVPalette(palette, filename); err != nil {
			return nil, err
		}
	}
	if hex, ok := palette["WHITE"]; ok {
		palette["BLANC"] = hex
	}
	if hex, ok := palette["ECRU"]; ok {
		palette["ECRUT"] = hex
	}

	return palette, nil
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

	required := []string{"Floss#", "Red", "Green", "Blue", "RGB code"}
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

		palette[code] = "#" + hex
	}

	return nil
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
