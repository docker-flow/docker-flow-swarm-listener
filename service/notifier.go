package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"../metrics"
)

// NotifyType is the type of notification to send
type NotifyType string

// NotificationSender sends notifications to listeners
type NotificationSender interface {
	Create(ctx context.Context, params string) error
	Remove(ctx context.Context, params string) error
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
func (n Notifier) Create(ctx context.Context, params string) error {
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
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		n.log.Printf("ERROR: Incorrect fullURL: %s", fullURL)
		metrics.RecordError(n.createErrorMetric)
		return err
	}
	req = req.WithContext(ctx)

	n.log.Printf("Sending %s created notification to %s", n.notifyType, fullURL)
	retryChan := make(chan int, 1)
	retryChan <- 1
	for {
		select {
		case i := <-retryChan:
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				if strings.Contains(err.Error(), "context") {
					n.log.Printf("Canceling %s create notification to %s", n.notifyType, fullURL)
					return nil
				}
				if i <= n.retries && n.interval > 0 {
					n.log.Printf("Retrying %s created notification to %s (%d try)", n.notifyType, fullURL, i)
					time.Sleep(time.Second * time.Duration(n.interval))
					retryChan <- i + 1
					continue
				} else {
					n.log.Printf("ERROR: %v", err)
					metrics.RecordError(n.createErrorMetric)
					return err
				}
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict {
				return nil
			} else if i <= n.retries && n.interval > 0 {
				n.log.Printf("Retrying %s created notification to %s (%d try)", n.notifyType, fullURL, i)
				time.Sleep(time.Second * time.Duration(n.interval))
				retryChan <- i + 1
				continue
			} else if resp.StatusCode == http.StatusConflict || resp.StatusCode != http.StatusOK {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					err = fmt.Errorf("Failed at retrying request to %s returned status code %d", fullURL, resp.StatusCode)
					n.log.Printf("ERROR: %v", err)
					metrics.RecordError(n.createErrorMetric)
					return err
				}
				err = fmt.Errorf("Failed at retrying request to %s returned status code %d\n%s", fullURL, resp.StatusCode, string(body[:]))
				n.log.Printf("ERROR: %v", err)
				metrics.RecordError(n.createErrorMetric)
				return err
			}
			err = fmt.Errorf("Failed at retrying request to %s returned status code %d", fullURL, resp.StatusCode)
			n.log.Printf("ERROR: %v", err)
			metrics.RecordError(n.createErrorMetric)
			return err
		case <-ctx.Done():
			n.log.Printf("Canceling %s create notification to %s", n.notifyType, fullURL)
			return nil
		}

	}
}

// Remove sends remove notifications to listeners
func (n Notifier) Remove(ctx context.Context, params string) error {
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
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		n.log.Printf("ERROR: Incorrect fullURL: %s", fullURL)
		metrics.RecordError(n.removeErrorMetric)
		return err
	}
	req = req.WithContext(ctx)

	n.log.Printf("Sending %s removed notification to %s", n.notifyType, fullURL)
	retryChan := make(chan int, 1)
	retryChan <- 1
	for {
		select {
		case i := <-retryChan:
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				if strings.Contains(err.Error(), "context") {
					n.log.Printf("Canceling %s remove notification to %s", n.notifyType, fullURL)
					return nil
				}
				if i <= n.retries && n.interval > 0 {
					n.log.Printf("Retrying %s removed notification to %s (%d try)", n.notifyType, fullURL, i)
					time.Sleep(time.Second * time.Duration(n.interval))
					retryChan <- i + 1
					continue
				} else {
					n.log.Printf("ERROR: %v", err)
					metrics.RecordError(n.removeErrorMetric)
					return err
				}
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return nil
			} else if i <= n.retries && n.interval > 0 {
				n.log.Printf("Retrying %s removed notification to %s (%d try)", n.notifyType, fullURL, i)
				time.Sleep(time.Second * time.Duration(n.interval))
				retryChan <- i + 1
				continue
			} else {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					err = fmt.Errorf("Failed at retrying request to %s returned status code %d", fullURL, resp.StatusCode)
					n.log.Printf("ERROR: %v", err)
					metrics.RecordError(n.removeErrorMetric)
					return err
				}
				err = fmt.Errorf("Failed at retrying request to %s returned status code %d\n%s", fullURL, resp.StatusCode, string(body[:]))
				n.log.Printf("ERROR: %v", err)
				metrics.RecordError(n.removeErrorMetric)
				return err
			}
		case <-ctx.Done():
			n.log.Printf("Canceling %s remove notification to %s", n.notifyType, fullURL)
			return nil
		}
	}
}
