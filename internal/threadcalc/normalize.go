package threadcalc

import (
	"regexp"
	"strconv"
	"strings"
)

var firstNumber = regexp.MustCompile(`\d+`)

func normalizeCode(code string) string {
	code = strings.TrimSpace(code)
	code = strings.ReplaceAll(code, " ", "")
	code = strings.Map(func(r rune) rune {
		switch r {
		case 'А', 'а':
			return 'A'
		case 'В', 'в':
			return 'B'
		case 'Е', 'е':
			return 'E'
		case 'К', 'к':
			return 'K'
		case 'М', 'м':
			return 'M'
		case 'О', 'о':
			return 'O'
		case 'Р', 'р':
			return 'P'
		case 'С', 'с':
			return 'C'
		case 'Т', 'т':
			return 'T'
		case 'Х', 'х':
			return 'X'
		default:
			return r
		}
	}, code)
	code = strings.ToUpper(code)
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

func paletteLookupCode(code string, palette map[string]string) string {
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
		return match
	}
	return code
}
