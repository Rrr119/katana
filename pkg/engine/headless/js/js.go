package js

import (
	_ "embed"
	"fmt"

	"github.com/pkg/errors"
	"github.com/projectdiscovery/rod"
)

var (
	//go:embed utils.js
	utilsJavascriptBundle string

	//go:embed page-init.js
	pageInitJavascriptBundle string
)

// InitJavascriptEnv injects the necessary javascript code into the browser
func InitJavascriptEnv(page *rod.Page) error {
	return InitJavascriptEnvWithHooks(page, false)
}

// InitJavascriptEnvWithHooks always installs SPA route sinks. When hooks is
// true it additionally instruments event listeners for the interactive crawler;
// instrumenting every listener slows ordinary crawls.
func InitJavascriptEnvWithHooks(page *rod.Page, hooks bool) error {
	if _, err := page.EvalOnNewDocument(utilsJavascriptBundle); err != nil {
		return errors.Wrap(err, "failed to inject utils.js")
	}
	// Navigation sinks are cheap and are needed by ordinary headless crawls to
	// discover SPA routes that never issue a document request. Listener and
	// form-reset instrumentation remains opt-in because it adds per-element
	// work on every page.
	options := fmt.Sprintf(`window.__katanaHooksOptions = { hooked: %t, navigationSinks: true };`, hooks)
	if _, err := page.EvalOnNewDocument(options); err != nil {
		return errors.Wrap(err, "failed to configure javascript hooks")
	}
	if _, err := page.EvalOnNewDocument(pageInitJavascriptBundle); err != nil {
		return errors.Wrap(err, "failed to inject page-init.js")
	}
	return nil
}
