package simulator

import (
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

var codeUnsafePattern = regexp.MustCompile(`[^A-Z0-9-]+`)

func chargerCode(prefix string, runID string, index int) string {
	normalized := strings.ToUpper(strings.TrimSpace(prefix))
	normalized = codeUnsafePattern.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		normalized = "SIM"
	}
	return normalized + "-" + runID + "-" + leftPad(index, 3)
}

func connectorCode(chargerCode string, number int) string {
	return chargerCode + "-" + leftPad(number, 2)
}

func leftPad(value int, width int) string {
	text := strconv.Itoa(value)
	for len(text) < width {
		text = "0" + text
	}
	return text
}

func powerFor(curve string, tick int, totalTicks int, maxPowerKW float64, rng *rand.Rand) float64 {
	if maxPowerKW <= 0 {
		return 0
	}
	switch curve {
	case LoadCommute:
		return commutePower(tick, totalTicks, maxPowerKW)
	case LoadRandom:
		return round2(maxPowerKW * (0.25 + rng.Float64()*0.7))
	default:
		return round2(maxPowerKW * 0.85)
	}
}

func commutePower(tick int, totalTicks int, maxPowerKW float64) float64 {
	if totalTicks <= 1 {
		return round2(maxPowerKW * 0.75)
	}
	position := float64(tick) / float64(totalTicks-1)
	wave := 0.5 + 0.5*math.Sin((position*math.Pi*2)-math.Pi/2)
	return round2(maxPowerKW * (0.35 + wave*0.6))
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
