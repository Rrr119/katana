package files

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/katana/pkg/navigation"
	"github.com/projectdiscovery/katana/pkg/utils"
	"github.com/projectdiscovery/retryablehttp-go"
	"github.com/projectdiscovery/utils/errkit"
)

// feedPath is the well-known location requested by the feed crawler.
const feedPath = "/feed.xml"

type feedXmlCrawler struct {
	httpclient *retryablehttp.Client
}

// Visit visits the provided URL and parses its RSS/Atom feed.
func (f *feedXmlCrawler) Visit(ctx context.Context, URL string) (navigationRequests []*navigation.Request, err error) {
	URL = strings.TrimSuffix(URL, "/")
	requestURL := fmt.Sprintf("%s%s", URL, feedPath)
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "feedcrawler: could not create request")
	}
	req.Header.Set("User-Agent", utils.WebUserAgent())
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml")

	resp, err := f.httpclient.Do(req)
	if err != nil {
		return nil, errkit.Wrap(err, "feedcrawler: could not do request")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			gologger.Error().Msgf("Error closing response body: %v\n", err)
		}
	}()

	navigationRequests, err = f.parseReader(resp.Body, resp)
	if err != nil {
		return nil, errkit.Wrap(err, "feedcrawler: could not parse feed")
	}
	return
}

// feedStruct models the union of an RSS 2.0 and an Atom 1.0 document. Both
// formats use <link> to carry the URLs worth crawling: RSS item/link is plain
// text, Atom link is an href attribute, hence the two fields.
type feedStruct struct {
	Links   []feedLink  `xml:"channel>link"`
	Items   []feedItem  `xml:"channel>item"`
	Entries []feedEntry `xml:"entry"`
}

type feedItem struct {
	Link string `xml:"link"`
}

type feedEntry struct {
	Links []feedLink `xml:"link"`
}

type feedLink struct {
	Href string `xml:"href,attr"`
	Text string `xml:",chardata"`
}

func (f *feedXmlCrawler) parseReader(reader io.Reader, resp *http.Response) (navigationRequests []*navigation.Request, err error) {
	feed := feedStruct{}
	if err := xml.NewDecoder(reader).Decode(&feed); err != nil {
		return nil, errkit.Wrap(err, "feedcrawler: could not decode xml")
	}

	rootHostname := resp.Request.URL.Hostname()
	navResp := &navigation.Response{
		Depth:        2,
		Resp:         resp,
		RootHostname: rootHostname,
		StatusCode:   resp.StatusCode,
		Headers:      utils.FlattenHeaders(resp.Header),
	}

	addLink := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		navigationRequests = append(navigationRequests, navigation.NewNavigationRequestURLFromResponse(raw, resp.Request.URL.String(), "file", "feedxml", navResp))
	}

	// RSS channel self link (plain text element).
	for _, link := range feed.Links {
		addLink(link.Text)
	}
	// RSS items.
	for _, item := range feed.Items {
		addLink(item.Link)
	}
	// Atom entries, including rel="self"/"alternate" href links.
	for _, entry := range feed.Entries {
		for _, link := range entry.Links {
			addLink(link.Href)
		}
	}
	return
}
