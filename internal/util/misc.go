package util

import (
	"math/rand"
	"strconv"
	"time"
)

// based on Open Location Code Base20 alphabet, add E, L, S and T because I felt like it
const letterBytes = "23456789CEFGHJLMPQRSTVWX"

var randomNumbersGenerator *rand.Rand

func RandString(n int) string {
	// make sure we have a random source of numbers
	if randomNumbersGenerator == nil {
		randomNumbersGenerator = rand.New(rand.NewSource(time.Now().Unix()))
	}

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[randomNumbersGenerator.Intn(len(letterBytes))]
	}
	return string(b)
}

func IsInteger(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}
