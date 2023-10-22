package main

import (
	"bytes"
	"flag"
	"os"

	"github.com/prometheus/common/version"

	"net/http"

	"git.arvancloud.ir/project/cdn/ar-prometheus-exporter/collector"
	"git.arvancloud.ir/project/cdn/ar-prometheus-exporter/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	configFile  = flag.String("config-file", "", "config file to load")
	logFormat   = flag.String("log-format", "json", "log format text or json (default json)")
	logLevel    = flag.String("log-level", "info", "log level")
	metricsPath = flag.String("path", "/metrics", "path to answer requests on")
	token       = flag.String("token", "", "authentication token for API")
	port        = flag.String("port", ":9436", "port number to listen on")
	timeout     = flag.Duration("timeout", collector.DefaultTimeout, "timeout when connecting to devices")
	ver         = flag.Bool("version", false, "find the version of binary")

	withCDN    = flag.Bool("with-cdn", false, "retrieves CDN metrics")
	withObject = flag.Bool("with-object", false, "retrieves ObjectStorage metrics")

	cfg *config.Config
)

func init() {
	prometheus.MustRegister(version.NewCollector("arvancloud_exporter"))
}

func main() {
	flag.Parse()

	configureLog()

	log.Info("Welcome to Arvancloud Prometheus Exporter")

	log.Info("Version: 1.0.0")

	c, err := loadConfig()
	if err != nil {
		log.Errorf("Could not load config: %v", err)
		os.Exit(3)
	}
	cfg = c

	startServer()
}

func configureLog() {
	ll, err := log.ParseLevel(*logLevel)
	if err != nil {
		panic(err)
	}

	log.SetLevel(ll)

	if *logFormat == "text" {
		log.SetFormatter(&log.TextFormatter{})
	} else {
		log.SetFormatter(&log.JSONFormatter{})
	}
}

func loadConfig() (*config.Config, error) {
	if *configFile != "" {
		return loadConfigFromFile()
	}

	return loadConfigFromFlags()
}

func loadConfigFromFile() (*config.Config, error) {
	b, err := os.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}

	return config.Load(bytes.NewReader(b))
}

func loadConfigFromFlags() (*config.Config, error) {
	return &config.Config{
		Token: *token,
	}, nil
}

func startServer() {
	h, err := createMetricsHandler()
	if err != nil {
		log.Fatal(err)
	}
	http.Handle(*metricsPath, h)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
			<head><title>Arvancloud Exporter</title></head>
			<body>
			<h1>Arvancloud Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Info("Listening on ", *port)
	log.Fatal(http.ListenAndServe(*port, nil))
}

func createMetricsHandler() (http.Handler, error) {
	opts := collectorOptions()
	nc, err := collector.NewCollector(cfg, opts...)
	if err != nil {
		return nil, err
	}

	registry := prometheus.NewRegistry()
	err = registry.Register(nc)
	if err != nil {
		return nil, err
	}

	return promhttp.HandlerFor(registry,
		promhttp.HandlerOpts{
			ErrorLog:      log.New(),
			ErrorHandling: promhttp.ContinueOnError,
		}), nil
}

func collectorOptions() []collector.Option {
	opts := []collector.Option{}

	if *withCDN || cfg.Products.CDN {
		opts = append(opts, collector.WithCDN())
	}

	if *withObject || cfg.Products.OBJECT {
		opts = append(opts, collector.WithObject())
	}

	return opts
}
