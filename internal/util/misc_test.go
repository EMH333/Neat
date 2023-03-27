package util

import (
	"testing"
)

func BenchmarkGenerateHTML(b *testing.B) {
	testContent := RandString(1000) + "\n\n\n\n" + RandString(1000) + "\n\n\n\n" + RandString(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateHTML(testContent)
	}
}
