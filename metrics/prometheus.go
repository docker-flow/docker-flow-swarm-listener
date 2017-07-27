package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var serviceName = "swarm_listener"
var errorCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: "docker_flow",
		Name: "error",
		Help: "Error counter",
	},
	[]string{"service", "operation"},
)

func init() {
	prometheus.MustRegister(errorCounter)
}

func RecordError(operation string) {
	errorCounter.With(prometheus.Labels{
		"service":   serviceName,
		"operation": operation,
	}).Inc()
}