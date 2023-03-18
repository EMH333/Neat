package util

import (
	"math/rand"
	"strconv"
)

// based on Open Location Code Base20 alphabet, add E, L, S and T because I felt like it
const letterBytes = "23456789CEFGHJLMPQRSTVWX"

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func IsInteger(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}
