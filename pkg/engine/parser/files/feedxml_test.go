package files

import (
	"net/http"
	"strings"
	"testing"

	urlutil "github.com/projectdiscovery/utils/url"
	"github.com/stretchr/testify/require"
)

func TestFeedXmlParseReaderRSS(t *testing.T) {
	crawler := &feedXmlCrawler{}

	content := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Example</title>
    <link>https://example.com/</link>
    <item>
      <title>First</title>
      <link>https://example.com/posts/1</link>
    </item>
    <item>
      <title>Second</title>
      <link> /posts/2 </link>
    </item>
  </channel>
</rss>`
	parsed, err := urlutil.Parse("http://localhost/feed.xml")
	require.Nil(t, err)

	navigationRequests, err := crawler.parseReader(strings.NewReader(content), &http.Response{Request: &http.Request{URL: parsed.URL}})
	require.Nil(t, err)

	var urls []string
	for _, navReq := range navigationRequests {
		urls = append(urls, navReq.URL)
		require.Equal(t, "file", navReq.Tag)
		require.Equal(t, "feedxml", navReq.Attribute)
		require.Equal(t, "localhost", navReq.RootHostname)
	}
	require.ElementsMatch(t, urls, []string{
		"https://example.com/",
		"https://example.com/posts/1",
		"http://localhost/posts/2",
	}, "could not get correct elements")
}

func TestFeedXmlParseReaderAtom(t *testing.T) {
	crawler := &feedXmlCrawler{}

	content := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Example Atom</title>
  <link href="https://example.com/"/>
  <entry>
    <title>Entry one</title>
    <link rel="alternate" href="https://example.com/entries/1"/>
  </entry>
  <entry>
    <title>Entry two</title>
    <link rel="alternate" href="/entries/2"/>
    <link rel="self" href="https://example.com/entries/2/atom"/>
  </entry>
</feed>`
	parsed, err := urlutil.Parse("http://localhost/feed.xml")
	require.Nil(t, err)

	navigationRequests, err := crawler.parseReader(strings.NewReader(content), &http.Response{Request: &http.Request{URL: parsed.URL}})
	require.Nil(t, err)

	var urls []string
	for _, navReq := range navigationRequests {
		urls = append(urls, navReq.URL)
		require.Equal(t, "localhost", navReq.RootHostname)
	}
	require.ElementsMatch(t, urls, []string{
		"http://localhost/entries/2",
		"https://example.com/entries/1",
		"https://example.com/entries/2/atom",
	}, "could not get correct elements")
}

func TestFeedXmlParseReaderInvalidXML(t *testing.T) {
	crawler := &feedXmlCrawler{}
	parsed, err := urlutil.Parse("http://localhost/feed.xml")
	require.Nil(t, err)

	_, err = crawler.parseReader(strings.NewReader("not xml at all"), &http.Response{Request: &http.Request{URL: parsed.URL}})
	require.Error(t, err)
}
