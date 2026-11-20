package files

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSelectsParsers(t *testing.T) {
	tests := []struct {
		name     string
		files    string
		expected int
	}{
		{"robotstxt only", "robotstxt", 1},
		{"sitemapxml only", "sitemapxml", 1},
		{"feedxml only", "feedxml", 1},
		{"all alias", "all", 3},
		{"empty enables every parser", "", 3},
		{"unknown value falls back to every parser", "something-else", 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			knownFiles := New(nil, test.files)
			require.Len(t, knownFiles.parsers, test.expected)
		})
	}
}

func TestNewAllKeepsDeterministicOrder(t *testing.T) {
	knownFiles := New(nil, "all")
	require.Equal(t, []string{"robotstxt", "sitemapxml", "feedxml"}, knownFiles.kinds)

	// Empty input keeps the same historical default order.
	require.Equal(t, knownFiles.kinds, New(nil, "").kinds)
}
