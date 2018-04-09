package boomerang

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestExponentialBackoffNextInterval(t *testing.T) {

	eb := NewExponentialBackoff(2*time.Millisecond, 10*time.Millisecond, 2.0)

	assert.Equal(t, 4*time.Millisecond, eb.NextInterval(1))
}

func TestExponentialBackoffMaxOverflow(t *testing.T) {

	eb := NewExponentialBackoff(2*time.Millisecond, 10*time.Millisecond, 2.0)

	assert.Equal(t, 10*time.Millisecond, eb.NextInterval(4))
}

func TestConstantBackoffNextInterval(t *testing.T) {

	cb := NewConstantBackoff(10 * time.Millisecond)

	assert.Equal(t, 10*time.Millisecond, cb.NextInterval(1))
	assert.Equal(t, 10*time.Millisecond, cb.NextInterval(10))
}

func TestJitterBackoffMaxOverflow(t *testing.T) {

	jb := NewJitterBackoff(2*time.Millisecond, 10*time.Millisecond, 2.0)
	assert.Equal(t, 10*time.Millisecond, jb.NextInterval(10))
}

func TestJitterBackoffNextInterval(t *testing.T) {
	var dur time.Duration

	low := 10 * time.Millisecond

	jb := NewJitterBackoff(low, 3*time.Second, 2.0)
	dur = jb.NextInterval(1)
	assert.Equal(t, between(t, dur, low, 20*time.Millisecond), true)
	dur = jb.NextInterval(2)
	assert.Equal(t, between(t, dur, low, 50*time.Millisecond), true)
}

func between(t *testing.T, val, low, high time.Duration) bool {
	if val < low {
		return false
	}
	if val > high {
		return false
	}
	if val >= low && val <= high {
		return true
	}
	return false
}
