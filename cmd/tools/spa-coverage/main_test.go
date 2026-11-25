package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractURLs(t *testing.T) {
	output := "http://127.0.0.1:8080/atlas/\n[source] http://127.0.0.1:8080/atlas/api/overview\n/pulse/items/1\nnot a url\n"
	require.Equal(t, []string{
		"http://127.0.0.1:8080/atlas/",
		"[source] http://127.0.0.1:8080/atlas/api/overview",
		"/pulse/items/1",
	}, extractURLs(output))
}

func TestSelectModes(t *testing.T) {
	selected, err := selectModes("headless, hybrid,headless")
	require.NoError(t, err)
	require.Len(t, selected, 2)
	require.Equal(t, "headless", selected[0].name)
	require.Equal(t, "hybrid", selected[1].name)

	_, err = selectModes("")
	require.Error(t, err)
	_, err = selectModes("teleport")
	require.ErrorContains(t, err, "unknown mode")
}
