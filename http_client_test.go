package boomerang

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var (
	defaultClientConfig = &ClientConfig{
		Timeout:   10 * time.Millisecond,
		Transport: DefaultTransport(),
	}
)

func TestHttpClient_Get(t *testing.T) {
	client := NewHttpClient(defaultClientConfig)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	resp, err := client.Get(testServer.URL)
	require.NoError(t, err, "Http client Get call failed")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestHttpClient_Post(t *testing.T) {
	ctType := "application/json"
	client := NewHttpClient(defaultClientConfig)
	backoffStratergy := NewExponentialBackoff(2*time.Millisecond, 10*time.Millisecond, 2.0)
	client.SetBackoff(backoffStratergy)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, r.Header.Get("Content-Type"), ctType)

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err, "error reading body of post request")
		expected := `{"foo":"bar"}`
		assert.Equal(t, string(body), expected)

		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	resp, err := client.Post(testServer.URL, ctType, strings.NewReader(`{"foo":"bar"}`))
	require.NoError(t, err, "Http client Post call failed")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestHttpClient_Do(t *testing.T) {
	ctType := "application/json"
	client := NewHttpClient(defaultClientConfig)
	client.MaxRetries = 5

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, r.Header.Get("Content-Type"), ctType)

		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err, "error reading body of post request")
		expected := `{"foo":"bar"}`
		assert.Equal(t, string(body), expected)

		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	req, rErr := NewRequest("PUT", testServer.URL, strings.NewReader(`{"foo":"bar"}`))
	req.Header.Set("Content-Type", ctType)
	require.NoError(t, rErr, "error making a new request")

	resp, err := client.Do(req)
	require.NoError(t, err, "Http client PUT call failed")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestHttpClient_Do_Fail(t *testing.T) {
	ctType := "application/json"
	client := NewHttpClient(defaultClientConfig)
	client.MaxRetries = 5

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, r.Header.Get("Content-Type"), ctType)

		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err, "error reading body of post request")
		expected := `{"foo":"bar"}`
		assert.Equal(t, string(body), expected)

		w.WriteHeader(http.StatusBadGateway)
	}))
	defer testServer.Close()

	req, rErr := NewRequest("PUT", testServer.URL, strings.NewReader(`{"foo":"bar"}`))
	req.Header.Set("Content-Type", ctType)
	require.NoError(t, rErr, "error making a new request")

	_, err := client.Do(req)
	require.Error(t, err, "Http client PUT call failed")
	assert.Contains(t, err.Error(), "giving up")
}
