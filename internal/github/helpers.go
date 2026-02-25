package github

import (
	"fmt"
	"html/template"
	"math"
	"regexp"
	"strconv"
)

var hexColorRe = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)

// sanitizeLabelColor validates a 6-digit hex color string.
// Returns the input if valid, or "cccccc" as a safe fallback.
func sanitizeLabelColor(color string) string {
	if hexColorRe.MatchString(color) {
		return color
	}
	return "cccccc"
}

// computeTextColor returns "#ffffff" or "#000000" based on the relative
// luminance of the given 6-digit hex color, using the WCAG formula.
func computeTextColor(hexColor string) string {
	r, _ := strconv.ParseUint(hexColor[0:2], 16, 8)
	g, _ := strconv.ParseUint(hexColor[2:4], 16, 8)
	b, _ := strconv.ParseUint(hexColor[4:6], 16, 8)

	lr := linearize(float64(r) / 255.0)
	lg := linearize(float64(g) / 255.0)
	lb := linearize(float64(b) / 255.0)

	luminance := 0.2126*lr + 0.7152*lg + 0.0722*lb

	if luminance > 0.179 {
		return "#000000"
	}
	return "#ffffff"
}

func linearize(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// computeLabelStyle returns a safe CSS style string for a label with the given
// hex color. The returned template.CSS value includes both background-color and
// a contrast-appropriate text color.
func computeLabelStyle(color string) template.CSS {
	safe := sanitizeLabelColor(color)
	text := computeTextColor(safe)
	return template.CSS(fmt.Sprintf("background-color: #%s; color: %s", safe, text))
}
