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
	if len(service.NotifyCreateServiceUrl) > 0 {
		logPrintf("Starting iterations")
		for {
			allServices, _ := service.GetServices()
			newServices, _ := service.GetNewServices(allServices)
			service.NotifyServicesCreate(newServices, args.Retry, args.RetryInterval)
			removedServices := service.GetRemovedServices(allServices)
			service.NotifyServicesRemove(removedServices, args.Retry, args.RetryInterval)
			time.Sleep(time.Second * time.Duration(args.Interval))
		}
	}
}
