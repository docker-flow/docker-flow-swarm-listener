package main

import (
	"./service"
	"time"
	"./metrics"
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
			if err != nil { metrics.RecordError("GetServices") }
			newServices, err := s.GetNewServices(allServices)
			if err != nil { metrics.RecordError("GetNewServices") }
			err = n.ServicesCreate(
				newServices,
				args.Retry,
				args.RetryInterval,
			)
			if err != nil {
				metrics.RecordError("ServicesCreate")
			}
			removedServices := s.GetRemovedServices(allServices)
			err = n.ServicesRemove(removedServices, args.Retry, args.RetryInterval)
			if err != nil { metrics.RecordError("ServicesRemove") }
			time.Sleep(time.Second * time.Duration(args.Interval))
		}
	}
}

