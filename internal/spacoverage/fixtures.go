// Package spacoverage provides three deterministic single-page applications
// and the manual route manifest used to measure katana's rendered coverage.
package spacoverage

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Site describes one benchmark application.
type Site struct {
	Name   string
	Path   string
	Manual []string
}

// Lab is a running set of SPA fixtures.
type Lab struct {
	BaseURL string
	Sites   []Site

	server   *http.Server
	listener net.Listener
}

// Start serves the three fixtures on a random loopback port.
func Start() (*Lab, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	baseURL := "http://" + listener.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/atlas/", atlasHandler)
	mux.HandleFunc("/orbit/", orbitHandler)
	mux.HandleFunc("/pulse/", pulseHandler)

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		_ = server.Serve(listener)
	}()

	return &Lab{
		BaseURL: baseURL,
		Sites: []Site{
			{
				Name: "atlas-hash",
				Path: "/atlas/",
				Manual: []string{
					"/atlas/",
					"/atlas/api/overview",
					"/atlas/api/reports",
					"/atlas/api/settings",
				},
			},
			{
				Name: "orbit-history",
				Path: "/orbit/",
				Manual: []string{
					"/orbit/",
					"/orbit/projects/1",
					"/orbit/projects/2",
					"/orbit/team",
					"/orbit/api/projects/1",
					"/orbit/api/projects/2",
					"/orbit/api/team",
				},
			},
			{
				Name: "pulse-form",
				Path: "/pulse/",
				Manual: []string{
					"/pulse/",
					"/pulse/api/search",
					"/pulse/items/1",
					"/pulse/items/2",
					"/pulse/api/items/1",
					"/pulse/api/items/2",
				},
			},
		},
		server:   server,
		listener: listener,
	}, nil
}

// Close stops the fixture server.
func (l *Lab) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return l.server.Shutdown(ctx)
}

func (l *Lab) URL(site Site) string {
	return l.BaseURL + site.Path
}

func writeHTML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(body))
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(body))
}

func atlasHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/atlas/api/overview":
		writeJSON(w, `{"route":"overview"}`)
	case "/atlas/api/reports":
		writeJSON(w, `{"route":"reports"}`)
	case "/atlas/api/settings":
		writeJSON(w, `{"route":"settings"}`)
	default:
		writeHTML(w, atlasShell)
	}
}

const atlasShell = `<!doctype html>
<html>
<head><title>Atlas</title></head>
<body>
<nav id="nav"></nav>
<main id="view">loading</main>
<script>
  const routes = ["#/overview", "#/reports", "#/settings"];
  const nav = document.getElementById("nav");
  nav.innerHTML = routes.map(function (route) {
    return '<a href="' + route + '">' + route.slice(2) + '</a>';
  }).join(" ");

  function render() {
    const route = (location.hash || "#/overview").slice(2);
    document.getElementById("view").textContent = "loading " + route;
    fetch("/atlas/api/" + route).then(function (response) {
      return response.json();
    }).then(function (data) {
      document.getElementById("view").textContent = "loaded " + data.route;
    });
  }
  window.addEventListener("hashchange", render);
  render();
</script>
</body>
</html>`

func orbitHandler(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/orbit/api/") {
		writeJSON(w, fmt.Sprintf(`{"route":%q}`, strings.TrimPrefix(r.URL.Path, "/orbit/api/")))
		return
	}
	writeHTML(w, orbitShell)
}

const orbitShell = `<!doctype html>
<html>
<head><title>Orbit</title></head>
<body>
<nav id="nav"></nav>
<main id="view">loading</main>
<script>
  const routes = [
    ["/orbit/projects/1", "Project one"],
    ["/orbit/projects/2", "Project two"],
    ["/orbit/team", "Team"]
  ];
  const nav = document.getElementById("nav");
  nav.innerHTML = routes.map(function (route) {
    return '<button data-path="' + route[0] + '">' + route[1] + '</button>';
  }).join(" ");

  function load(path) {
    const view = document.getElementById("view");
    view.textContent = "loading " + path;
    fetch("/orbit/api" + path.slice("/orbit".length)).then(function (response) {
      return response.json();
    }).then(function (data) {
      view.textContent = "loaded " + data.route;
    });
  }

  function route(path) {
    history.pushState({}, "", path);
    load(path);
  }

  nav.addEventListener("click", function (event) {
    const path = event.target.getAttribute("data-path");
    if (path) route(path);
  });
  load(location.pathname);
</script>
</body>
</html>`

func pulseHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/pulse/api/search":
		writeJSON(w, `{"items":[1,2]}`)
	case strings.HasPrefix(r.URL.Path, "/pulse/api/items/"):
		writeJSON(w, fmt.Sprintf(`{"route":%q}`, strings.TrimPrefix(r.URL.Path, "/pulse/api/")))
	case strings.HasPrefix(r.URL.Path, "/pulse/items/"):
		writeHTML(w, pulseShell)
	default:
		writeHTML(w, pulseShell)
	}
}

// The search form deliberately has no submit button. It is rendered by the
// page script and submitted by the browser engine's Enter/requestSubmit path.
const pulseShell = `<!doctype html>
<html>
<head><title>Pulse</title></head>
<body>
<main id="app"></main>
<script>
  const app = document.getElementById("app");
  app.innerHTML = '<form id="search" action="/pulse/search" method="get">' +
    '<input name="q" placeholder="Search">' +
    '</form><div id="results"></div>';

  document.getElementById("search").addEventListener("submit", function (event) {
    event.preventDefault();
    const value = document.querySelector('#search input[name="q"]').value;
    fetch("/pulse/api/search?q=" + encodeURIComponent(value)).then(function (response) {
      return response.json();
    }).then(function (data) {
      const results = document.getElementById("results");
      results.innerHTML = data.items.map(function (item) {
        return '<a href="/pulse/items/' + item + '">item ' + item + '</a>';
      }).join(" ");
    });
  });

  if (location.pathname.indexOf("/pulse/items/") === 0) {
    fetch("/pulse/api" + location.pathname.slice("/pulse".length));
  }
</script>
</body>
</html>`
