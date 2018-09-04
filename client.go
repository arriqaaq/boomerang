package boomerang

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultMaxHttpRetries = 1
	DefaultTimeout        = 100 * time.Millisecond
)

var (
	// We need to consume response bodies to maintain http connections, but
	// limit the size we consume to respReadLimit.
	respReadLimit = int64(4096)
)

// Client Is a generic HTTP client interface
type Client interface {
	Get(url string) (*http.Response, error)
	Head(url string) (*http.Response, error)
	Post(url string, contentType string, body io.ReadSeeker) (*http.Response, error)
	PostForm(url string, data url.Values) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
}
