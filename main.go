package main

import (
	"time"
)

func main() {
	logPrintf("Starting Docker Flow: Swarm Listener")
	service := NewServiceFromEnv()
	args := GetArgs()
	t := time.NewTicker(time.Second * time.Duration(args.Interval))
	for {
		if len(service.NotifUrl) > 0 {
			services, _ := service.GetNewServices()
			service.NotifyServices(services, args.Retry, args.RetryInterval)
		}
		<-t.C
		logPrintf("Tick")
	}
}
