package boomerang

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// LenReader is an interface implemented by many in-memory io.Reader's. Used
// for automatically sending the right Content-Length header when possible.
type LenReader interface {
	Len() int
}

// Request wraps the metadata needed to create HTTP requests.
type Request struct {
	// body is a seekable reader over the request body payload. This is
	// used to rewind the request data in between retries.
	body io.ReadSeeker

	// Embed an HTTP request directly. This makes a *Request act exactly
	// like an *http.Request so that all meta methods are supported.
	*http.Request
}

func NewRequest(method, url string, body io.ReadSeeker) (*Request, error) {
	// Wrap the body in a noop ReadCloser if non-nil. This prevents the
	// reader from being closed by the HTTP client.
	var rcBody io.ReadCloser
	if body != nil {
		rcBody = ioutil.NopCloser(body)
	}

	// Make the request with the noop-closer for the body.
	httpReq, err := http.NewRequest(method, url, rcBody)
	if err != nil {
		return nil, err
	}

	// Check if we can set the Content-Length automatically.
	if lr, ok := body.(LenReader); ok {
		httpReq.ContentLength = int64(lr.Len())
	}

	return &Request{body, httpReq}, nil
}

// DefaultRetryPolicy provides a default callback for Client.CheckRetry, which
// will retry on connection errors and server errors.
func DefaultRetryPolicy(resp *http.Response, err error) (bool, error) {
	if err != nil {
		return true, err
	}
	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like 0 and 999.
	if resp.StatusCode == 0 || resp.StatusCode >= 500 {
		return true, nil
	}

	return false, nil
}

func DefaultHttpClient() Client {

	nc := new(HttpClient)
	nc.client = &http.Client{
		Timeout:   DefaultTimeout,
		Transport: DefaultTransport(),
	}
	nc.Logger = log.New(os.Stderr, "", log.LstdFlags)
	nc.Backoff = NewConstantBackoff(
		defaultMinTimeout,
	)
	nc.CheckRetry = DefaultRetryPolicy
	nc.MaxRetries = DefaultMaxHttpRetries
	return nc
}

func NewHttpClient(timeout time.Duration, transport *http.Transport) *HttpClient {

	nc := new(HttpClient)
	nc.client = &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	nc.Logger = log.New(os.Stderr, "", log.LstdFlags)
	nc.Backoff = NewConstantBackoff(
		defaultMinTimeout,
	)
	nc.CheckRetry = DefaultRetryPolicy
	nc.MaxRetries = DefaultMaxHttpRetries
	return nc
}

type HttpClient struct {
	// client     *http.Client
	client *http.Client
	Logger *log.Logger // Customer logger instance.

	Backoff Backoff
	// CheckRetry specifies the policy for handling retries, and is called
	// after each request. The default policy is DefaultRetryPolicy.
	CheckRetry CheckRetry
	MaxRetries int
}

func (c *HttpClient) SetRetries(retry int) {
	c.MaxRetries = retry
}

func (c *HttpClient) SetBackoff(bc Backoff) {
	c.Backoff = bc
}

func (c *HttpClient) Head(url string) (*http.Response, error) {
	req, err := NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)

}

func (c *HttpClient) Get(url string) (*http.Response, error) {
	req, err := NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *HttpClient) Post(url string, contentType string, body io.ReadSeeker) (*http.Response, error) {
	req, err := NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)

}

func (c *HttpClient) PostForm(url string, data url.Values) (*http.Response, error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))

}

func (c *HttpClient) Do(req *Request) (*http.Response, error) {
	req.Close = true

	for i := 0; i < c.MaxRetries; i++ {

		// Always rewind the request body when non-nil.
		if req.body != nil {
			if _, err := req.body.Seek(0, 0); err != nil {
				return nil, fmt.Errorf("failed to seek body: %v", err)
			}
		}
		// Attempt the request
		resp, err := c.client.Do(req.Request)

		// Check if we should continue with retries.
		checkOK, checkErr := c.CheckRetry(resp, err)

		if err != nil {
			c.Logger.Printf("[ERR] %s %s request failed: %v", req.Method, req.URL, err)
		}

		if !checkOK {
			if checkErr != nil {
				err = checkErr
			}
			return resp, err
		}

		// We're going to retry, consume any response to reuse the connection.
		if err == nil {
			c.drainBody(resp.Body)
		}

		waitTime := c.Backoff.NextInterval(i)

		desc := fmt.Sprintf("%s %s", req.Method, req.URL)
		// desc = fmt.Sprintf("%s (status: %d)", desc, code)
		c.Logger.Printf("[DEBUG] %s: retrying in %s (%d left)", desc, waitTime, i)
		time.Sleep(waitTime)

	}

	// Return an error if we fall out of the retry loop
	return nil, fmt.Errorf("%s %s giving up after %d attempts",
		req.Method, req.URL, c.MaxRetries)

}

// Try to read the response body so we can reuse this connection.
func (c *HttpClient) drainBody(body io.ReadCloser) {
	defer body.Close()
	_, err := io.Copy(ioutil.Discard, io.LimitReader(body, respReadLimit))
	if err != nil {
		c.Logger.Printf("[ERR] error reading response body: %v", err)
	}
}
