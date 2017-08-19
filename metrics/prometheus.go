package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var serviceName = "swarm_listener"
var errorCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: "docker_flow",
		Name:      "error",
		Help:      "Error counter",
	},
	[]string{"service", "operation"},
)

var serviceGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: "docker_flow",
		Name:      "service_count",
		Help:      "Service gauge",
	},
	[]string{"service"},
)

func init() {
	prometheus.MustRegister(errorCounter, serviceGauge)
}

func RecordError(operation string) {
	errorCounter.With(prometheus.Labels{
		"service":   serviceName,
		"operation": operation,
	}).Inc()
}

func RecordService(count int) {
	serviceGauge.With(prometheus.Labels{
		"service": serviceName,
	}).Set(float64(count))
}
