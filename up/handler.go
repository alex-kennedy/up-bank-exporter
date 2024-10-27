package up

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewMetricsHandler(upBearerToken string) (http.Handler, error) {
	client, err := NewUpMetricsClient(upBearerToken)
	if err != nil {
		return nil, err
	}
	return &metricsHandler{c: client, p: promhttp.Handler()}, nil
}

type metricsHandler struct {
	c *UpMetricsClient
	p http.Handler
}

func (h *metricsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if err := h.c.UpdateMetrics(req.Context()); err != nil {
		log.Printf("failed to update metrics: %v", err)
	}
	h.p.ServeHTTP(w, req)
}
