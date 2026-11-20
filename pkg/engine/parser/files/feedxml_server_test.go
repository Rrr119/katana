package files

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	retryablehttp "github.com/projectdiscovery/retryablehttp-go"
	"github.com/stretchr/testify/require"
)

func TestFeedXmlVisit(t *testing.T) {
	var requestedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <item><link>` + r.Host + ` is not a url, ignore me</link></item>
    <item><link>/feed/found</link></item>
  </channel>
</rss>`))
	}))
	defer srv.Close()

	httpClient := &http.Client{Timeout: 5 * time.Second}
	client := retryablehttp.NewWithHTTPClient(httpClient, retryablehttp.DefaultOptionsSpraying)
	crawler := &feedXmlCrawler{httpclient: client}

	requests, err := crawler.Visit(context.Background(), srv.URL)
	require.Nil(t, err)
	require.Equal(t, feedPath, requestedPath, "feed crawler must request the well-known feed path")
	require.Len(t, requests, 2)

	require.Equal(t, srv.URL+"/feed/found", requests[1].URL)
	require.Equal(t, "file", requests[1].Tag)
	require.Equal(t, "feedxml", requests[1].Attribute)
	require.Equal(t, "127.0.0.1", requests[1].RootHostname)
	require.Equal(t, 2, requests[1].Depth)
}

func TestKnownFilesFeedXmlSelection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case feedPath:
			w.Header().Set("Content-Type", "application/atom+xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry><link href="/atom/found"/></entry>
</feed>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	httpClient := &http.Client{Timeout: 5 * time.Second}
	client := retryablehttp.NewWithHTTPClient(httpClient, retryablehttp.DefaultOptionsSpraying)

	knownFiles := New(client, "feedxml")
	requests, err := knownFiles.RequestWithContext(context.Background(), srv.URL)
	require.Nil(t, err)
	require.Len(t, requests, 1)
	require.Equal(t, srv.URL+"/atom/found", requests[0].URL)
	require.Equal(t, "feedxml", requests[0].Attribute)
}
