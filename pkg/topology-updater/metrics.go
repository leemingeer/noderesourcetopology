package topologyupdater

import "github.com/prometheus/client_golang/prometheus"

const (
	scanErrorsQuery = "ppio_topology_updater_scan_errors_total"
)

var (
	scanErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: scanErrorsQuery,
		Help: "Number of errors in scanning resource allocation of pods.",
	})
)
