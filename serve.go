package main

import (
	"encoding/json"
	"log"
	"net/http"

	"./metrics"
	"./service"
	"github.com/prometheus/client_golang/prometheus"
)

var httpListenAndServe = http.ListenAndServe
var httpWriterSetContentType = func(w http.ResponseWriter, value string) {
	w.Header().Set("Content-Type", value)
}

//Response message
type Response struct {
	Status string
}

// Serve is the instance structure
type Serve struct {
	SwarmListener service.SwarmListening
	Log           *log.Logger
}

// NewServe returns a new instance of the `Serve`
func NewServe(swarmListener service.SwarmListening, logger *log.Logger) *Serve {
	return &Serve{
		SwarmListener: swarmListener,
		Log:           logger,
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
	m.SwarmListener.NotifyServices(false)
	js, _ := json.Marshal(Response{Status: "OK"})
	httpWriterSetContentType(w, "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}

// GetServices retrieves all services with the `com.df.notify` label set to `true`
func (m *Serve) GetServices(w http.ResponseWriter, req *http.Request) {
	parameters, err := m.SwarmListener.GetServicesParameters(req.Context())
	if err != nil {
		m.Log.Printf("ERROR: Unable to prepare response: %s", err)
		metrics.RecordError("serveGetServices")
		w.WriteHeader(http.StatusInternalServerError)
	}
	bytes, err := json.Marshal(parameters)
	if err != nil {
		m.Log.Printf("ERROR: Unable to prepare response: %s", err)
		metrics.RecordError("serveGetServices")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		// NOTE: For an unknown reason, `httpWriterSetContentType` does not work so the header is set directly
		w.Header().Set("Content-Type", "application/json")
		httpWriterSetContentType(w, "application/json")
		w.Write(bytes)
	}
}

// GetNodes retrieves all nodes
func (m *Serve) GetNodes(w http.ResponseWriter, req *http.Request) {
	parameters, err := m.SwarmListener.GetNodesParameters(req.Context())
	if err != nil {
		m.Log.Printf("ERROR: Unable to prepare response: %s", err)
		metrics.RecordError("serveGetNodes")
		w.WriteHeader(http.StatusInternalServerError)
	}
	bytes, err := json.Marshal(parameters)
	if err != nil {
		m.Log.Printf("ERROR: Unable to prepare response: %s", err)
		metrics.RecordError("serveGetNodes")
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
