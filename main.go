package main

import (
	"time"
)

func main() {
	logPrintf("Starting Docker Flow: Swarm Listener")
	service := NewServiceFromEnv()
	serve := NewServe(service)
	serve.Run()

	args := GetArgs()
	t := time.NewTicker(time.Second * time.Duration(args.Interval))
	logPrintf("Starting iterations")
	for {
		if len(service.NotifCreateServiceUrl) > 0 {
			allServices, _ := service.GetServices()
			newServices, _ := service.GetNewServices(allServices)
			service.NotifyServicesCreate(newServices, args.Retry, args.RetryInterval)
			removedServices := service.GetRemovedServices(allServices)
			service.NotifyServicesRemove(removedServices, args.Retry, args.RetryInterval)
		}
		<-t.C
	}
}
