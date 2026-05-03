package dmccode

import (
	"strconv"
	"strings"
)

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

// Normalize converts a user or report DMC code to the canonical app form.
func Normalize(code string) string {
	code = BaseNormalize(code)
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

// ShouldShowNormalizationNote reports whether normalization is meaningful to a user.
func ShouldShowNormalizationNote(rawCode, normalizedCode string) bool {
	return BaseNormalize(rawCode) != normalizedCode
}

// BaseNormalize only performs technical cleanup: spaces, case and look-alike letters.
func BaseNormalize(code string) string {
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
