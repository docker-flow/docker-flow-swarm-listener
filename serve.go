package main

import (
	"./service"
	"encoding/json"
	"net/http"
	"github.com/prometheus/client_golang/prometheus"
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
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/docker-flow-swarm-listener/notify-services", m.NotifyServices)
	mux.HandleFunc("/v1/docker-flow-swarm-listener/get-services", m.GetServices)
	mux.Handle("/metrics", prometheus.Handler())
	if err := httpListenAndServe(":8080", mux); err != nil {
		return err
	}
	return nil
}

func (m *Serve) NotifyServices(w http.ResponseWriter, req *http.Request) {
	services, _ := m.Service.GetServices()
	go m.Notification.ServicesCreate(services, 10, 5)
	// TODO: Add response message
	httpWriterSetContentType(w, "application/json")
}

func (m *Serve) GetServices(w http.ResponseWriter, req *http.Request) {
	services, _ := m.Service.GetServices()
	parameters := m.Service.GetServicesParameters(services)
	bytes, error := json.Marshal(parameters)
	if error != nil {
		logPrintf("ERROR: Unable to prepare response: %s", error)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.Write(bytes)
	}
	httpWriterSetContentType(w, "application/json")
}

func NewServe(service service.Servicer, notification service.Sender) *Serve {
	return &Serve{
		Service:      service,
		Notification: notification,
	}
}
