package main

import (
	"testing"
)

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		{"large integer as float64", float64(95893899), "95893899"},
		{"small integer as float64", float64(42), "42"},
		{"zero", float64(0), "0"},
		{"negative integer as float64", float64(-100), "-100"},
		{"actual float", float64(3.14), "3.14"},
		{"string value", "hello", "hello"},
		{"bool value", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.val)
			if got != tt.want {
				t.Errorf("formatValue(%v) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
}
