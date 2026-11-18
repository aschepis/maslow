package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"

	"github.com/aschepis/maslow/internal/spec"
)

const defaultBrowserTimeout = 30 * time.Second

// browserCandidates are tried in order when MASLOW_BROWSER is not set.
var browserCandidates = []string{
	"google-chrome",
	"google-chrome-stable",
	"chromium",
	"chromium-browser",
	"chrome",
	"headless-shell",
	"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	"/Applications/Chromium.app/Contents/MacOS/Chromium",
	`C:\Program Files\Google\Chrome\Application\chrome.exe`,
}

// findBrowser returns the Chrome or Chromium binary that browser steps drive.
func findBrowser() (string, error) {
	if p := os.Getenv("MASLOW_BROWSER"); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("MASLOW_BROWSER=%s: %w", p, err)
		}
		return p, nil
	}
	for _, name := range browserCandidates {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", errors.New("no Chrome or Chromium found; install one or set MASLOW_BROWSER to its path")
}

// browser is one headless browser with one tab, owned by a scope.
type browser struct {
	tab    context.Context
	cancel func()

	mu      sync.Mutex
	console []string
}

func startBrowser() (*browser, error) {
	path, err := findBrowser()
	if err != nil {
		return nil, err
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(path), chromedp.WindowSize(1280, 800))
	if os.Geteuid() == 0 {
		// Chrome refuses to start as root, as in many CI containers, unless sandboxing is off.
		opts = append(opts, chromedp.NoSandbox)
	}
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	tab, tabCancel := chromedp.NewContext(allocCtx, chromedp.WithErrorf(func(string, ...any) {}))
	b := &browser{tab: tab, cancel: func() {
		_ = chromedp.Cancel(tab)
		tabCancel()
		allocCancel()
	}}

	// No timeout here: the first call's context would bound the browser's whole life.
	if err := chromedp.Do(tab); err != nil {
		b.cancel()
		return nil, fmt.Errorf("starting %s: %w", path, err)
	}
	messages := chromedp.Console(tab)
	go func() {
		for m, err := range messages {
			if err != nil {
				return
			}
			if m.Type == chromedp.ConsoleError || m.IsException() {
				b.mu.Lock()
				b.console = append(b.console, m.String())
				b.mu.Unlock()
			}
		}
	}()
	return b, nil
}

func (b *browser) close() {
	if b != nil {
		b.cancel()
	}
}

func (b *browser) consoleErrors() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.console...)
}

// pageURL is best effort: it is evidence about a failure, not part of the check.
func (b *browser) pageURL() string {
	ctx, cancel := context.WithTimeout(b.tab, 5*time.Second)
	defer cancel()
	loc, _ := chromedp.Run(ctx, chromedp.Location())
	return loc
}

func (b *browser) screenshot(path string) error {
	ctx, cancel := context.WithTimeout(b.tab, 10*time.Second)
	defer cancel()
	buf, err := chromedp.Run(ctx, chromedp.CaptureScreenshot())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf, 0o644)
}

// elementState is what one inspection of the page saw.
type elementState struct {
	Found   bool   `json:"found"`
	Visible bool   `json:"visible"`
	Enabled bool   `json:"enabled"`
	Text    string `json:"text"`
}

// inspect reads an element, or the whole page for "", in one round trip so
// the result describes a single moment.
func inspect(ctx context.Context, selector string) (elementState, error) {
	sel, _ := json.Marshal(selector)
	expr := `(sel => {
  const el = sel ? document.querySelector(sel) : document.body;
  if (!el) return {found: false};
  const visible = !!(el.offsetWidth || el.offsetHeight || el.getClientRects().length);
  return {found: true, visible: visible, enabled: !el.disabled, text: el.innerText || ""};
})(` + string(sel) + `)`
	return chromedp.Run(ctx, chromedp.Evaluate[elementState](expr))
}

// requireActionable fails at once, rather than waiting, when an element is
// missing or hidden; a contract waits only through an explicit wait step.
func requireActionable(ctx context.Context, selector string) error {
	el, err := inspect(ctx, selector)
	switch {
	case err != nil:
		return err
	case !el.Found:
		return fmt.Errorf("selector %q not found (add a wait step if it appears later)", selector)
	case !el.Visible:
		return fmt.Errorf("selector %q is not visible", selector)
	}
	return nil
}

// checkBrowserStep rejects a step the schema would reject, before a browser starts.
func checkBrowserStep(step spec.Step) error {
	switch step.Do {
	case "navigate":
		if step.URL == "" {
			return errors.New("navigate needs url")
		}
	case "click", "fill", "wait":
		if step.Selector == "" {
			return fmt.Errorf("%s needs selector", step.Do)
		}
	case "assert":
		e := step.Expect
		if e == nil || (e.TextContains == "" && e.URLContains == "" && e.Visible == nil && e.Enabled == nil) {
			return errors.New("assert needs expect with text_contains, url_contains, visible or enabled")
		}
		if step.Selector == "" && (e.Visible != nil || e.Enabled != nil) {
			return errors.New("assert on visible or enabled needs selector")
		}
	default:
		return fmt.Errorf("unknown browser operation %q (want navigate, click, fill, wait or assert)", step.Do)
	}
	return nil
}

// runBrowserStep runs one operation, carrying the scenario's cookies into the
// browser before it and back out after, so HTTP and browser steps share a session.
func runBrowserStep(step spec.Step, state *stepState, sc *scope) error {
	if err := checkBrowserStep(step); err != nil {
		return err
	}
	if sc.browser == nil {
		b, err := startBrowser()
		if err != nil {
			return err
		}
		sc.browser = b
	}

	timeout := defaultBrowserTimeout
	if step.Timeout != "" {
		d, err := parseDuration(step.Timeout)
		if err != nil {
			return fmt.Errorf("timeout %q: %w", step.Timeout, err)
		}
		timeout = d
	}
	ctx, cancel := context.WithTimeout(sc.browser.tab, timeout)
	defer cancel()

	target := step.URL
	if step.Do != "navigate" {
		target = sc.browser.pageURL()
	}
	if err := pushCookies(ctx, sc.client.Jar, target); err != nil {
		return fmt.Errorf("copying cookies into the browser: %w", err)
	}

	state.output = ""
	if err := browserOperation(ctx, step, state); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("timed out after %s", timeout)
		}
		return err
	}
	if err := pullCookies(ctx, sc.client.Jar); err != nil {
		return fmt.Errorf("copying cookies from the browser: %w", err)
	}
	return nil
}

func browserOperation(ctx context.Context, step spec.Step, state *stepState) error {
	switch step.Do {
	case "navigate":
		return chromedp.Do(ctx, chromedp.Navigate(step.URL))

	case "click":
		if err := requireActionable(ctx, step.Selector); err != nil {
			return err
		}
		return chromedp.Do(ctx, chromedp.Click(chromedp.CSS(step.Selector)))

	case "fill":
		if err := requireActionable(ctx, step.Selector); err != nil {
			return err
		}
		// chromedp.SetValue rejects "", so the field is cleared directly before typing.
		sel, _ := json.Marshal(step.Selector)
		clear := `document.querySelector(` + string(sel) + `).value = ""`
		if _, err := chromedp.Run(ctx, chromedp.Evaluate[string](clear)); err != nil {
			return err
		}
		return chromedp.Do(ctx, chromedp.SendKeys(chromedp.CSS(step.Selector), step.Text))

	case "wait":
		return chromedp.Do(ctx, chromedp.WaitVisible(chromedp.CSS(step.Selector)))

	case "assert":
		return assertPage(ctx, step, state)
	}
	return nil
}

func assertPage(ctx context.Context, step spec.Step, state *stepState) error {
	e := step.Expect
	if e.URLContains != "" {
		loc, err := chromedp.Run(ctx, chromedp.Location())
		if err != nil {
			return err
		}
		if !strings.Contains(loc, e.URLContains) {
			return fmt.Errorf("expected url to contain %q, got %q", e.URLContains, loc)
		}
	}
	if e.TextContains == "" && e.Visible == nil && e.Enabled == nil {
		return nil
	}

	el, err := inspect(ctx, step.Selector)
	if err != nil {
		return err
	}
	if !el.Found {
		return fmt.Errorf("selector %q not found", step.Selector)
	}
	state.output = el.Text
	if e.TextContains != "" && !strings.Contains(el.Text, e.TextContains) {
		return fmt.Errorf("expected text to contain %q", e.TextContains)
	}
	if e.Visible != nil && el.Visible != *e.Visible {
		return fmt.Errorf("expected %q visible=%t, got %t", step.Selector, *e.Visible, el.Visible)
	}
	if e.Enabled != nil && el.Enabled != *e.Enabled {
		return fmt.Errorf("expected %q enabled=%t, got %t", step.Selector, *e.Enabled, el.Enabled)
	}
	return nil
}

// pushCookies gives the browser the jar's cookies for pageURL. The jar exposes
// only names and values, so they are set for the whole host.
func pushCookies(ctx context.Context, jar http.CookieJar, pageURL string) error {
	u, err := url.Parse(pageURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil
	}
	var params []*network.CookieParam
	for _, c := range jar.Cookies(u) {
		params = append(params, &network.CookieParam{Name: c.Name, Value: c.Value, URL: pageURL, Path: "/"})
	}
	if len(params) == 0 {
		return nil
	}
	_, err = chromedp.Call(ctx, network.SetCookies, network.SetCookiesParams{Cookies: params})
	return err
}

// pullCookies copies every browser cookie into the jar, keeping its scope.
func pullCookies(ctx context.Context, jar http.CookieJar) error {
	res, err := chromedp.CallBrowser(ctx, storage.GetCookies, storage.GetCookiesParams{})
	if err != nil {
		return err
	}
	for _, c := range res.Cookies {
		scheme := "http"
		if c.Secure {
			scheme = "https"
		}
		u := &url.URL{Scheme: scheme, Host: strings.TrimPrefix(c.Domain, "."), Path: c.Path}
		hc := &http.Cookie{Name: c.Name, Value: c.Value, Path: c.Path, Secure: c.Secure, HttpOnly: c.HTTPOnly}
		// A leading dot marks a domain cookie; without one it is host-only.
		if strings.HasPrefix(c.Domain, ".") {
			hc.Domain = c.Domain
		}
		if !c.Session {
			hc.Expires = time.UnixMilli(int64(c.Expires * 1000))
		}
		jar.SetCookies(u, []*http.Cookie{hc})
	}
	return nil
}

var unsafePathChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// screenshotPath is where a scenario's failing browser step leaves its
// screenshot: fixed per scenario, so reruns replace rather than accumulate.
func screenshotPath(contract, scenario string) string {
	slug := func(s string) string { return strings.Trim(unsafePathChars.ReplaceAllString(s, "-"), "-") }
	return "reports/browser/" + slug(contract) + "/" + slug(scenario) + ".png"
}
