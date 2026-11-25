# SPA coverage benchmark

The benchmark answers a narrow question: after a page has rendered its own DOM,
how much of a manually verified route set does katana discover? Coverage is
reported as `covered / expected`; a route is covered when any crawled URL has
the same path. Query strings and fragments are ignored.

## Fixtures

All fixtures are served from `internal/spacoverage` on a random loopback port.
They deliberately avoid third-party assets and timers so runs are repeatable.

| Site | Rendering pattern | Manually verified routes |
| --- | --- | --- |
| `atlas-hash` | Links and content are created by JavaScript; `hashchange` loads each view through `fetch` | `/atlas/`, `/atlas/api/overview`, `/atlas/api/reports`, `/atlas/api/settings` |
| `orbit-history` | JavaScript creates buttons; clicks use `history.pushState` and load data through `fetch` | `/orbit/`, `/orbit/projects/1`, `/orbit/projects/2`, `/orbit/team`, `/orbit/api/projects/1`, `/orbit/api/projects/2`, `/orbit/api/team` |
| `pulse-form` | A JavaScript-rendered search form has no submit button; submitting loads result links, each opening a detail view | `/pulse/`, `/pulse/api/search`, `/pulse/items/1`, `/pulse/items/2`, `/pulse/api/items/1`, `/pulse/api/items/2` |

The manual lists are the routes a person reaches by opening the fixture in a
browser and following every visible navigation and the search form. They are
stored next to the fixture handlers so a route change is reviewed together with
the expected coverage.

## Running

Build katana, then run the comparison:

```console
go build -o katana ./cmd/katana
go run ./cmd/tools/spa-coverage -katana ./katana -modes standard,headless,hybrid
```

Useful flags:

- `-modes` selects any comma-separated subset of `standard`, `headless`, `hybrid`.
- `-min-coverage` fails the command when the headless overall coverage is below the threshold (default `0.75`).
- `-json` writes per-site and aggregate reports.
- `-crawl-seconds` bounds each crawl (default `90`).

Measured on the fixtures with a 20 second budget per site:

```text
mode       site           covered  expected  coverage  missing
headless   atlas-hash     4        4         100%
headless   orbit-history  7        7         100%
headless   pulse-form     6        6         100%
headless   all            17       17        100%
standard   all            3        17        18%       ...
hybrid     all            3        17        18%       ...
```

Headless is slower than standard, which is expected: it must load Chrome,
execute the page scripts, and replay route states. Hybrid renders the page for
output and parses the rendered DOM for links/forms, but it does not click
arbitrary JavaScript controls; use headless when route discovery depends on
button clicks or `history.pushState`.

## Browser test

`go test ./internal/spacoverage` runs the same comparison through the headless
crawler. It skips when Chrome/Chromium is not installed. Set
`KATANA_SPA_E2E=1` to let rod download a browser and run the e2e path in CI.
