package service

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"../metrics"
)

// NotifyType is the type of notification to send
type NotifyType string

// NotificationSender sends notifications to listeners
type NotificationSender interface {
	Create(params string) error
	Remove(params string) error
	GetCreateAddr() string
	GetRemoveAddr() string
}

// Notifier implements `NotificationSender`
type Notifier struct {
	createAddr        string
	removeAddr        string
	notifyType        string
	retries           int
	interval          int
	createErrorMetric string
	removeErrorMetric string
	log               *log.Logger
}

// NewNotifier returns a `Notifier`
func NewNotifier(
	createAddr, removeAddr, notifyType string,
	retries int, interval int, logger *log.Logger) *Notifier {
	return &Notifier{
		createAddr:        createAddr,
		removeAddr:        removeAddr,
		notifyType:        notifyType,
		retries:           retries,
		interval:          interval,
		createErrorMetric: fmt.Sprintf("notificationSendCreate%sRequest", notifyType),
		removeErrorMetric: fmt.Sprintf("notificationSendRemove%sRequest", notifyType),
		log:               logger,
	}
}

// GetCreateAddr returns create addresses
func (n Notifier) GetCreateAddr() string {
	return n.createAddr
}

// GetRemoveAddr returns create addresses
func (n Notifier) GetRemoveAddr() string {
	return n.removeAddr
}

// Create sends create notifications to listeners
func (n Notifier) Create(params string) error {
	if len(n.createAddr) == 0 {
		return nil
	}

	urlObj, err := url.Parse(n.createAddr)
	if err != nil {
		n.log.Printf("ERROR: %v", err)
		metrics.RecordError(n.createErrorMetric)
		return err
	}
	urlObj.RawQuery = params
	fullURL := urlObj.String()
	n.log.Printf("Sending %s created notification to %s", n.notifyType, fullURL)
	for i := 1; i <= n.retries; i++ {
		resp, err := http.Get(fullURL)
		if err == nil &&
			(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict) {
			return nil
		} else if i < n.retries {
			if n.interval > 0 {
				n.log.Printf("Retrying %s created notification to %s (%d try)", n.notifyType, fullURL, i)
				time.Sleep(time.Second * time.Duration(n.interval))
			}
		} else {
			if err != nil {
				n.log.Printf("ERROR: %v", err)
				metrics.RecordError(n.createErrorMetric)
				return err
			} else if resp.StatusCode == http.StatusConflict || resp.StatusCode != http.StatusOK {
				body, _ := ioutil.ReadAll(resp.Body)
				err := fmt.Errorf("Request %s returned status code %d\n%s", fullURL, resp.StatusCode, string(body[:]))
				n.log.Printf("ERROR: %v", err)
				metrics.RecordError(n.createErrorMetric)
				return err
			}
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}
	return nil
}

// Remove sends remove notifications to listeners
func (n Notifier) Remove(params string) error {
	if len(n.removeAddr) == 0 {
		return nil
	}

	urlObj, err := url.Parse(n.removeAddr)
	if err != nil {
		n.log.Printf("ERROR: %v", err)
		metrics.RecordError(n.removeErrorMetric)
		return err
	}
	urlObj.RawQuery = params
	fullURL := urlObj.String()
	n.log.Printf("Sending %s removed notification to %s", n.notifyType, fullURL)
	for i := 1; i <= n.retries; i++ {
		resp, err := http.Get(fullURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		} else if i < n.retries {
			if n.interval > 0 {
				n.log.Printf("Retrying %s removed notification to %s (%d try)", n.notifyType, fullURL, i)
				time.Sleep(time.Second * time.Duration(n.interval))
			}
		} else {
			if err != nil {
				n.log.Printf("ERROR: %v", err)
				metrics.RecordError(n.removeErrorMetric)
				return err
			} else if resp.StatusCode != http.StatusOK {
				body, _ := ioutil.ReadAll(resp.Body)
				err := fmt.Errorf("Request %s returned status code %d\n%s", fullURL, resp.StatusCode, string(body[:]))
				n.log.Printf("ERROR: %v", err)
				metrics.RecordError(n.removeErrorMetric)
				return err
			}
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}
	return nil
}
