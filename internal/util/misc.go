package util

import (
	"html/template"
	"math/rand"
	"strconv"
	"strings"
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

func GenerateHTML(content string) template.HTML {
	var output strings.Builder

	output.Write([]byte("<p>"))
	output.WriteString(strings.ReplaceAll(content, "\r\n\r\n", "</p><p>"))
	output.Write([]byte("</p>"))

	return template.HTML(output.String())
}
