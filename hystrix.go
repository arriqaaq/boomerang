package boomerang

import (
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultMinTimeout        = 10 * time.Millisecond
	defaultMaxTimeout        = 20 * time.Millisecond
	defaultFactor            = 2
	DefaultMaxHystrixRetries = 1
)

type fallbackFunc func(error) error

type HystrixCommandConfig struct {
	Timeout                int `json:"timeout"`
	MaxConcurrentRequests  int `json:"max_concurrent_requests"`
	RequestVolumeThreshold int `json:"request_volume_threshold"`
	SleepWindow            int `json:"sleep_window"`
	ErrorPercentThreshold  int `json:"error_percent_threshold"`
	CommandName            string
	Transport              *http.Transport
}

func NewHystrixClient(timeout time.Duration, hc HystrixCommandConfig) *HystrixClient {
	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: hc.Transport,
	}
	hysCmdConfig := hystrix.CommandConfig{
		Timeout:                hc.Timeout,
		MaxConcurrentRequests:  hc.MaxConcurrentRequests,
		RequestVolumeThreshold: hc.RequestVolumeThreshold,
		SleepWindow:            hc.SleepWindow,
		ErrorPercentThreshold:  hc.ErrorPercentThreshold,
	}

	hystrix.ConfigureCommand(hc.CommandName, hysCmdConfig)

	return &HystrixClient{
		client:      httpClient,
		MaxRetries:  DefaultMaxHystrixRetries,
		commandName: hc.CommandName,
		Backoff: NewConstantBackoff(
			defaultMinTimeout,
		),
	}
}

type HystrixClient struct {
	commandName string
	client      *http.Client
	Logger      *log.Logger // Customer logger instance.

	Backoff Backoff

	// CheckRetry specifies the policy for handling retries, and is called
	// after each request. The default policy is DefaultRetryPolicy.
	CheckRetry CheckRetry
	MaxRetries int

	fallbackFunc func(err error) error
}

func (c *HystrixClient) SetFallbackFunc(fbf func(err error) error) {
	c.fallbackFunc = fbf
}

func (c *HystrixClient) Head(url string) (*http.Response, error) {
	req, err := NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)

}

func (c *HystrixClient) Get(url string) (*http.Response, error) {
	req, err := NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *HystrixClient) Post(url string, contentType string, body io.ReadSeeker) (*http.Response, error) {
	req, err := NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)

}

func (c *HystrixClient) PostForm(url string, data url.Values) (*http.Response, error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))

}

func (c *HystrixClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i < c.MaxRetries; i++ {

		err = hystrix.Do(c.commandName, func() error {
			resp, err = c.client.Do(req)
			if err != nil {
				c.Logger.Printf("[ERR] %s %s request failed: %v", req.Method, req.URL, err)
			}

			// Check if we should continue with retries.
			checkOK, checkErr := c.CheckRetry(resp, err)

			if !checkOK {
				if checkErr != nil {
					err = checkErr
				}
				return err
			}

			// if response.StatusCode >= http.StatusInternalServerError {
			// 	return fmt.Errorf("Server is down: returned status code: %d", response.StatusCode)
			// }
			return nil
		}, c.fallbackFunc)

		if err != nil {
			waitTime := c.Backoff.NextInterval(i)
			desc := fmt.Sprintf("%s %s", req.Method, req.URL)
			// desc = fmt.Sprintf("%s (status: %d)", desc, code)
			c.Logger.Printf("[DEBUG] %s: retrying in %s (%d left)", desc, waitTime, i)
			time.Sleep(waitTime)
			continue
		}

		// We're going to retry, consume any response to reuse the connection.
		if err == nil {
			c.drainBody(resp.Body)
			return resp, err
		}

		break
	}

	// Return an error if we fall out of the retry loop
	return nil, fmt.Errorf("%s %s giving up after %d attempts",
		req.Method, req.URL, c.MaxRetries+1)

}

// Try to read the response body so we can reuse this connection.
func (c *HystrixClient) drainBody(body io.ReadCloser) {
	defer body.Close()
	_, err := io.Copy(ioutil.Discard, io.LimitReader(body, respReadLimit))
	if err != nil {
		c.Logger.Printf("[ERR] error reading response body: %v", err)
	}
}
