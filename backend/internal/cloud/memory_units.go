package cloud

import (
	"math"
	"strconv"
	"strings"
)

func formatMemoryGB(value int64) string {
	if value <= 0 {
		return ""
	}
	return strconv.FormatInt(value, 10) + "G"
}

func formatMemoryMBToGB(value int64) string {
	if value <= 0 {
		return ""
	}
	return formatMemoryFloatGB(float64(value) / 1024)
}

func formatMemoryFloatGB(value float64) string {
	if value <= 0 {
		return ""
	}
	rounded := math.Round(value)
	if math.Abs(value-rounded) < 1e-9 {
		return strconv.FormatInt(int64(rounded), 10) + "G"
	}
	text := strconv.FormatFloat(value, 'f', 2, 64)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	return text + "G"
}
