package up

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	outgoingInflights = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "up_bank_http_outgoing_inflights",
		Help: "Number of Up bank outgoing HTTP requests inflight",
	})
	requestTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "up_bank_http_request_total",
		Help: "Total HTTP requests to the Up API",
	}, []string{"path", "code"})
	requestLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "up_bank_http_request_latency",
		Help:    "Latency histogram of Up API requests (ms)",
		Buckets: prometheus.ExponentialBucketsRange(1.0, 2000.0, 25),
	}, []string{"path", "code"})
	responseSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "up_bank_http_response_size",
		Help:    "Up bank API response size bytes",
		Buckets: prometheus.ExponentialBucketsRange(1.0, math.Pow(2, 20), 25),
	}, []string{"path", "code"})
)

type PromHTTPClient struct {
	c *http.Client
}

func NewPromHTTPClient() *PromHTTPClient {
	return &PromHTTPClient{c: http.DefaultClient}
}

func (p *PromHTTPClient) Do(req *http.Request) (*http.Response, error) {
	outgoingInflights.Inc()
	defer outgoingInflights.Dec()

	start := time.Now()
	resp, err := p.c.Do(req)
	if err != nil {
		log.Println(err)
		return resp, err
	}
	latency := time.Since(start)

	labels := prometheus.Labels{
		"path": req.URL.Path,
		"code": strconv.FormatInt(int64(resp.StatusCode), 10),
	}
	requestTotal.With(labels).Inc()
	requestLatency.With(labels).Observe(latency.Seconds() * 1000)
	responseSize.With(labels).Observe(float64(resp.ContentLength))

	return resp, err
}
