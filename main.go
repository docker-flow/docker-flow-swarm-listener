package main

import (
	"./service"
	"time"
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
			n.ServicesCreate(newServices, args.Retry, args.RetryInterval)
			removedServices := s.GetRemovedServices(allServices)
			n.ServicesRemove(removedServices, args.Retry, args.RetryInterval)
			time.Sleep(time.Second * time.Duration(args.Interval))
		}
	}
}
