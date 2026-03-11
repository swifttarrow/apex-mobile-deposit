package settlement

import "time"

const eodHour = 18
const eodMinute = 30
const eodLocation = "America/Chicago"

// IsAfterEOD returns true if the given time is after the EOD cutoff (6:30 PM CT).
func IsAfterEOD(t time.Time) bool {
	loc, err := time.LoadLocation(eodLocation)
	if err != nil {
		// Fallback to UTC-6 offset if timezone data unavailable
		loc = time.FixedZone("CT", -6*3600)
	}
	local := t.In(loc)
	cutoff := time.Date(local.Year(), local.Month(), local.Day(), eodHour, eodMinute, 0, 0, loc)
	return local.After(cutoff)
}
