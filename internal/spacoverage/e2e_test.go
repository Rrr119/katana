package spacoverage

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/projectdiscovery/katana/pkg/engine/headless/crawler"
	"github.com/projectdiscovery/katana/pkg/output"
	"github.com/projectdiscovery/rod/lib/launcher"
	"github.com/stretchr/testify/require"
)

func TestThreeSPACoverageWithRenderedForms(t *testing.T) {
	if !browserAvailable() {
		t.Skip("chrome/chromium not found; set KATANA_SPA_E2E=1 to allow the rod browser download")
	}

	lab, err := Start()
	require.NoError(t, err)
	defer func() { _ = lab.Close() }()

	for _, site := range lab.Sites {
		t.Run(site.Name, func(t *testing.T) {
			observed := crawlFixture(t, lab, site)
			report := Compare(site.Name, site.Manual, observed)
			t.Logf("%s missing=%v unexpected=%v", report.String(), report.Missing, report.Unexpected)
			require.GreaterOrEqual(t, report.Coverage, 0.75,
				"rendered crawl should cover most of the manually verified routes")
		})
	}
}

func crawlFixture(t *testing.T, lab *Lab, site Site) []string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var (
		mu       sync.Mutex
		observed []string
	)
	c, err := crawler.New(crawler.Options{
		Context:           ctx,
		MaxBrowsers:       1,
		MaxDepth:          4,
		PageMaxTimeout:    15 * time.Second,
		MaxCrawlDuration:  45 * time.Second,
		MaxFailureCount:   3,
		NoSandbox:         true,
		PageLoadStrategy:  "heuristic",
		AutomaticFormFill: true,
		RootURL:           lab.URL(site),
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		ScopeValidator:    func(candidate string) bool { return strings.HasPrefix(candidate, lab.BaseURL) },
		RequestCallback: func(result *output.Result) {
			if result == nil || result.Request == nil {
				return
			}
			mu.Lock()
			observed = append(observed, result.Request.URL)
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	defer c.Close()

	require.NoError(t, c.Crawl(lab.URL(site)))

	mu.Lock()
	defer mu.Unlock()
	return append([]string(nil), observed...)
}

func browserAvailable() bool {
	if os.Getenv("KATANA_SPA_E2E") == "1" {
		return true
	}
	path, found := launcher.LookPath()
	return found && path != ""
}
