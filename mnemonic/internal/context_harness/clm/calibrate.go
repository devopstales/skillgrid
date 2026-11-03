package clm

// baseFactor is the initial chars/token estimate (the context_harness base).
const baseFactor = 4.0

// BaseFactor exposes the initial chars/token estimate for callers that record
// a calibration baseline before any provider token count is observed.
func BaseFactor() float64 {
	return baseFactor
}

// Calibrate corrects the chars/token factor toward the observed ratio,
// using an exponential moving average (alpha 0.5) to stay stable.
func Calibrate(estimatedChars, actualTokens int, prevFactor float64) float64 {
	if actualTokens == 0 {
		return prevFactor
	}
	observed := float64(estimatedChars) / float64(actualTokens)
	if observed <= 0 {
		return prevFactor
	}
	const alpha = 0.5
	return (1-alpha)*prevFactor + alpha*observed
}
