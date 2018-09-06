package main

import (
	"log"
	"os"

	"github.com/docker-flow/docker-flow-swarm-listener/service"
)

func main() {
	l := log.New(os.Stdout, "", log.LstdFlags)

	l.Printf("Starting Docker Flow: Swarm Listener")
	args := getArgs()
	swarmListener, err := service.NewSwarmListenerFromEnv(
		args.Retry, args.RetryInterval,
		args.ServicePollingInterval, args.NodePollingInterval, l)
	if err != nil {
		l.Printf("Failed to initialize Docker Flow: Swarm Listener")
		l.Printf("ERROR: %v", err)
		return
	}

	l.Printf("Sending notifications for running services and nodes")
	swarmListener.NotifyServices(true)
	swarmListener.NotifyNodes(true)

	swarmListener.Run()
	serve := NewServe(swarmListener, l)
	l.Fatal(Run(serve))
}
