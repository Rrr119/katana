package testutils

import (
	"strings"

	"github.com/projectdiscovery/utils/errkit"
)

type TestCase struct {
	Name        string
	Target      string
	Args        string
	Expected    []string
	CompareFunc func(target string, got []string) error
}

var TestCases = []TestCase{
	{
		Name:     "Standard Mode Crawl",
		Target:   "", // filled at runtime with local server URL
		Args:     "-depth 2 -silent",
		Expected: nil,
		CompareFunc: func(target string, got []string) error {
			if len(got) < 3 {
				return errkit.Newf("expected at least 3 URLs, got %d: %v", len(got), got)
			}
			expectedPaths := []string{"/about", "/contact", "/blog"}
			for _, path := range expectedPaths {
				found := false
				for _, u := range got {
					if strings.Contains(u, path) {
						found = true
						break
					}
				}
				if !found {
					return errkit.Newf("expected %s in output, got: %s", path, strings.Join(got, "\n"))
				}
			}
			return nil
		},
	},
	{
		Name:     "Standard Mode Depth 3",
		Target:   "",
		Args:     "-depth 3 -silent",
		Expected: nil,
		CompareFunc: func(target string, got []string) error {
			expectedPaths := []string{"/blog/post-1", "/blog/post-2", "/team"}
			for _, path := range expectedPaths {
				found := false
				for _, u := range got {
					if strings.Contains(u, path) {
						found = true
						break
					}
				}
				if !found {
					return errkit.Newf("depth 3 should discover %s, got: %s", path, strings.Join(got, "\n"))
				}
			}
			return nil
		},
	},
	{
		Name:     "Headless Browser Crawl",
		Target:   "",
		Args:     "-headless -no-incognito -depth 2 -silent -no-sandbox",
		Expected: nil,
		CompareFunc: func(target string, got []string) error {
			if len(got) < 2 {
				return errkit.Newf("expected at least 2 URLs in headless mode, got %d: %v", len(got), got)
			}
			for _, res := range got {
				if strings.Contains(res, target) {
					return nil
				}
			}
			return errkit.Newf("expected %v target in output, but got %v", target, strings.Join(got, "\n"))
		},
	},
	{
		Name:   "Known Files Feed XML Parser",
		Target: "",
		Args:   "-kf feedxml -depth 3 -silent",
		CompareFunc: func(target string, got []string) error {
			joined := strings.Join(got, "\n")
			if !strings.Contains(joined, "/feed/article") {
				return errkit.Newf("feed parser should discover /feed/article, got: %s", joined)
			}
			if strings.Contains(joined, "external.example.net") {
				return errkit.Newf("scope control must drop the external feed link, got: %s", joined)
			}
			return nil
		},
	},
	{
		Name:   "Known Files All Parsers",
		Target: "",
		Args:   "-kf all -depth 3 -silent",
		CompareFunc: func(target string, got []string) error {
			if len(got) < 3 {
				return errkit.Newf("expected combined known-file crawlers to still produce output, got %d: %v", len(got), got)
			}
			joined := strings.Join(got, "\n")
			if strings.Contains(joined, "external.example.net") {
				return errkit.Newf("out-of-scope feed link leaked with -kf all, got: %s", joined)
			}
			return nil
		},
	},
}
