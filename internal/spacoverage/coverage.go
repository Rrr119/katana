package spacoverage

import (
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"
)

// Report compares discovered routes with the manually verified manifest.
type Report struct {
	Site       string   `json:"site"`
	Mode       string   `json:"mode,omitempty"`
	Expected   int      `json:"expected"`
	Covered    int      `json:"covered"`
	Coverage   float64  `json:"coverage"`
	Missing    []string `json:"missing,omitempty"`
	Unexpected []string `json:"unexpected,omitempty"`
}

// Compare matches observed URLs to expected paths. Query strings and fragments
// are ignored: a SPA route and its data request share the same benchmark route.
func Compare(site string, expected, observed []string) Report {
	expectedRoutes := normalizeAll(expected)
	observedRoutes := normalizeAll(observed)

	covered := 0
	var missing []string
	for route := range expectedRoutes {
		if _, ok := observedRoutes[route]; ok {
			covered++
		} else {
			missing = append(missing, route)
		}
	}

	var unexpected []string
	for route := range observedRoutes {
		if _, ok := expectedRoutes[route]; ok {
			continue
		}
		if isAsset(route) {
			continue
		}
		unexpected = append(unexpected, route)
	}
	sort.Strings(missing)
	sort.Strings(unexpected)

	coverage := 0.0
	if len(expectedRoutes) > 0 {
		coverage = float64(covered) / float64(len(expectedRoutes))
	}
	return Report{
		Site:       site,
		Expected:   len(expectedRoutes),
		Covered:    covered,
		Coverage:   coverage,
		Missing:    missing,
		Unexpected: unexpected,
	}
}

// NormalizeRoute returns the path form used for comparisons.
func NormalizeRoute(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "data:") || strings.HasPrefix(raw, "blob:") {
		return "", false
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.Scheme != "" {
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return "", false
		}
		return routePath(parsed)
	}
	if !strings.Contains(raw, "://") {
		raw = "http://benchmark.invalid" + ensureLeadingSlash(raw)
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}
	return routePath(parsed)
}

func routePath(parsed *url.URL) (string, bool) {
	route := parsed.Path
	if route == "" {
		route = "/"
	}
	if route != "/" {
		route = strings.TrimSuffix(route, "/")
	}
	return route, true
}

func normalizeAll(raw []string) map[string]struct{} {
	routes := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		route, ok := NormalizeRoute(item)
		if ok {
			routes[route] = struct{}{}
		}
	}
	return routes
}

func ensureLeadingSlash(value string) string {
	if strings.HasPrefix(value, "/") {
		return value
	}
	return "/" + value
}

func isAsset(route string) bool {
	if route == "/favicon.ico" {
		return true
	}
	switch strings.ToLower(path.Ext(route)) {
	case ".js", ".mjs", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".map":
		return true
	default:
		return false
	}
}

// Aggregate merges per-site reports into an overall report.
func Aggregate(mode string, reports []Report) Report {
	overall := Report{Site: "all", Mode: mode}
	for _, report := range reports {
		overall.Expected += report.Expected
		overall.Covered += report.Covered
		for _, route := range report.Missing {
			overall.Missing = append(overall.Missing, report.Site+" "+route)
		}
	}
	if overall.Expected > 0 {
		overall.Coverage = float64(overall.Covered) / float64(overall.Expected)
	}
	sort.Strings(overall.Missing)
	return overall
}

func (r Report) String() string {
	return fmt.Sprintf("%s: %d/%d (%.0f%%)", r.Site, r.Covered, r.Expected, r.Coverage*100)
}
