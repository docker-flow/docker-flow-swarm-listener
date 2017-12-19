package main

import (
	"encoding/json"
	"net/http"

	"./metrics"
	"./service"
	"github.com/prometheus/client_golang/prometheus"
)

var httpListenAndServe = http.ListenAndServe
var httpWriterSetContentType = func(w http.ResponseWriter, value string) {
	w.Header().Set("Content-Type", value)
}

// Serve is the instance structure
type Serve struct {
	Service      service.Servicer
	Notification service.Sender
}

//Response message
type Response struct {
	Status string
}

// NewServe returns a new instance of the `Serve`
func NewServe(service service.Servicer, notification service.Sender) *Serve {
	return &Serve{
		Service:      service,
		Notification: notification,
	}
}

// Run executes a server
func (m *Serve) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/docker-flow-swarm-listener/notify-services", m.NotifyServices)
	mux.HandleFunc("/v1/docker-flow-swarm-listener/get-services", m.GetServices)
	mux.HandleFunc("/v1/docker-flow-swarm-listener/ping", m.PingHandler)
	mux.Handle("/metrics", prometheus.Handler())
	return httpListenAndServe(":8080", mux)
}

// NotifyServices notifies all configured endpoints of new, updated, or removed services
func (m *Serve) NotifyServices(w http.ResponseWriter, req *http.Request) {
	services, _ := m.Service.GetServices()
	go m.Notification.ServicesCreate(services, 10, 5)
	js, _ := json.Marshal(Response{Status: "OK"})
	httpWriterSetContentType(w, "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}

// GetServices retrieves all services with the `com.df.notify` label set to `true`
func (m *Serve) GetServices(w http.ResponseWriter, req *http.Request) {
	services, _ := m.Service.GetServices()
	parameters := m.Service.GetServicesParameters(services)
	bytes, error := json.Marshal(parameters)
	if error != nil {
		logPrintf("ERROR: Unable to prepare response: %s", error)
		metrics.RecordError("serveGetServices")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		// NOTE: For an unknown reason, `httpWriterSetContentType` does not work so the header is set directly
		w.Header().Set("Content-Type", "application/json")
		httpWriterSetContentType(w, "application/json")
		w.Write(bytes)
	}
}

// PingHandler is used for health checks
func (m *Serve) PingHandler(w http.ResponseWriter, req *http.Request) {
	js, _ := json.Marshal(Response{Status: "OK"})
	httpWriterSetContentType(w, "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}
