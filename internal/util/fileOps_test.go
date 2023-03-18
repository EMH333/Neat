package util

import "testing"

func Test_isInteger(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{
			name: "yes",
			s:    "123",
			want: true,
		},
		{
			name: "no",
			s:    "nope",
			want: false,
		},
		{
			name: "maybe so",
			s:    "n123",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInteger(tt.s); got != tt.want {
				t.Errorf("IsInteger() = %v, want %v", got, tt.want)
			}
		})
	}
}
