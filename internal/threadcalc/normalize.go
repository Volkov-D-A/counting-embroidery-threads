package threadcalc

import (
	"counting-embroidery-threads/internal/dmc"
	"counting-embroidery-threads/internal/dmccode"

	"regexp"
	"strings"
)

var firstNumber = regexp.MustCompile(`\d+`)
var letterPrefix = regexp.MustCompile(`^[A-Z]+`)

func normalizeCode(code string) string {
	return dmccode.Normalize(code)
}

func shouldShowNormalizationNote(rawCode, normalizedCode string) bool {
	return dmccode.ShouldShowNormalizationNote(rawCode, normalizedCode)
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
