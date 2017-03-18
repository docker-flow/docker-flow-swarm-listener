package service

import (
	"github.com/docker/docker/api/types/swarm"
	"net/http"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type Alert struct {
	CreateServiceAddr []string
	RemoveServiceAddr []string
}

type AlertBody struct {
	Name string
}

func NewAlert(createServiceAddr, removeServiceAddr []string) *Alert {
	return &Alert{
		CreateServiceAddr: createServiceAddr,
		RemoveServiceAddr: removeServiceAddr,
	}
}

func NewAlertFromEnv() *Notification {
	createServiceAddr, removeServiceAddr := getSenderAddressesFromEnvVars("alert", "alert", "")
	return NewNotification(createServiceAddr, removeServiceAddr)
}

func (m *Alert) ServicesCreate(services *[]swarm.Service, retries, interval int) error {
	client := &http.Client{}
	for _, s := range *services {
		// TODO: Fail if com.df.alert.name is empty
		// TODO: Add IF
		// TODO: Fail if com.df.alert.if is empty
		// TODO: Add FOR (optional)
		// TODO: Implement multiple alerts per service
		// TODO: Add shortcut IF statements (e.g. @memLimit)
		name := fmt.Sprintf("%s%s", s.Spec.Name, s.Spec.Labels["com.df.alert.name"])
		name = strings.Replace(name, "-", "", -1)
		name = strings.Replace(name, "_", "", -1)
		js, _ := json.Marshal(AlertBody{
			Name: name,
		})
		for _, addr := range m.CreateServiceAddr {
			req, _ := http.NewRequest("PUT", addr, bytes.NewBuffer(js))
			// TODO: error
			client.Do(req)
		}
	}
	return nil
}

func (m *Alert) ServicesRemove(remove *[]string, retries, interval int) error {
	return nil
}