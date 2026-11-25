package spacoverage

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeRoute(t *testing.T) {
	tests := []struct {
		raw  string
		want string
		ok   bool
	}{
		{raw: "http://example.com/atlas/api/overview?x=1", want: "/atlas/api/overview", ok: true},
		{raw: "/orbit/projects/1/", want: "/orbit/projects/1", ok: true},
		{raw: "pulse/items/1", want: "/pulse/items/1", ok: true},
		{raw: "https://example.com", want: "/", ok: true},
		{raw: "mailto:a@example.com"},
		{raw: "javascript:void(0)"},
		{raw: "   "},
	}
	for _, tt := range tests {
		got, ok := NormalizeRoute(tt.raw)
		require.Equal(t, tt.ok, ok, tt.raw)
		require.Equal(t, tt.want, got, tt.raw)
	}
}

func TestCompareReportsManualCoverage(t *testing.T) {
	report := Compare("atlas", []string{
		"/atlas/",
		"/atlas/api/overview",
		"/atlas/api/reports",
		"/atlas/api/settings",
	}, []string{
		"http://127.0.0.1:1234/atlas/",
		"http://127.0.0.1:1234/atlas/api/overview?from=spa",
		"http://127.0.0.1:1234/atlas/api/reports/",
		"http://127.0.0.1:1234/static/app.js",
	})

	require.Equal(t, 4, report.Expected)
	require.Equal(t, 3, report.Covered)
	require.InDelta(t, 0.75, report.Coverage, 0.0001)
	require.Equal(t, []string{"/atlas/api/settings"}, report.Missing)
	require.Empty(t, report.Unexpected, "static assets are not crawl routes")
}

func TestAggregate(t *testing.T) {
	overall := Aggregate("headless", []Report{
		{Site: "a", Expected: 2, Covered: 1, Missing: []string{"/a2"}},
		{Site: "b", Expected: 2, Covered: 2},
	})
	require.Equal(t, 4, overall.Expected)
	require.Equal(t, 3, overall.Covered)
	require.InDelta(t, 0.75, overall.Coverage, 0.0001)
	require.Equal(t, []string{"a /a2"}, overall.Missing)
}

func TestFixturesExposeManualRoutes(t *testing.T) {
	lab, err := Start()
	require.NoError(t, err)
	defer func() { _ = lab.Close() }()

	require.Len(t, lab.Sites, 3)
	client := &http.Client{Timeout: 5 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, site := range lab.Sites {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, lab.URL(site), nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Contains(t, string(body), "<script>", site.Name)
		require.NotEmpty(t, site.Manual, site.Name)
	}
}
