// spa-coverage measures katana's rendered coverage on three deterministic SPA
// fixtures and compares each crawl mode with a manually verified route list.
//
//	go run ./cmd/tools/spa-coverage -katana ./katana
//	go run ./cmd/tools/spa-coverage -modes headless,hybrid -json coverage.json
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/projectdiscovery/katana/internal/spacoverage"
)

type mode struct {
	name string
	args []string
}

var modes = map[string]mode{
	"standard": {name: "standard", args: []string{"-aff"}},
	"headless": {name: "headless", args: []string{"-hl", "-nos", "-aff"}},
	"hybrid":   {name: "hybrid", args: []string{"-hh", "-nos", "-aff"}},
}

type output struct {
	MinCoverage float64                         `json:"min_coverage"`
	Reports     map[string][]spacoverage.Report `json:"reports"`
	Overall     map[string]spacoverage.Report   `json:"overall"`
}

func main() {
	katana := flag.String("katana", "katana", "katana binary to execute")
	modeList := flag.String("modes", "standard,headless,hybrid", "comma separated crawl modes")
	minCoverage := flag.Float64("min-coverage", 0.75, "minimum overall coverage for the headless mode")
	crawlSeconds := flag.Int("crawl-seconds", 90, "crawl duration per site and mode")
	jsonPath := flag.String("json", "", "write the full report as json")
	flag.Parse()

	selected, err := selectModes(*modeList)
	if err != nil {
		fatal(err)
	}
	lab, err := spacoverage.Start()
	if err != nil {
		fatal(err)
	}
	defer func() { _ = lab.Close() }()

	result := output{
		MinCoverage: *minCoverage,
		Reports:     make(map[string][]spacoverage.Report, len(selected)),
		Overall:     make(map[string]spacoverage.Report, len(selected)),
	}
	for _, run := range selected {
		for _, site := range lab.Sites {
			report, err := crawlSite(*katana, run, lab.URL(site), site, *crawlSeconds)
			if err != nil {
				fatal(fmt.Errorf("%s/%s: %w", site.Name, run.name, err))
			}
			result.Reports[run.name] = append(result.Reports[run.name], report)
		}
		result.Overall[run.name] = spacoverage.Aggregate(run.name, result.Reports[run.name])
	}

	printReport(result)
	if *jsonPath != "" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fatal(err)
		}
		data = append(data, '\n')
		if err := os.WriteFile(*jsonPath, data, 0o644); err != nil {
			fatal(err)
		}
	}

	headless, ok := result.Overall["headless"]
	if ok && headless.Coverage < *minCoverage {
		os.Exit(1)
	}
}

func crawlSite(katana string, run mode, target string, site spacoverage.Site, seconds int) (spacoverage.Report, error) {
	args := []string{
		"-u", target,
		"-silent",
		"-disable-update-check",
		"-d", "4",
		"-ct", fmt.Sprint(seconds),
	}
	args = append(args, run.args...)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds+30)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, katana, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return spacoverage.Report{}, fmt.Errorf("katana %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}

	report := spacoverage.Compare(site.Name, site.Manual, extractURLs(stdout.String()))
	report.Mode = run.name
	return report, nil
}

func extractURLs(output string) []string {
	var urls []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.Contains(line, "://") {
			urls = append(urls, line)
			continue
		}
		// Accept path-only output from templates or custom runners.
		if strings.HasPrefix(line, "/") {
			urls = append(urls, line)
		}
	}
	return urls
}

func selectModes(value string) ([]mode, error) {
	seen := map[string]struct{}{}
	var selected []mode
	for _, name := range strings.Split(value, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		run, ok := modes[name]
		if !ok {
			return nil, fmt.Errorf("unknown mode %q (want standard, headless or hybrid)", name)
		}
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}
		selected = append(selected, run)
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("at least one mode is required")
	}
	return selected, nil
}

func printReport(result output) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "mode\tsite\tcovered\texpected\tcoverage\tmissing")
	for _, name := range sortedModeNames(result.Reports) {
		for _, report := range result.Reports[name] {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%.0f%%\t%s\n",
				name, report.Site, report.Covered, report.Expected, report.Coverage*100, strings.Join(report.Missing, ","))
		}
		overall := result.Overall[name]
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%.0f%%\t%s\n",
			name, "all", overall.Covered, overall.Expected, overall.Coverage*100, strings.Join(overall.Missing, ","))
	}
	_ = w.Flush()
}

func sortedModeNames(reports map[string][]spacoverage.Report) []string {
	names := make([]string, 0, len(reports))
	for name := range reports {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, "spa-coverage:", err)
	os.Exit(1)
}
