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

	actual := NewNotification(expected, []string{})

	s.Equal(expected, actual.CreateServiceAddr)
}

func (s *NotificationTestSuite) Test_NewNotification_SetsRemoveUrl() {
	expected := []string{"this-is-a-url", "this-is-a-different-url"}

	actual := NewNotification([]string{}, expected)

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

	s.verifyNotifyServiceCreate(labels, true, fmt.Sprintf("distribute=true&serviceName=%s", "my-service"))
}

func (s *NotificationTestSuite) Test_ServicesCreate_UsesLabelFromEnvVars() {
	notifyLabelOrig := os.Getenv("DF_NOTIFY_LABEL")
	defer func() { os.Setenv("DF_NOTIFY_LABEL", notifyLabelOrig) }()
	os.Setenv("DF_NOTIFY_LABEL", "com.df.something")

	labels := make(map[string]string)
	labels["com.df.something"] = "true"
	labels["com.df.distribute"] = "true"
	labels["label.without.correct.prefix"] = "something"

	s.verifyNotifyServiceCreate(labels, true, fmt.Sprintf("distribute=true&serviceName=%s", "my-service"))
}

func (s *NotificationTestSuite) Test_ServicesCreate_ReturnsError_WhenUrlCannotBeParsed() {
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"
	n := NewNotification([]string{"%%%"}, []string{})
	err := n.ServicesCreate(s.getSwarmServices(labels), 1, 0)

	s.Error(err)
}

func (s *NotificationTestSuite) Test_ServicesCreate_DoesNotSendRequest_WhenDfNotifyIsNotDefined() {
	labels := make(map[string]string)
	labels["DF_key1"] = "value1"

	s.verifyNotifyServiceCreate(labels, false, "")
}

func (s *NotificationTestSuite) Test_ServicesCreate_ReturnsError_WhenHttpStatusIsNot200() {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"

	n := NewNotification([]string{httpSrv.URL}, []string{})
	err := n.ServicesCreate(s.getSwarmServices(labels), 1, 0)

	s.Error(err)
}

func (s *NotificationTestSuite) Test_ServicesCreate_ReturnsError_WhenHttpRequestReturnsError() {
	labels := make(map[string]string)
	labels["com.df.notify"] = "true"

	n := NewNotification([]string{"this-does-not-exist"}, []string{})
	err := n.ServicesCreate(s.getSwarmServices(labels), 1, 0)

	s.Error(err)
}

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

	n := NewNotification([]string{httpSrv.URL}, []string{})
	err := n.ServicesCreate(s.getSwarmServices(labels), 2, 1)

	s.NoError(err)
}

// ServicesRemove

func (s *NotificationTestSuite) Test_ServicesRemove_SendsRequests() {
	Services = make(map[string]swarm.Service)
	s.verifyNotifyServiceRemove(true, fmt.Sprintf("distribute=true&serviceName=%s", "my-removed-service-1"))
}

func (s *NotificationTestSuite) Test_ServicesRemove_ReturnsError_WhenUrlCannotBeParsed() {
	Services = make(map[string]swarm.Service)
	n := NewNotification([]string{}, []string{"%%%"})
	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 1, 0)

	s.Error(err)
}

func (s *NotificationTestSuite) Test_ServicesRemove_ReturnsError_WhenHttpStatusIsNot200() {
	Services = make(map[string]swarm.Service)
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	n := NewNotification([]string{}, []string{httpSrv.URL})
	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 1, 0)

	s.Error(err)
}

func (s *NotificationTestSuite) Test_ServicesRemove_ReturnsError_WhenHttpRequestReturnsError() {
	Services = make(map[string]swarm.Service)
	n := NewNotification([]string{}, []string{"this-does-not-exist"})
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

	n := NewNotification([]string{}, []string{httpSrv.URL})
	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 3, 0)

	s.NoError(err)
}

// Util

func (s *NotificationTestSuite) getSwarmServices(labels map[string]string) *[]swarm.Service {
	ann := swarm.Annotations{
		Name:   "my-service",
		Labels: labels,
	}
	spec := swarm.ServiceSpec{
		Annotations: ann,
	}
	serv := swarm.Service{
		Spec: spec,
	}
	return &[]swarm.Service{serv}
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

	n := NewNotification([]string{url}, []string{})
	err := n.ServicesCreate(s.getSwarmServices(labels), 1, 0)

	s.NoError(err)
	s.Equal(expectSent, actualSent)
	if expectSent {
		s.Equal(expectQuery, actualQuery)
	}
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
	n := NewNotification([]string{}, []string{url})

	Services["my-removed-service-1"] = swarm.Service{}
	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 1, 0)

	s.NoError(err)
	s.Equal(expectSent, actualSent)
	if expectSent {
		s.Equal(expectQuery, actualQuery)
		s.NotContains(Services, "my-removed-service-1")
	}
}
