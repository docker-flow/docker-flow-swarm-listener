package main

import (
	"./service"
	"time"
	"./metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	logPrintf("Starting Docker Flow: Swarm Listener")
	s := service.NewServiceFromEnv()
	n := service.NewNotificationFromEnv()
	serve := NewServe(s, n)
	go serve.Run()

	args := GetArgs()
	if len(n.CreateServiceAddr) > 0 {
		logPrintf("Starting iterations")
		for {
			allServices, err := s.GetServices()
			if err != nil { recordError("GetServices") }
			newServices, err := s.GetNewServices(allServices)
			if err != nil { recordError("GetNewServices") }
			err = n.ServicesCreate(newServices, args.Retry, args.RetryInterval)
			if err != nil { recordError("ServicesCreate") }
			removedServices := s.GetRemovedServices(allServices)
			err = n.ServicesRemove(removedServices, args.Retry, args.RetryInterval)
			if err != nil { recordError("ServicesRemove") }
			time.Sleep(time.Second * time.Duration(args.Interval))
		}
	}
}

func recordError(operation string) {
	metrics.ErrorCounter.With(prometheus.Labels{
		"service":   metrics.ServiceName,
		"operation": operation,
	}).Inc()
}
