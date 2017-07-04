package web

import (
	"fmt"
	"time"
)

// twoYears is the maximum HSTS header time, recommended by
// https://hstspreload.org.
const twoYears = 730 * 24 * time.Hour

// hstsStart is the day this code was written.  It is used to determine the
// max-age value for a HSTS header.
var hstsStart = time.Date(2017, time.July, 4, 0, 0, 0, 0, time.UTC)

// HSTSHeader returns the appropriate value for a HSTS header using the time
// elapsed since a set starting date.
func HSTSHeader(now time.Time) string {
	// Values recommended by https://hstspreload.org/.
	ds := []time.Duration{
		5 * time.Minute,
		7 * 24 * time.Hour,
		30 * 24 * time.Hour,
	}

	var age time.Duration
	var preload bool

	for _, d := range ds {
		if now.After(hstsStart.Add(d)) {
			continue
		}

		age = d
		break
	}

	// If no values match, set age to two years and turn on preload flag.
	if age == 0 {
		age = twoYears
		preload = true
	}

	h := fmt.Sprintf("max-age=%d; includeSubdomains", int(age.Seconds()))
	if preload {
		h += "; preload"
	}

	return h
}
