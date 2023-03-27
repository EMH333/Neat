package util

import (
	"ethohampton.com/Neat/internal/types"
	"math/rand"
	"time"
)

const ShortReleaseSeparation = time.Hour * 24

// ShortReleaseTimeVariation up to 300 seconds of variation is added to the release times
// so they don't all appear at the same time after the day has passed
const ShortReleaseTimeVariation = 300

var releaseTimeRandomNumbersGenerator *rand.Rand

// FindNextValidShortReleaseTime find the next release time after a set time
// this makes sure too many shorts aren't coming out at once
func FindNextValidShortReleaseTime(after time.Time, otherShorts []types.Short) time.Time {
	if releaseTimeRandomNumbersGenerator == nil {
		releaseTimeRandomNumbersGenerator = rand.New(rand.NewSource(time.Now().Unix()))
	}

	// find the latest short to be released
	var maxOtherReleaseTime time.Time
	for _, short := range otherShorts {
		if short.ReleaseDate.After(maxOtherReleaseTime) {
			maxOtherReleaseTime = short.ReleaseDate
		}
	}

	// if it matches the constraints of the release after time, then use it
	if maxOtherReleaseTime.Add(ShortReleaseSeparation).After(after) {
		randomSep := time.Second * time.Duration(releaseTimeRandomNumbersGenerator.Intn(ShortReleaseTimeVariation))
		//add the release separation plus a random amount to mix things up
		return maxOtherReleaseTime.Add(ShortReleaseSeparation).Add(randomSep)
	} else {
		// otherwise just return the time as is (maybe there are no pending releases)
		return after
	}
}
