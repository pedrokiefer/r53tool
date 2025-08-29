package dns

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeDomain(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"example.com", "example.com."},
		{"example.com.", "example.com."},
		{"", "."},
	}
	for _, c := range cases {
		got := NormalizeDomain(c.in)
		require.Equal(t, c.out, got, "NormalizeDomain(%q)", c.in)
	}
}

func TestDenormalizeDomain(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"example.com.", "example.com"},
		{"example.com", "example.com"},
		{".", ""},
		{"", ""},
	}
	for _, c := range cases {
		got := DenormalizeDomain(c.in)
		require.Equal(t, c.out, got, "DenormalizeDomain(%q)", c.in)
	}
}
