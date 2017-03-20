package service

import (
	"fmt"
	"github.com/docker/docker/api/types/swarm"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	"os"
)

type Notification struct {
	CreateServiceAddr []string
	RemoveServiceAddr []string
}

func NewNotification(createServiceAddr, removeServiceAddr []string) *Notification {
	return &Notification{
		CreateServiceAddr: createServiceAddr,
		RemoveServiceAddr: removeServiceAddr,
	}
}

func NewNotificationFromEnv() *Notification {
	createServiceAddr, removeServiceAddr := getSenderAddressesFromEnvVars("notification", "notify", "notif")
	return NewNotification(createServiceAddr, removeServiceAddr)
}

func (m *Notification) ServicesCreate(services *[]swarm.Service, retries, interval int) error {
	errs := []error{}
	for _, s := range *services {
		if _, ok := s.Spec.Labels[os.Getenv("DF_NOTIFY_LABEL")]; ok {
			parameters := url.Values{}
			parameters.Add("serviceName", s.Spec.Name)
			for k, v := range s.Spec.Labels {
				if strings.HasPrefix(k, "com.df") && k != os.Getenv("DF_NOTIFY_LABEL") {
					parameters.Add(strings.TrimPrefix(k, "com.df."), v)
				}
			}
			for _, addr := range m.CreateServiceAddr {
				urlObj, err := url.Parse(addr)
				if err != nil {
					logPrintf("ERROR: %s", err.Error())
					errs = append(errs, err)
					break
				}
				urlObj.RawQuery = parameters.Encode()
				fullUrl := urlObj.String()
				logPrintf("Sending service created notification to %s", fullUrl)
				for i := 1; i <= retries; i++ {
					resp, err := http.Get(fullUrl)
					if err == nil && resp.StatusCode == http.StatusOK {
						break
					} else if i < retries {
						if interval > 0 {
							t := time.NewTicker(time.Second * time.Duration(interval))
							<-t.C
						}
					} else {
						if err != nil {
							logPrintf("ERROR: %s", err.Error())
							errs = append(errs, err)
						} else if resp.StatusCode != http.StatusOK {
							body, _ := ioutil.ReadAll(resp.Body)
							msg := fmt.Errorf("Request %s returned status code %d\n%s", fullUrl, resp.StatusCode, string(body[:]))
							logPrintf("ERROR: %s", msg.Error())
							errs = append(errs, msg)
						}
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

func (m *Notification) ServicesRemove(remove *[]string, retries, interval int) error {
	errs := []error{}
	for _, v := range *remove {
		parameters := url.Values{}
		parameters.Add("serviceName", v)
		parameters.Add("distribute", "true")
		for _, addr := range m.RemoveServiceAddr {
			urlObj, err := url.Parse(addr)
			if err != nil {
				logPrintf("ERROR: %s", err.Error())
				errs = append(errs, err)
				break
			}
			urlObj.RawQuery = parameters.Encode()
			fullUrl := urlObj.String()
			logPrintf("Sending service removed notification to %s", fullUrl)
			for i := 1; i <= retries; i++ {
				resp, err := http.Get(fullUrl)
				if err == nil && resp.StatusCode == http.StatusOK {
					delete(Services, v)
					break
				} else if i < retries {
					if interval > 0 {
						t := time.NewTicker(time.Second * time.Duration(interval))
						<-t.C
					}
				} else {
					if err != nil {
						logPrintf("ERROR: %s", err.Error())
						errs = append(errs, err)
					} else if resp.StatusCode != http.StatusOK {
						msg := fmt.Errorf("Request %s returned status code %d", fullUrl, resp.StatusCode)
						logPrintf("ERROR: %s", msg)
						errs = append(errs, msg)
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
