package main

import (
	"time"
	"./service"
)

func main() {
	logPrintf("Starting Docker Flow: Swarm Listener")
	s := service.NewServiceFromEnv()
	n := service.NewNotificationFromEnv()
	serve := NewServe(s, n)
	go serve.Run()

	args := GetArgs()
	if len(n.NotifyCreateServiceUrl) > 0 {
		logPrintf("Starting iterations")
		for {
			allServices, _ := s.GetServices()
			newServices, _ := s.GetNewServices(allServices)
			n.NotifyServicesCreate(newServices, args.Retry, args.RetryInterval)
			removedServices := s.GetRemovedServices(allServices)
			n.NotifyServicesRemove(removedServices, args.Retry, args.RetryInterval)
			time.Sleep(time.Second * time.Duration(args.Interval))
		}
	}
}
