package main

import (
	"./metrics"
	"./service"
)

func main() {
	logPrintf("Starting Docker Flow: Swarm Listener")
	s := service.NewServiceFromEnv()
	n := service.NewNotificationFromEnv()
	el := service.NewEventListenerFromEnv()
	serve := NewServe(s, n)
	go serve.Run()

	args := getArgs()
	if len(n.CreateServiceAddr) == 0 {
		return
	}

	logPrintf("Sending notifications for running services")
	allServices, err := s.GetServices()
	if err != nil {
		metrics.RecordError("GetServices")
	}

	newServices, err := s.GetNewServices(allServices)
	if err != nil {
		metrics.RecordError("GetNewServices")
	}
	err = n.ServicesCreate(
		newServices,
		args.Retry,
		args.RetryInterval,
	)
	if err != nil {
		metrics.RecordError("ServicesCreate")
	}

	logPrintf("Start listening to docker service events")
	events, errs := el.ListenForEvents()
	for {
		select {
		case event := <-events:
			if event.Action == "create" || event.Action == "update" {
				eventServices, err := s.GetServicesFromID(event.ServiceID)
				if err != nil {
					metrics.RecordError("GetServicesFromID")
				}
				newServices, err := s.GetNewServices(eventServices)
				if err != nil {
					metrics.RecordError("GetNewServices")
				}
				err = n.ServicesCreate(
					newServices,
					args.Retry,
					args.RetryInterval,
				)
				if err != nil {
					metrics.RecordError("ServicesCreate")
				}

			} else if event.Action == "remove" {
				err = n.ServicesRemove(&[]string{event.ServiceID}, args.Retry, args.RetryInterval)
				metrics.RecordService(len(service.CachedServices))
				if err != nil {
					metrics.RecordError("ServicesRemove")
				}
			}
		case <-errs:
			metrics.RecordError("ListenForEvents")
			// Restart listening for events
			events, errs = el.ListenForEvents()
		}
	}
}
