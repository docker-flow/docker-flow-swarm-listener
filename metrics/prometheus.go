package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ServiceName = "swarm_listener"
	ErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "docker_flow",
			Name: "error",
			Help: "Error counter",
		},
		[]string{"service", "operation"},
	)
)

func init() {
	prometheus.MustRegister(ErrorCounter)
}

