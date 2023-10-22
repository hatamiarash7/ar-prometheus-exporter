package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

type collectorContext struct {
	ch    chan<- prometheus.Metric
	token string
}
