package main

import (
	"./service"
	"encoding/json"
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
	case "/v1/docker-flow-swarm-listener/get-services":
		services, _ := m.Service.GetServices()
		parameters := m.Service.GetServicesParameters(services)
		bytes, error := json.Marshal(parameters)
		if error != nil {
			logPrintf("ERROR: Unable to prepare response: %s", error)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Write(bytes)
			w.WriteHeader(http.StatusOK)
		}
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
