package service

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"../metrics"
)

// Notification defines the structure with exported functions
type Notification struct {
	CreateServiceAddr []string
	RemoveServiceAddr []string
}

func newNotification(createServiceAddr, removeServiceAddr []string) *Notification {
	return &Notification{
		CreateServiceAddr: createServiceAddr,
		RemoveServiceAddr: removeServiceAddr,
	}
}

// NewNotificationFromEnv returns `notification` instance
func NewNotificationFromEnv() *Notification {
	createServiceAddr, removeServiceAddr := getSenderAddressesFromEnvVars("notification", "notify", "notif")
	return newNotification(createServiceAddr, removeServiceAddr)
}

// ServicesCreate sends create service notifications
func (m *Notification) ServicesCreate(services *[]SwarmService, retries, interval int) error {
	for _, s := range *services {
		if _, ok := s.Spec.Labels[os.Getenv("DF_NOTIFY_LABEL")]; ok {
			params := getServiceParams(&s)
			urlValues := url.Values{}
			for k, v := range params {
				urlValues.Add(k, v)
			}
			for _, addr := range m.GetCreateServiceAddr(urlValues) {
				go m.sendCreateServiceRequest(s.Spec.Name, addr, urlValues, retries, interval)
			}
		}
	}
	return nil
}

// GetCreateServiceAddr returns create service addresses
func (m *Notification) GetCreateServiceAddr(urlValues map[string][]string) []string {
	if val, ok := urlValues["notifyService"]; ok {
		addresses := []string{}
		services := strings.Split(val[0], ",")
		for _, s := range services {
			for _, addr := range m.CreateServiceAddr {
				if strings.Contains(addr, s) {
					addresses = append(addresses, addr)
					break
				}
			}
		}
		return addresses
	}
	return m.CreateServiceAddr
}

// ServicesRemove sends remove service notifications
func (m *Notification) ServicesRemove(remove *[]string, retries, interval int) error {
	errs := []error{}
	for _, v := range *remove {
		parameters := url.Values{}
		parameters.Add("serviceName", v)
		parameters.Add("distribute", "true")
		for _, addr := range m.GetRemoveServiceAddr(parameters) {
			urlObj, err := url.Parse(addr)
			if err != nil {
				logPrintf("ERROR: %s", err.Error())
				errs = append(errs, err)
				break
			}
			urlObj.RawQuery = parameters.Encode()
			fullURL := urlObj.String()
			logPrintf("Sending service removed notification to %s", fullURL)
			for i := 1; i <= retries; i++ {
				resp, err := http.Get(fullURL)
				if err == nil && resp.StatusCode == http.StatusOK {
					delete(CachedServices, v)
					break
				} else if i < retries {
					if interval > 0 {
						t := time.NewTicker(time.Second * time.Duration(interval))
						<-t.C
					}
				} else {
					if err != nil {
						logPrintf("ERROR: %s", err.Error())
						metrics.RecordError("notificationServicesRemove")
						errs = append(errs, err)
					} else if resp.StatusCode != http.StatusOK {
						msg := fmt.Errorf("Request %s returned status code %d", fullURL, resp.StatusCode)
						logPrintf("ERROR: %s", msg)
						metrics.RecordError("notificationServicesRemove")
						errs = append(errs, msg)
					}
				}
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("At least one request produced errors. Please consult logs for more details.")
	}
	return nil
}

// GetRemoveServiceAddr returns remove service addresses
func (m *Notification) GetRemoveServiceAddr(urlValues map[string][]string) []string {
	return m.RemoveServiceAddr
}

func (m *Notification) sendCreateServiceRequest(serviceName, addr string, params url.Values, retries, interval int) {
	urlObj, err := url.Parse(addr)
	if err != nil {
		logPrintf("ERROR: %s", err.Error())
		metrics.RecordError("notificationSendCreateServiceRequest")
		return
	}
	urlObj.RawQuery = params.Encode()
	fullUrl := urlObj.String()
	logPrintf("Sending service created notification to %s", fullUrl)
	for i := 1; i <= retries; i++ {
		if _, ok := CachedServices[serviceName]; !ok {
			logPrintf("Service %s was removed. Service created notifications are stopped.", serviceName)
			break
		}
		resp, err := http.Get(fullUrl)
		if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict) {
			break
		} else if i < retries {
			logPrintf("Retrying service created notification to %s", fullUrl)
			if interval > 0 {
				t := time.NewTicker(time.Second * time.Duration(interval))
				<-t.C
			}
		} else {
			if err != nil {
				logPrintf("ERROR: %s", err.Error())
				metrics.RecordError("notificationSendCreateServiceRequest")
			} else if resp.StatusCode == http.StatusConflict {
				body, _ := ioutil.ReadAll(resp.Body)
				logPrintf(fmt.Sprintf("Request %s returned status code %d\n%s", fullUrl, resp.StatusCode, string(body[:])))
				metrics.RecordError("notificationSendCreateServiceRequest")
			} else if resp.StatusCode != http.StatusOK {
				body, _ := ioutil.ReadAll(resp.Body)
				msg := fmt.Errorf("Request %s returned status code %d\n%s", fullUrl, resp.StatusCode, string(body[:]))
				logPrintf("ERROR: %s", msg.Error())
				metrics.RecordError("notificationSendCreateServiceRequest")
			}
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}
}
