// Package scraper implements the (reverse engineered) HTTP client for the
// Setecna REG web interface at s5a.eu. It handles login, CSRF tokens,
// session-cookie management and automatic session-expiry detection.
package scraper

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://s5a.eu"

// ErrSessionExpired is returned when the server no longer recognizes the
// session (e.g. the Laravel cookie expired) and a new Login is required.
var ErrSessionExpired = errors.New("setecna session expired")

var (
	metaTagRe = regexp.MustCompile(`(?is)<meta\s+[^>]*name\s*=\s*["']csrf-token["'][^>]*>`)
	contentRe = regexp.MustCompile(`(?is)content\s*=\s*["']([^"']+)["']`)
)

// extractCSRFToken finds the <meta name="csrf-token" content="..."> tag in
// the login page without requiring a full HTML parser.
func extractCSRFToken(body []byte) (string, error) {
	tag := metaTagRe.Find(body)
	if tag == nil {
		return "", errors.New("csrf-token meta tag not found in login page")
	}
	m := contentRe.FindSubmatch(tag)
	if m == nil {
		return "", errors.New("csrf-token meta tag has no content attribute")
	}
	return string(m[1]), nil
}

// FlexString unmarshals JSON values that may be either strings or numbers.
type FlexString string

func (fi *FlexString) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		return json.Unmarshal(b, (*string)(fi))
	}
	var i int
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}
	*fi = FlexString(strconv.Itoa(i))
	return nil
}

// Datum is a single parameter returned by the getres endpoint.
type Datum struct {
	ID string     `json:"Id"`
	V  FlexString `json:"V"`
}

// Response is the payload of the getres endpoint.
type Response struct {
	Status    string  `json:"Status"`
	Timestamp string  `json:"Timestamp"`
	Latest    string  `json:"Latest"`
	Rows      int     `json:"Rows"`
	Data      []Datum `json:"Data"`
}

// Map converts the response data to a key/value map.
func (r *Response) Map() map[string]string {
	m := make(map[string]string, len(r.Data))
	for _, d := range r.Data {
		m[d.ID] = string(d.V)
	}
	return m
}

// Scraper is the stateful HTTP client for a single Setecna system.
type Scraper struct {
	client          *http.Client
	loginURL        string
	fetchUpdatesURL string
	askRefreshURL   string
	pushUpdatesURL  string
	lastFetch       string
	username        string
	password        string
}

// New creates a scraper for the given system ID.
func New(systemID string) (*Scraper, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("creating cookie jar: %w", err)
	}
	return &Scraper{
		client: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		loginURL:        baseURL + "/login",
		fetchUpdatesURL: baseURL + "/station/" + systemID + "/getres?timestamp=",
		askRefreshURL:   baseURL + "/station/" + systemID + "/askrefresh?connrq=1",
		pushUpdatesURL:  baseURL + "/station/" + systemID + "/putmprop?statid=" + systemID + "&userid=guest&pcount=1",
	}, nil
}

// Login authenticates against the Setecna servers. Credentials are stored
// so the scraper can transparently re-authenticate when the session expires.
func (s *Scraper) Login(username, password string) error {
	s.username, s.password = username, password
	return s.relogin()
}

func (s *Scraper) relogin() error {
	// Fetch the login page to obtain a fresh CSRF token.
	resp, err := s.client.Get(s.loginURL)
	if err != nil {
		return fmt.Errorf("fetching login page: %w", err)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("reading login page: %w", err)
	}
	token, err := extractCSRFToken(body)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("_token", token)
	data.Set("email", s.username)
	data.Set("password", s.password)

	postResp, err := s.client.PostForm(s.loginURL, data)
	if err != nil {
		return fmt.Errorf("posting credentials: %w", err)
	}
	defer postResp.Body.Close()
	io.Copy(io.Discard, postResp.Body)

	// After following redirects, landing back on /login means failure.
	if postResp.Request != nil && strings.Contains(postResp.Request.URL.Path, "login") {
		return errors.New("login rejected: check username and password")
	}
	return nil
}

// AskRefresh asks the Setecna cloud to poll the controller for fresh data.
func (s *Scraper) AskRefresh() error {
	resp, err := s.client.Get(s.askRefreshURL)
	if err != nil {
		return fmt.Errorf("asking refresh: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

// Fetch retrieves the latest parameter values. It returns ErrSessionExpired
// when the server answered with the login page instead of JSON, in which
// case the caller should Relogin (or Fetch again after Login succeeded).
func (s *Scraper) Fetch() (Response, error) {
	requestURL := s.fetchUpdatesURL
	if s.lastFetch != "" {
		requestURL += url.QueryEscape(s.lastFetch)
	}

	resp, err := s.client.Get(requestURL)
	if err != nil {
		return Response{}, fmt.Errorf("fetching updates: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return Response{}, fmt.Errorf("reading updates: %w", err)
	}

	// Session expired: the server redirects to the HTML login page.
	trimmed := strings.TrimSpace(string(body))
	if resp.StatusCode == http.StatusUnauthorized ||
		(resp.Request != nil && strings.Contains(resp.Request.URL.Path, "login")) ||
		strings.HasPrefix(trimmed, "<") {
		return Response{}, ErrSessionExpired
	}
	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("unexpected status %d from getres", resp.StatusCode)
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return Response{}, fmt.Errorf("decoding getres JSON: %w", err)
	}

	s.lastFetch = result.Timestamp

	// Expose the server-side "Latest" timestamp as a virtual parameter.
	result.Data = append(result.Data, Datum{ID: "LAST_UPDATE", V: FlexString(result.Latest)})
	return result, nil
}

// FetchWithRelogin fetches and, on session expiry, transparently logs in
// again once before retrying.
func (s *Scraper) FetchWithRelogin() (Response, error) {
	r, err := s.Fetch()
	if !errors.Is(err, ErrSessionExpired) {
		return r, err
	}
	if err := s.relogin(); err != nil {
		return Response{}, fmt.Errorf("re-login after session expiry failed: %w", err)
	}
	return s.Fetch()
}

// Push writes a single parameter value to the Setecna system.
func (s *Scraper) Push(key, value string) error {
	fullURL := s.pushUpdatesURL + "&p0=" + url.QueryEscape(key) + "&nb0=" + url.QueryEscape(value)
	resp, err := s.client.Get(fullURL)
	if err != nil {
		return fmt.Errorf("pushing %s=%s: %w", key, value, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pushing %s=%s: unexpected status %d", key, value, resp.StatusCode)
	}
	return nil
}
