package main

import (
	"./service"
	"net/http"
)

var httpListenAndServe = http.ListenAndServe
var httpWriterSetContentType = func(w http.ResponseWriter, value string) {
	w.Header().Set("Content-Type", value)
}

type Server interface {
	Run()
}

type Serve struct {
	Service      service.Servicer
	Notification service.Sender
}

func (m *Serve) Run() error {
	if err := httpListenAndServe(":8080", m); err != nil {
		return err
	}
	return nil
}

func (m *Serve) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	httpWriterSetContentType(w, "application/json")
	switch req.URL.Path {
	case "/v1/docker-flow-swarm-listener/notify-services":
		services, _ := m.Service.GetServices()
		go m.Notification.ServicesCreate(services, 10, 5)
		// TODO: Add response message
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func NewServe(service service.Servicer, notification service.Sender) *Serve {
	return &Serve{
		Service:      service,
		Notification: notification,
	}
}
