package settlement

import "time"

const eodHour = 18
const eodMinute = 30
const eodLocation = "America/Chicago"

// IsAfterEOD returns true if the given time is after the EOD cutoff (6:30 PM CT).
func IsAfterEOD(t time.Time) bool {
	loc := loadEODLocation()
	local := t.In(loc)
	cutoff := time.Date(local.Year(), local.Month(), local.Day(), eodHour, eodMinute, 0, 0, loc)
	return local.After(cutoff)
}

// TriggerSettlementDate returns the local CT business day for a settlement trigger.
func TriggerSettlementDate(t time.Time) time.Time {
	loc := loadEODLocation()
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

// SettlementDateForDeposit returns the settlement business day for a deposit timestamp.
// Deposits after cutoff are assigned to the next business day.
func SettlementDateForDeposit(t time.Time) time.Time {
	loc := loadEODLocation()
	local := t.In(loc)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	if IsAfterEOD(local) {
		return nextBusinessDay(day)
	}
	return day
}

func nextBusinessDay(day time.Time) time.Time {
	next := day.AddDate(0, 0, 1)
	for next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func loadEODLocation() *time.Location {
	loc, err := time.LoadLocation(eodLocation)
	if err != nil {
		// Fallback to UTC-6 offset if timezone data unavailable
		return time.FixedZone("CT", -6*3600)
	}
	return loc
}
