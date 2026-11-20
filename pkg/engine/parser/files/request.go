package files

import (
	"context"

	"github.com/projectdiscovery/katana/pkg/navigation"
	"github.com/projectdiscovery/retryablehttp-go"
)

type visitFunc func(ctx context.Context, URL string) ([]*navigation.Request, error)

type KnownFiles struct {
	parsers    []visitFunc
	kinds      []string
	httpclient *retryablehttp.Client
}

// New returns a new known files parser instance
func New(httpclient *retryablehttp.Client, files string) *KnownFiles {
	parser := &KnownFiles{
		httpclient: httpclient,
	}
	switch files {
	case "robotstxt":
		parser.parsers = append(parser.parsers, (&robotsTxtCrawler{httpclient: httpclient}).Visit)
		parser.kinds = append(parser.kinds, "robotstxt")
	case "sitemapxml":
		parser.parsers = append(parser.parsers, (&sitemapXmlCrawler{httpclient: httpclient}).Visit)
		parser.kinds = append(parser.kinds, "sitemapxml")
	case "feedxml":
		parser.parsers = append(parser.parsers, (&feedXmlCrawler{httpclient: httpclient}).Visit)
		parser.kinds = append(parser.kinds, "feedxml")
	default:
		// "all" (and any unrecognized value, for backward compatibility with
		// callers that do not go through CLI enum validation) runs every parser
		// in a fixed order so crawl output stays deterministic.
		parser.parsers = append(parser.parsers,
			(&robotsTxtCrawler{httpclient: httpclient}).Visit,
			(&sitemapXmlCrawler{httpclient: httpclient}).Visit,
			(&feedXmlCrawler{httpclient: httpclient}).Visit,
		)
		parser.kinds = append(parser.kinds, "robotstxt", "sitemapxml", "feedxml")
	}
	return parser
}

// Request requests all known files with visitors (backward-compatible, no context).
func (k *KnownFiles) Request(URL string) (navigationRequests []*navigation.Request, err error) {
	return k.RequestWithContext(context.Background(), URL)
}

// RequestWithContext requests all known files with context support.
// If ctx is nil, context.Background() is used.
func (k *KnownFiles) RequestWithContext(ctx context.Context, URL string) (navigationRequests []*navigation.Request, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	for _, visitor := range k.parsers {
		navRequests, err := visitor(ctx, URL)
		if err != nil {
			return navigationRequests, err
		}
		navigationRequests = append(navigationRequests, navRequests...)
	}
	return
}
