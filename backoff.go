package boomerang

import (
	"math"
	"math/rand"
	"net/http"
	"time"
)

// CheckRetry specifies a policy for handling retries. It is called
// following each request with the response and error values returned by
// the http.Client. If CheckRetry returns false, the Client stops retrying
// and returns the response to the caller. If CheckRetry returns an error,
// that error value is returned in lieu of the error from the request. The
// Client will close any response body when retrying

type CheckRetry func(resp *http.Response, err error) (bool, error)

type Backoff interface {
	NextInterval(retry int) time.Duration
}

type BackoffFunc func(retry int) time.Duration

func (b BackoffFunc) NextInterval(retry int) time.Duration {
	return b(retry)
}

func NewBackoffFunc(f BackoffFunc) Backoff {
	return f
}

type exponentialBackoff struct {
	factor     float64
	minTimeout time.Duration
	maxTimeout time.Duration
}

// NewExponentialBackoff returns an instance of ExponentialBackoff
func NewExponentialBackoff(minTimeout, maxTimeout time.Duration, exponentFactor float64) Backoff {
	return &exponentialBackoff{
		factor:     exponentFactor,
		minTimeout: minTimeout,
		maxTimeout: maxTimeout,
	}
}

// Next returns next time for retrying operation with exponential strategy
func (e *exponentialBackoff) NextInterval(retryCount int) time.Duration {
	if retryCount <= 0 {
		return 0 * time.Millisecond
	}

	efac := math.Pow(e.factor, float64(retryCount)) * float64(e.minTimeout)
	sleep := math.Min(efac, float64(e.maxTimeout))

	return time.Duration(sleep)
}

type jitterBackoff struct {
	factor     float64
	minTimeout time.Duration
	maxTimeout time.Duration
}

// NewExponentialBackoff returns an instance of ExponentialBackoff
func NewJitterBackoff(minTimeout, maxTimeout time.Duration, exponentFactor float64) Backoff {
	return &jitterBackoff{
		factor:     exponentFactor,
		minTimeout: minTimeout,
		maxTimeout: maxTimeout,
	}
}

// Next returns next time for retrying operation with exponential strategy
func (e *jitterBackoff) NextInterval(retryCount int) time.Duration {
	if retryCount <= 0 {
		return 0 * time.Millisecond
	}

	//calculate this duration
	minf := float64(e.minTimeout)
	durf := minf * math.Pow(e.factor, float64(retryCount))
	durf = rand.Float64()*(durf-minf) + minf
	dur := time.Duration(durf)
	//keep within bounds
	if dur < e.minTimeout {
		return e.minTimeout
	} else if dur > e.maxTimeout {
		return e.maxTimeout
	}

	return dur
}

type constantBackoff struct {
	timeout time.Duration
}

// NewConstanctBackoff returns an instance of ConstantBackoff
func NewConstantBackoff(timeout time.Duration) Backoff {
	return &constantBackoff{
		timeout: timeout,
	}
}

// Next returns next time for retrying operation with exponential strategy
func (cb *constantBackoff) NextInterval(retryCount int) time.Duration {
	if retryCount <= 0 {
		return 0 * time.Millisecond
	}

	return cb.timeout
}
