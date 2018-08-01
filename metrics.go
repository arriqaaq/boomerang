package boomerang

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

const (
	DEFAULT_PROM_METRICS_NAMESPACE = "boomerang"
)

type Metrics interface {
	Record(error, int, time.Duration)
}

func NewPrometheusMetrics(namespace, subsystem string) Metrics {
	fieldKeys := []string{"error"}

	trc := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)

	rl := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request_latency",
		Help:      "Total duration of requests in seconds.",
	}, fieldKeys)

	scc := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "status_code",
		Help:      "Count of different response status codes.",
	}, []string{"status_code"})

	prometheus.MustRegister(trc)
	prometheus.MustRegister(rl)
	prometheus.MustRegister(scc)

	return &promMetrics{
		totalRequestCount: trc,
		requestLatency:    rl,
		statusCodeCounter: scc,
	}

}

type promMetrics struct {
	totalRequestCount *prometheus.CounterVec
	requestLatency    *prometheus.SummaryVec
	statusCodeCounter *prometheus.CounterVec
}

func (p *promMetrics) Record(err error, statusCode int, duration time.Duration) {
	sc := fmt.Sprintf("%dxx", statusCode/100)
	errLabel := prometheus.Labels{"error": fmt.Sprint(err)}
	p.totalRequestCount.With(errLabel).Add(1)
	p.requestLatency.With(errLabel).Observe(duration.Seconds())
	p.statusCodeCounter.With(prometheus.Labels{"status_code": sc}).Add(1)
}
