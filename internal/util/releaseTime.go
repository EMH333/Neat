package util

import (
	"ethohampton.com/Neat/internal/types"
	"time"
)

const ShortReleaseSeparation = time.Hour * 24

// FindNextValidShortReleaseTime find the next release time after a set time
// this makes sure too many shorts aren't coming out at once
func FindNextValidShortReleaseTime(after time.Time, otherShorts []types.Short) time.Time {
	// find the latest short to be released
	var maxOtherReleaseTime time.Time
	for _, short := range otherShorts {
		if short.ReleaseDate.After(maxOtherReleaseTime) {
			maxOtherReleaseTime = short.ReleaseDate
		}
	}

	// if it matches the constraints of the release after time, then use it
	if maxOtherReleaseTime.After(after) {
		return maxOtherReleaseTime.Add(ShortReleaseSeparation)
	} else {
		// otherwise just return the time as is (maybe there are no pending releases)
		return after
	}
}
