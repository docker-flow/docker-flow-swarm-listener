package service

import (
	"fmt"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type NotificationTestSuite struct {
	suite.Suite
}

func TestNotificationUnitTestSuite(t *testing.T) {
	s := new(NotificationTestSuite)
	logPrintfOrig := logPrintf
	defer func() {
		logPrintf = logPrintfOrig
		os.Unsetenv("DF_NOTIFY_LABEL")
	}()
	logPrintf = func(format string, v ...interface{}) {}
	os.Setenv("DF_NOTIFY_LABEL", "com.df.notify")
	suite.Run(t, s)
}

// NewNotification

func (s *NotificationTestSuite) Test_NewNotification_SetsCreateUrl() {
	expected := []string{"this-is-a-url", "this-is-a-different-url"}

	actual := newNotification(expected, []string{})

	s.Equal(expected, actual.CreateServiceAddr)
}

func (s *NotificationTestSuite) Test_NewNotification_SetsRemoveUrl() {
	expected := []string{"this-is-a-url", "this-is-a-different-url"}

	actual := newNotification([]string{}, expected)

	s.Equal(expected, actual.RemoveServiceAddr)
}

// NewNotificationFromEnv

func (s *NotificationTestSuite) Test_NewNotificationFromEnv_SetsNotifyCreateUrlFromEnvVars() {
	tests := []struct {
		envKey string
	}{
		{"DF_NOTIFY_CREATE_SERVICE_URL"},
		{"DF_NOTIF_CREATE_SERVICE_URL"},
		{"DF_NOTIFICATION_URL"},
	}
	for _, t := range tests {
		host := os.Getenv(t.envKey)
		expected := []string{"this-is-a-url", "this-is-a-different-url"}
		os.Setenv(t.envKey, strings.Join(expected, ","))

		actual := NewNotificationFromEnv()

		s.Equal(expected, actual.CreateServiceAddr)
		os.Setenv(t.envKey, host)
	}
}

func (s *NotificationTestSuite) Test_NewNotificationFromEnv_SetsNotifyRemoveUrlFromEnvVars() {
	tests := []struct {
		envKey string
	}{
		{"DF_NOTIFY_REMOVE_SERVICE_URL"},
		{"DF_NOTIF_REMOVE_SERVICE_URL"},
		{"DF_NOTIFICATION_URL"},
	}
	for _, t := range tests {
		host := os.Getenv(t.envKey)
		expected := []string{"this-is-a-url", "this-is-a-different-url"}
		os.Setenv(t.envKey, strings.Join(expected, ","))

		n := NewNotificationFromEnv()

		s.Equal(expected, n.RemoveServiceAddr, "Failed to fetch information from the env. var. %s.", t.envKey)
		os.Setenv(t.envKey, host)
	}
}

// ServicesCreate

func (s *NotificationTestSuite) Test_ServicesCreate_SendsRequests() {
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	labels["com.df.distribute"] = "true"
	labels["label.without.correct.prefix"] = "something"

	actualSent1 := false
	actualSent2 := false
	actualQuery1 := ""
	actualQuery2 := ""
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualPath := r.URL.Path
		if r.Method == "GET" {
			switch actualPath {
			case "/v1/docker-flow-proxy/reconfigure":
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				actualQuery1 = r.URL.RawQuery
				actualSent1 = true
			case "/something/else":
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				actualQuery2 = r.URL.RawQuery
				actualSent2 = true
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer func() { httpSrv.Close() }()
	url1 := fmt.Sprintf("%s/v1/docker-flow-proxy/reconfigure", httpSrv.URL)
	url2 := fmt.Sprintf("%s/something/else", httpSrv.URL)

	n := newNotification([]string{url1, url2}, []string{})
	n.ServicesCreate(s.getSwarmServices(labels), 1, 0)
	passed := false
	for i := 0; i < 100; i++ {
		if actualSent1 {
			s.Equal("distribute=true&serviceName=my-service", actualQuery1)
			passed = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.True(passed)
	passed = false
	for i := 0; i < 100; i++ {
		if actualSent2 {
			s.Equal("distribute=true&serviceName=my-service", actualQuery2)
			passed = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.True(passed)
}

func (s *NotificationTestSuite) Test_ServicesCreate_UsesShortServiceName() {
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	labels["com.df.distribute"] = "true"
	labels["com.df.shortName"] = "true"
	labels["com.docker.stack.namespace"] = "my-stack"
	ann := swarm.Annotations{
		Name:   "my-stack_my-service",
		Labels: labels,
	}
	spec := swarm.ServiceSpec{
		Annotations: ann,
	}
	srv := swarm.Service{
		Spec: spec,
	}
	CachedServices = map[string]SwarmService{}
	CachedServices[ann.Name] = SwarmService{srv}
	ss := SwarmService{srv}
	services := &[]SwarmService{ss}

	actualSent := false
	actualQuery := ""
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		actualQuery = r.URL.RawQuery
		actualSent = true
	}))
	defer func() { httpSrv.Close() }()
	url1 := fmt.Sprintf("%s/v1/docker-flow-proxy/reconfigure", httpSrv.URL)

	n := newNotification([]string{url1}, []string{})
	n.ServicesCreate(services, 1, 0)
	passed := false
	for i := 0; i < 100; i++ {
		if actualSent {
			s.Equal("distribute=true&serviceName=my-service&shortName=true", actualQuery)
			passed = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.True(passed)
}

func (s *NotificationTestSuite) Test_ServicesCreate_AddsReplicas() {
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	labels["com.df.distribute"] = "true"
	services := *s.getSwarmServices(labels)
	replicas := uint64(2)
	mode := swarm.ServiceMode{
		Replicated: &swarm.ReplicatedService{Replicas: &replicas},
	}
	services[0].Service.Spec.Mode = mode

	actualSent := false
	actualQuery := ""
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		actualQuery = r.URL.RawQuery
		actualSent = true
	}))
	defer func() { httpSrv.Close() }()
	url := fmt.Sprintf("%s/v1/docker-flow-proxy/reconfigure", httpSrv.URL)

	n := newNotification([]string{url}, []string{})
	n.ServicesCreate(&services, 1, 0)
	passed := false
	for i := 0; i < 1000; i++ {
		if actualSent {
			s.Equal("distribute=true&replicas=2&serviceName=my-service", actualQuery)
			passed = true
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	s.True(passed)
}

func (s *NotificationTestSuite) Test_ServicesCreate_UsesLabelsFromEnvVars() {
	notifyLabelOrig := os.Getenv("DF_NOTIFY_LABEL")
	defer func() { os.Setenv("DF_NOTIFY_LABEL", notifyLabelOrig) }()
	os.Setenv("DF_NOTIFY_LABEL", "com.df.something")

	labels := make(map[string]string)
	labels["com.df.something"] = "true"
	labels["com.df.distribute"] = "true"
	labels["label.without.correct.prefix"] = "something"

	s.verifyNotifyServiceCreate(labels, true, fmt.Sprintf("distribute=true&serviceName=%s", "my-service"))
}

func (s *NotificationTestSuite) Test_ServicesCreate_LogsError_WhenUrlCannotBeParsed() {
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	msg := ""
	logPrintfOrig := logPrintf
	defer func() { logPrintf = logPrintfOrig }()
	logPrintf = func(format string, v ...interface{}) {
		msg = format
	}

	n := newNotification([]string{"%%%"}, []string{})
	n.ServicesCreate(s.getSwarmServices(labels), 1, 0)

	for i := 0; i < 100; i++ {
		if strings.HasPrefix(msg, "ERROR") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.True(strings.HasPrefix(msg, "ERROR"))
}

func (s *NotificationTestSuite) Test_ServicesCreate_LogsError_WhenHttpStatusIsNot200() {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	msg := ""
	logPrintfOrig := logPrintf
	defer func() { logPrintf = logPrintfOrig }()
	logPrintf = func(format string, v ...interface{}) {
		msg = format
	}

	n := newNotification([]string{httpSrv.URL}, []string{})
	n.ServicesCreate(s.getSwarmServices(labels), 1, 0)

	for i := 0; i < 100; i++ {
		if strings.HasPrefix(msg, "ERROR") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.True(strings.HasPrefix(msg, "ERROR"))
}

func (s *NotificationTestSuite) Test_ServicesCreate_DoesNotReturnError_WhenHttpStatusIs409() {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"

	n := newNotification([]string{httpSrv.URL}, []string{})
	err := n.ServicesCreate(s.getSwarmServices(labels), 1, 0)

	s.NoError(err)
}

// TODO: Fails when running inside a container
//func (s *NotificationTestSuite) Test_ServicesCreate_LogsError_WhenHttpRequestReturnsError() {
//	labels := make(map[string]string)
//	labels["com.df.notify"] = "true"
//	logPrintfOrig := logPrintf
//	defer func() { logPrintf = logPrintfOrig }()
//	msg := ""
//	logPrintf = func(format string, v ...interface{}) {
//		msg = format
//	}
//
//	n := newNotification([]string{"this-does-not-exist"}, []string{})
//	n.ServicesCreate(s.getSwarmServices(labels), 1, 0)
//
//	for i := 0; i < 500; i++ {
//		if strings.HasPrefix(msg, "ERROR") {
//			break
//		}
//		time.Sleep(10 * time.Millisecond)
//	}
//	s.True(strings.HasPrefix(msg, "ERROR"))
//}

func (s *NotificationTestSuite) Test_ServicesCreate_RetriesRequests() {
	attempt := 0
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempt < 1 {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
		}
		attempt += 1
	}))

	n := newNotification([]string{httpSrv.URL}, []string{})
	err := n.ServicesCreate(s.getSwarmServices(labels), 2, 1)

	s.NoError(err)
}

func (s *NotificationTestSuite) Test_ServicesCreate_StopsSendingNotifications_WhenServiceIsRemoved() {
	attempt := 0
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		attempt++
		if attempt == 1 {
			delete(CachedServices, "my-service")
		}
	}))

	n := newNotification([]string{httpSrv.URL}, []string{})
	n.ServicesCreate(s.getSwarmServices(labels), 5, 0)

	time.Sleep(2 * time.Millisecond)
	s.Equal(1, attempt)
}

// ServicesRemove

func (s *NotificationTestSuite) Test_ServicesRemove_SendsRequests() {
	CachedServices = make(map[string]SwarmService)
	s.verifyNotifyServiceRemove(true, fmt.Sprintf("distribute=true&serviceName=%s", "my-removed-service-1"))
}

func (s *NotificationTestSuite) Test_ServicesRemove_ReturnsError_WhenUrlCannotBeParsed() {
	CachedServices = make(map[string]SwarmService)
	n := newNotification([]string{}, []string{"%%%"})
	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 1, 0)

	s.Error(err)
}

func (s *NotificationTestSuite) Test_ServicesRemove_ReturnsError_WhenHttpStatusIsNot200() {
	CachedServices = make(map[string]SwarmService)
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	n := newNotification([]string{}, []string{httpSrv.URL})
	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 1, 0)

	s.Error(err)
}

func (s *NotificationTestSuite) Test_ServicesRemove_ReturnsError_WhenHttpRequestReturnsError() {
	CachedServices = make(map[string]SwarmService)
	n := newNotification([]string{}, []string{"this-does-not-exist"})

	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 1, 0)

	s.Error(err)
}

func (s *NotificationTestSuite) Test_ServicesRemove_RetriesRequests() {
	attempt := 0
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempt < 2 {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
		}
		attempt = attempt + 1
	}))

	n := newNotification([]string{}, []string{httpSrv.URL})
	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 3, 0)

	s.NoError(err)
}

// Util

func (s *NotificationTestSuite) getSwarmServices(labels map[string]string) *[]SwarmService {
	ann := swarm.Annotations{
		Name:   "my-service",
		Labels: labels,
	}
	spec := swarm.ServiceSpec{
		Annotations: ann,
	}
	srv := swarm.Service{
		Spec: spec,
	}
	CachedServices = map[string]SwarmService{}
	CachedServices[ann.Name] = SwarmService{srv}
	ss := SwarmService{srv}
	return &[]SwarmService{ss}
}

func (s *NotificationTestSuite) verifyNotifyServiceCreate(labels map[string]string, expectSent bool, expectQuery string) {
	actualSent := false
	actualQuery := ""
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualPath := r.URL.Path
		if r.Method == "GET" {
			switch actualPath {
			case "/v1/docker-flow-proxy/reconfigure":
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				actualQuery = r.URL.RawQuery
				actualSent = true
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer func() { httpSrv.Close() }()
	url := fmt.Sprintf("%s/v1/docker-flow-proxy/reconfigure", httpSrv.URL)

	n := newNotification([]string{url}, []string{})
	n.ServicesCreate(s.getSwarmServices(labels), 1, 0)

	passed := false
	for i := 0; i < 100; i++ {
		if actualSent {
			s.Equal(expectQuery, actualQuery)
			passed = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.True(passed)
}

func (s *NotificationTestSuite) verifyNotifyServiceRemove(expectSent bool, expectQuery string) {
	actualSent := false
	actualQuery := ""
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualPath := r.URL.Path
		if r.Method == "GET" {
			switch actualPath {
			case "/v1/docker-flow-proxy/remove":
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				actualQuery = r.URL.RawQuery
				actualSent = true
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer func() { httpSrv.Close() }()
	url := fmt.Sprintf("%s/v1/docker-flow-proxy/remove", httpSrv.URL)
	n := newNotification([]string{}, []string{url})

	CachedServices["my-removed-service-1"] = SwarmService{}
	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 1, 0)

	s.NoError(err)
	s.Equal(expectSent, actualSent)
	if expectSent {
		s.Equal(expectQuery, actualQuery)
		s.NotContains(CachedServices, "my-removed-service-1")
	}
}
