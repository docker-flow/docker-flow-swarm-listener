package main

import (
	"time"
)

func main() {
	logPrintf("Starting Docker Flow: Swarm Listener")
	service := NewServiceFromEnv()
	serve := NewServe(service)
	go serve.Run()

	args := GetArgs()
	logPrintf("Starting iterations")
	for {
		if len(service.NotifCreateServiceUrl) > 0 {
			allServices, _ := service.GetServices()
			newServices, _ := service.GetNewServices(allServices)
			service.NotifyServicesCreate(newServices, args.Retry, args.RetryInterval)
			removedServices := service.GetRemovedServices(allServices)
			service.NotifyServicesRemove(removedServices, args.Retry, args.RetryInterval)
		}
		time.Sleep(time.Second * time.Duration(args.Interval))
	}
}
