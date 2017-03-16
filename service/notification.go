package service

import (
	"os"
	"strings"
	"github.com/docker/docker/api/types/swarm"
	"net/url"
	"net/http"
	"time"
	"io/ioutil"
	"fmt"
)

type Notifier interface {
	NotifyServicesCreate(services []swarm.Service, retries, interval int) error
	NotifyServicesRemove(services []string, retries, interval int) error
}

type Notification struct {
	NotifyCreateServiceUrl []string
	NotifyRemoveServiceUrl []string
}

func NewNotification(createServiceAddr, removeServiceAddr []string) *Notification {
	return &Notification{
		NotifyCreateServiceUrl: createServiceAddr,
		NotifyRemoveServiceUrl: removeServiceAddr,
	}
}

func NewNotificationFromEnv() *Notification {
	var createServiceAddr []string
	var removeServiceAddr []string
	if len(os.Getenv("DF_NOTIFY_CREATE_SERVICE_URL")) > 0 {
		createServiceAddr = strings.Split(os.Getenv("DF_NOTIFY_CREATE_SERVICE_URL"), ",")
	} else if len(os.Getenv("DF_NOTIF_CREATE_SERVICE_URL")) > 0 { // Deprecated since dec. 2016
		createServiceAddr = strings.Split(os.Getenv("DF_NOTIF_CREATE_SERVICE_URL"), ",")
	} else {
		createServiceAddr = strings.Split(os.Getenv("DF_NOTIFICATION_URL"), ",")
	}
	if len(os.Getenv("DF_NOTIFY_REMOVE_SERVICE_URL")) > 0 {
		removeServiceAddr = strings.Split(os.Getenv("DF_NOTIFY_REMOVE_SERVICE_URL"), ",")
	} else if len(os.Getenv("DF_NOTIF_REMOVE_SERVICE_URL")) > 0 { // Deprecated since dec. 2016
		removeServiceAddr = strings.Split(os.Getenv("DF_NOTIF_REMOVE_SERVICE_URL"), ",")
	} else {
		removeServiceAddr = strings.Split(os.Getenv("DF_NOTIFICATION_URL"), ",")
	}
	return NewNotification(createServiceAddr, removeServiceAddr)
}

func (m *Notification) NotifyServicesCreate(services []swarm.Service, retries, interval int) error {
	errs := []error{}
	for _, s := range services {
		if _, ok := s.Spec.Labels["com.df.notify"]; ok {
			parameters := url.Values{}
			parameters.Add("serviceName", s.Spec.Name)
			for k, v := range s.Spec.Labels {
				if strings.HasPrefix(k, "com.df") && k != "com.df.notify" {
					parameters.Add(strings.TrimPrefix(k, "com.df."), v)
				}
			}
			for _, addr := range m.NotifyCreateServiceUrl {
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
							logPrintf("ERROR: %s", msg)
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

func (m *Notification) NotifyServicesRemove(remove []string, retries, interval int) error {
	errs := []error{}
	for _, v := range remove {
		parameters := url.Values{}
		parameters.Add("serviceName", v)
		parameters.Add("distribute", "true")
		for _, addr := range m.NotifyRemoveServiceUrl {
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

