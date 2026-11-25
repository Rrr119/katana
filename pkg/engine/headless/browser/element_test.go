package browser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldStopLazyScroll(t *testing.T) {
	tests := []struct {
		name   string
		before pageScrollMetrics
		after  pageScrollMetrics
		stop   bool
	}{
		{
			name:   "stable bottom",
			before: pageScrollMetrics{Top: 500, Height: 1000, Viewport: 500},
			after:  pageScrollMetrics{Top: 500, Height: 1000, Viewport: 500},
			stop:   true,
		},
		{
			name:   "lazy content grew",
			before: pageScrollMetrics{Top: 500, Height: 1000, Viewport: 500},
			after:  pageScrollMetrics{Top: 500, Height: 1500, Viewport: 500},
			stop:   false,
		},
		{
			name:   "not at bottom",
			before: pageScrollMetrics{Top: 0, Height: 2000, Viewport: 500},
			after:  pageScrollMetrics{Top: 436, Height: 2000, Viewport: 500},
			stop:   false,
		},
		{
			name:   "subpixel tolerance",
			before: pageScrollMetrics{Top: 499, Height: 1000, Viewport: 500},
			after:  pageScrollMetrics{Top: 499.5, Height: 1000.5, Viewport: 500},
			stop:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.stop, shouldStopLazyScroll(test.before, test.after))
		})
	}
}

func TestIsRouteNavigation(t *testing.T) {
	for _, source := range []string{"history.pushState", "history.replaceState", "hashchange", "window.open"} {
		require.True(t, isRouteNavigation(source), source)
	}
	for _, source := range []string{"fetch", "websocket", "eventsource", ""} {
		require.False(t, isRouteNavigation(source), source)
	}
}

func TestShouldFollowNavigatedLink(t *testing.T) {
	ordinary := &BrowserPage{launcher: &Launcher{}}
	require.True(t, ordinary.shouldFollowNavigatedLink("history.pushState"))
	require.False(t, ordinary.shouldFollowNavigatedLink("fetch"))

	interactive := &BrowserPage{launcher: &Launcher{opts: LauncherOptions{Interactive: true}}}
	require.True(t, interactive.shouldFollowNavigatedLink("fetch"))
	require.False(t, interactive.shouldFollowNavigatedLink("unknown"))
}
