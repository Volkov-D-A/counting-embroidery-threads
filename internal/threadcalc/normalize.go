package threadcalc

import (
	"counting-embroidery-threads/internal/dmc"

	"regexp"
	"strconv"
	"strings"
)

var firstNumber = regexp.MustCompile(`\d+`)
var letterPrefix = regexp.MustCompile(`^[A-Z]+`)

var cyrillicToLatinCodeLetters = map[rune]rune{
	'А': 'A', 'а': 'A',
	'В': 'B', 'в': 'B',
	'Е': 'E', 'е': 'E',
	'Ё': 'E', 'ё': 'E',
	'З': 'Z', 'з': 'Z',
	'К': 'K', 'к': 'K',
	'М': 'M', 'м': 'M',
	'Н': 'H', 'н': 'H',
	'О': 'O', 'о': 'O',
	'Р': 'P', 'р': 'P',
	'С': 'C', 'с': 'C',
	'Т': 'T', 'т': 'T',
	'У': 'Y', 'у': 'Y',
	'Х': 'X', 'х': 'X',
	'І': 'I', 'і': 'I',
	'Ј': 'J', 'ј': 'J',
	'Ѕ': 'S', 'ѕ': 'S',
}

func normalizeCode(code string) string {
	code = baseNormalizeCode(code)
	switch code {
	case "5200":
		return "B5200"
	case "BLANC":
		return "WHITE"
	case "ECRUT", "ECRU":
		return "ECRU"
	}
	if len(code) == 1 {
		if value, err := strconv.Atoi(code); err == nil && value >= 1 && value <= 9 {
			return "0" + code
		}
	}
	return code
}

func shouldShowNormalizationNote(rawCode, normalizedCode string) bool {
	return baseNormalizeCode(rawCode) != normalizedCode
}

func baseNormalizeCode(code string) string {
	code = strings.TrimSpace(code)
	code = strings.ReplaceAll(code, " ", "")
	code = strings.Map(func(r rune) rune {
		if latin, ok := cyrillicToLatinCodeLetters[r]; ok {
			return latin
		}
		return r
	}, code)
	return strings.ToUpper(code)
}

func paletteLookupCode(code string, palette dmc.Palette) string {
	if _, ok := palette[code]; ok {
		return code
	}
	if code == "5200" {
		return "B5200"
	}
	if strings.HasPrefix(code, "B5200") {
		return "B5200"
	}
	if code == "WHITE" || code == "ECRU" {
		return code
	}
	match := firstNumber.FindString(code)
	if match == "5200" {
		return "B5200"
	}
	if match != "" && strings.HasPrefix(code, "E") {
		lightEffectsCode := "E" + match
		if _, ok := palette[lightEffectsCode]; ok {
			return lightEffectsCode
		}
		if _, ok := palette[match]; ok {
			return match
		}
	}
	if match != "" && letterPrefix.MatchString(code) {
		if _, ok := palette[match]; ok {
			return match
		}
	}
	return code
}
