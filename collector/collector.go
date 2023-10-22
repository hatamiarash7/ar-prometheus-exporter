package collector

import (
	"sync"
	"time"

	"git.arvancloud.ir/project/cdn/ar-prometheus-exporter/config"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const (
	namespace = "arvancloud"

	// DefaultTimeout defines the default timeout when connecting to a router
	DefaultTimeout = 5 * time.Second
)

// ArvancloudCollector is the interface a collector has to implement.
type ArvancloudCollector interface {
	describe(ch chan<- *prometheus.Desc)
	collect(ctx *collectorContext) error
}

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"arvancloud: duration of a collector scrape",
		[]string{},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"arvancloud: whether a collector succeeded",
		[]string{},
		nil,
	)
)

type collector struct {
	collectors []ArvancloudCollector
	timeout    time.Duration
	token      string
}

// WithCDN enables CDN metrics
func WithCDN() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newCDNCollector())
	}
}

// WithObject enables ObjectStorage metrics
func WithObject() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newObjectCollector())
	}
}

// WithTimeout sets timeout for connecting to router
func WithTimeout(d time.Duration) Option {
	return func(c *collector) {
		c.timeout = d
	}
}

// Option applies options to collector
type Option func(*collector)

// NewCollector creates a collector instance
func NewCollector(cfg *config.Config, opts ...Option) (prometheus.Collector, error) {
	log.Info("Setting up collector for products")

	c := &collector{
		timeout: DefaultTimeout,
		token:   cfg.Token,
	}

	for _, o := range opts {
		o(c)
	}

	return c, nil
}

// Describe implements the prometheus.Collector interface.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc

	for _, co := range c.collectors {
		co.describe(ch)
	}
}

// Collect implements the prometheus.Collector interface.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}

	wg.Add(1)

	go func() {
		c.startCollect(ch)
		wg.Done()
	}()

	wg.Wait()
}

func (c *collector) startCollect(ch chan<- prometheus.Metric) {
	begin := time.Now()

	err := c.connectAndCollect(ch)

	duration := time.Since(begin)
	var success float64
	if err != nil {
		log.Errorf("ERROR: collector failed after %fs: %s", duration.Seconds(), err)
		success = 0
	} else {
		log.Debugf("OK: collector succeeded after %fs.", duration.Seconds())
		success = 1
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds())
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success)
}

func (c *collector) connectAndCollect(ch chan<- prometheus.Metric) error {
	for _, co := range c.collectors {
		ctx := &collectorContext{ch, c.token}
		err := co.collect(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
