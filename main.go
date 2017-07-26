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
			recordWhenError("GetServices", err)
			newServices, err := s.GetNewServices(allServices)
			recordWhenError("GetNewServices", err)
			err = n.ServicesCreate(newServices, args.Retry, args.RetryInterval)
			recordWhenError("ServicesCreate", err)
			removedServices := s.GetRemovedServices(allServices)
			err = n.ServicesRemove(removedServices, args.Retry, args.RetryInterval)
			recordWhenError("ServicesRemove", err)
			time.Sleep(time.Second * time.Duration(args.Interval))
		}
	}
}

func recordWhenError(operation string, err error) {
	if err != nil {
		metrics.ErrorCounter.With(prometheus.Labels{
			"service":   metrics.ServiceName,
			"operation": operation,
		}).Inc()
	}
}
