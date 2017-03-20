package service

import (
	"github.com/docker/docker/api/types/swarm"
	"net/http"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Alert struct {
	CreateServiceAddr []string
	RemoveServiceAddr []string
}

type AlertBody struct {
	Name string
	If   string
	For  string
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
	// TODO: retries
	errs := []error{}
	client := &http.Client{}
	for _, s := range *services {
		// TODO: Implement multiple alerts per service
		// TODO: Add shortcut IF statements (e.g. @memLimit)
		alertName := s.Spec.Labels["com.df.alert.name"]
		ifStatement := s.Spec.Labels["com.df.alert.if"]
		if len(alertName) > 0 && len(ifStatement) > 0 {
			name := fmt.Sprintf("%s%s", s.Spec.Name, alertName)
			name = strings.Replace(name, "-", "", -1)
			name = strings.Replace(name, "_", "", -1)
			js, _ := json.Marshal(AlertBody{
				Name: name,
				If:   ifStatement,
				For:  s.Spec.Labels["com.df.alert.for"],
			})
			body := bytes.NewBuffer(js)
			// TODO: Move to util and use in notification.go
			for _, addr := range m.CreateServiceAddr {
				for i := 0; i <= retries; i++ {
					req, err := http.NewRequest("PUT", addr, body)
					if err != nil {
						logPrintf(err.Error())
						errs = append(errs, err)
						break
					}
					resp, err := client.Do(req)
					if err == nil && resp.StatusCode == http.StatusOK {
						break
					} else if i < retries {
						if interval > 0 {
							t := time.NewTicker(time.Second * time.Duration(interval))
							<-t.C
						}
					} else if err != nil {
						logPrintf(err.Error())
						errs = append(errs, err)
					} else {
						logPrintf("%s returned status %d", addr, resp.StatusCode)
						errs = append(errs, err)
					}
				}
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("At least one request produced errors. Please consult logs for more details.")
	}
	return nil
}

func (m *Alert) ServicesRemove(remove *[]string, retries, interval int) error {
	// TODO: Implement
	return nil
}