package reader

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	HttpType          = "http"
	DefaultUserAgent  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:99.0) Gecko/20100101 Firefox/99.0"
	UserAgentFragment = "user-agent"
)

type AuthType interface {
	ConfigureAuth(r *http.Request) error
}

type BasicAuth struct {
	Username string
	Password string
}

func (a *BasicAuth) ConfigureAuth(r *http.Request) error {
	r.SetBasicAuth(a.Username, a.Password)
	return nil
}

type Doer interface {
	Do(r *http.Request) (*http.Response, error)
}

type HTTP struct {
	Client Doer
}

func ParseHTTPItemFromURL(str string) (OpenReadCloser, error) {
	u, err := url.Parse(str)
	if err != nil {
		return nil, err
	}

	// using fragments to encode downloader parameters.  could be problematic
	// if a site requires specific fragments.
	fragment, _ := url.ParseQuery(u.Fragment)

	var out HTTPSourceConfiguration

	out.UserAgent = fragment.Get(UserAgentFragment)
	if u.User != nil {
		pass, _ := u.User.Password()
		out.Auth = &BasicAuth{
			Username: u.User.Username(),
			Password: pass,
		}
		u.User = nil
	}

	out.Method = http.MethodGet
	out.URL = u
	return &out, nil
}

type HTTPSourceConfiguration struct {
	URL       *url.URL
	Method    string
	Body      io.Reader
	Auth      AuthType
	UserAgent string
}

func (i *HTTPSourceConfiguration) Type() string {
	return HttpType
}

func (i *HTTPSourceConfiguration) String() string {
	// this is mainly for display purposes, so it doesn't need to include
	// *all* information.
	//
	// could use url.URL.string, but dont want to include auth information

	sb := strings.Builder{}
	sb.WriteString(i.URL.Scheme)
	sb.WriteString("://")
	sb.WriteString(i.URL.Host)
	if p := i.URL.Port(); p != "" {
		sb.WriteString(":")
		sb.WriteString(p)
	}
	sb.WriteString(i.URL.Path)
	if len(i.URL.RawQuery) > 0 {
		sb.WriteString("?")
		sb.WriteString(i.URL.RawQuery)
	}
	return sb.String()
}

func (i *HTTPSourceConfiguration) OpenReadCloser() (io.ReadCloser, int64, error) {
	method := defaultValue(http.MethodGet, i.Method)
	req, err := http.NewRequest(method, i.URL.String(), i.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("http: error building request: %w", err)
	}
	req.Header.Set("User-Agent", defaultValue(DefaultUserAgent, i.UserAgent))

	if i.Auth != nil {
		if err := i.Auth.ConfigureAuth(req); err != nil {
			return nil, 0, fmt.Errorf("http: error configuring auth: %w", err)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("http: error making request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("http: got a %v response from the server", resp.StatusCode)
	}

	return resp.Body, resp.ContentLength, nil
}

func defaultValue[T comparable](def, val T) T {
	var empty T
	if empty == val {
		return def
	}
	return val
}
