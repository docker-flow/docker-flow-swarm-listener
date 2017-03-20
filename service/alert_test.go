package service

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"os"
	"strings"
	"net/http"
	"net/http/httptest"
	"fmt"
	"github.com/docker/docker/api/types/swarm"
	"encoding/json"
	"io/ioutil"
)

type AlertTestSuite struct {
	suite.Suite
}

func TestAlertUnitTestSuite(t *testing.T) {
	s := new(AlertTestSuite)
	logPrintfOrig := logPrintf
	defer func() { logPrintf = logPrintfOrig }()
	logPrintf = func(format string, v ...interface{}) {}
	suite.Run(t, s)
}

// NewAlert

func (s *AlertTestSuite) Test_NewAlert_SetsCreateUrl() {
	expected := []string{"this-is-a-url", "this-is-a-different-url"}

	actual := NewAlert(expected, []string{})

	s.Equal(expected, actual.CreateServiceAddr)
}

func (s *AlertTestSuite) Test_NewAlert_SetsRemoveUrl() {
	expected := []string{"this-is-a-url", "this-is-a-different-url"}

	actual := NewAlert([]string{}, expected)

	s.Equal(expected, actual.RemoveServiceAddr)
}

// NewAlertFromEnv

func (s *AlertTestSuite) Test_NewAlertFromEnv_SetsNotifyCreateUrlFromEnvVars() {
	tests := []struct {
		envKey string
	}{
		{"DF_ALERT_CREATE_SERVICE_URL"},
		{"DF_ALERT_URL"},
	}
	for _, t := range tests {
		host := os.Getenv(t.envKey)
		expected := []string{"this-is-a-url", "this-is-a-different-url"}
		os.Setenv(t.envKey, strings.Join(expected, ","))

		actual := NewAlertFromEnv()

		s.Equal(expected, actual.CreateServiceAddr)
		os.Setenv(t.envKey, host)
	}
}

func (s *AlertTestSuite) Test_NewAlertFromEnv_SetsNotifyRemoveUrlFromEnvVars() {
	tests := []struct {
		envKey string
	}{
		{"DF_ALERT_REMOVE_SERVICE_URL"},
		{"DF_ALERT_URL"},
	}
	for _, t := range tests {
		host := os.Getenv(t.envKey)
		expected := []string{"this-is-a-url", "this-is-a-different-url"}
		os.Setenv(t.envKey, strings.Join(expected, ","))

		n := NewAlertFromEnv()

		s.Equal(expected, n.RemoveServiceAddr, "Failed to fetch information from the env. var. %s.", t.envKey)
		os.Setenv(t.envKey, host)
	}
}

// AlertCreate

func (s *AlertTestSuite) Test_ServicesCreate_SendsRequests() {

	actualMethod := ""
	actualBody := AlertBody{}
	expectedBody := AlertBody{
		Name: "myservice1Mem",
		If:   "An if statement",
		For:  "For statement",
	}
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		actualMethod = r.Method
		body, _ := ioutil.ReadAll(r.Body)
		alertBody := AlertBody{}
		json.Unmarshal(body, &alertBody)
		actualBody = alertBody

	}))
	defer httpSrv.Close()
	addr := fmt.Sprintf("%s/v1/docker-flow-proxy/reconfigure", httpSrv.URL)

	actual := NewAlert([]string{addr}, []string{})
	actual.ServicesCreate(&[]swarm.Service{s.getTestService()}, 0, 0)

	s.Equal("PUT", actualMethod)
	s.Equal(actualBody, expectedBody)
}

func (s *AlertTestSuite) Test_ServicesCreate_ReturnsError_WhenUrlCannotBeParsed() {
	actual := NewAlert([]string{"%%%"}, []string{})
	err := actual.ServicesCreate(&[]swarm.Service{s.getTestService()}, 1, 0)

	s.Error(err)
}

func (s *AlertTestSuite) Test_ServicesCreate_DoesNotSendRequest_WhenAlertNameIsNotDefined() {
	spec1 := swarm.ServiceSpec{}
	spec1.Name = "my_service-1"
	spec1.Labels = map[string]string {"com.df.alert.if": "An If statement"}
	service1 := swarm.Service{
		Spec: spec1,
	}
	services := []swarm.Service{service1}
	called := false
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer httpSrv.Close()
	addr := fmt.Sprintf("%s/v1/docker-flow-proxy/reconfigure", httpSrv.URL)

	actual := NewAlert([]string{addr}, []string{})
	actual.ServicesCreate(&services, 0, 0)

	s.False(called)
}

func (s *AlertTestSuite) Test_ServicesCreate_DoesNotSendRequest_WhenAlertIfIsNotDefined() {
	spec1 := swarm.ServiceSpec{}
	spec1.Name = "my_service-1"
	spec1.Labels = map[string]string {"com.df.alert.name": "my-alert"}
	service1 := swarm.Service{
		Spec: spec1,
	}
	services := []swarm.Service{service1}
	called := false
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer httpSrv.Close()
	addr := fmt.Sprintf("%s/v1/docker-flow-proxy/reconfigure", httpSrv.URL)

	actual := NewAlert([]string{addr}, []string{})
	actual.ServicesCreate(&services, 0, 0)

	s.False(called)
}

func (s *AlertTestSuite) Test_ServicesCreate_ReturnsError_WhenHttpStatusIsNot200() {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	actual := NewAlert([]string{httpSrv.URL}, []string{})
	err := actual.ServicesCreate(&[]swarm.Service{s.getTestService()}, 1, 0)

	s.Error(err)
}

func (s *AlertTestSuite) Test_ServicesCreate_ReturnsError_WhenHttpRequestReturnsError() {
	actual := NewAlert([]string{"this-does-not-exist"}, []string{})
	err := actual.ServicesCreate(&[]swarm.Service{s.getTestService()}, 1, 0)

	s.Error(err)
}

func (s *AlertTestSuite) Test_ServicesCreate_RetriesRequests() {
	attempt := 0
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempt < 1 {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
		}
		attempt += 1
	}))
	defer httpSrv.Close()

	actual := NewAlert([]string{httpSrv.URL}, []string{})
	err := actual.ServicesCreate(&[]swarm.Service{s.getTestService()}, 3, 1)

	s.NoError(err)
}

func (s *AlertTestSuite) Test_ServicesCreate_SendsMultipleAlerts() {
	spec := swarm.ServiceSpec{}
	spec.Labels = make(map[string]string)
	spec.Name = "my-service_1"
	service := swarm.Service{
		Spec: spec,
	}
	expectedBodies := []AlertBody{}
	actualBodies := []AlertBody{}
	for i := 1; i <= 2; i++ {
		service.Spec.Labels[fmt.Sprintf("com.df.alert.%d.name", i)] = fmt.Sprintf("name-%d", i)
		service.Spec.Labels[fmt.Sprintf("com.df.alert.%d.if", i)] = fmt.Sprintf("if-%d", i)
		service.Spec.Labels[fmt.Sprintf("com.df.alert.%d.for", i)] = fmt.Sprintf("for-%d", i)
		expectedBodies = append(expectedBodies, AlertBody{
			Name: "myservice1" + service.Spec.Labels[fmt.Sprintf("com.df.alert.%d.name", i)],
			If:   service.Spec.Labels[fmt.Sprintf("com.df.alert.%d.if", i)],
			For:  service.Spec.Labels[fmt.Sprintf("com.df.alert.%d.for", i)],
		})
	}
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		alertBody := AlertBody{}
		json.Unmarshal(body, &alertBody)
		actualBodies = append(actualBodies, alertBody)

	}))
	defer httpSrv.Close()
	addr := fmt.Sprintf("%s/v1/docker-flow-proxy/reconfigure", httpSrv.URL)

	actual := NewAlert([]string{addr}, []string{})
	actual.ServicesCreate(&[]swarm.Service{service}, 0, 0)

	s.Len(actualBodies, 2)
	s.Equal(expectedBodies, actualBodies)
}

//// ServicesRemove
//
//func (s *AlertTestSuite) Test_ServicesRemove_SendsRequests() {
//	Services = make(map[string]swarm.Service)
//	s.verifyNotifyServiceRemove(true, fmt.Sprintf("distribute=true&serviceName=%s", "my-removed-service-1"))
//}
//
//func (s *AlertTestSuite) Test_ServicesRemove_ReturnsError_WhenUrlCannotBeParsed() {
//	Services = make(map[string]swarm.Service)
//	n := NewAlert([]string{}, []string{"%%%"})
//	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 1, 0)
//
//	s.Error(err)
//}
//
//func (s *AlertTestSuite) Test_ServicesRemove_ReturnsError_WhenHttpStatusIsNot200() {
//	Services = make(map[string]swarm.Service)
//	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		w.WriteHeader(http.StatusNotFound)
//	}))
//
//	n := NewAlert([]string{}, []string{httpSrv.URL})
//	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 1, 0)
//
//	s.Error(err)
//}
//
//func (s *AlertTestSuite) Test_ServicesRemove_ReturnsError_WhenHttpRequestReturnsError() {
//	Services = make(map[string]swarm.Service)
//	n := NewAlert([]string{}, []string{"this-does-not-exist"})
//	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 1, 0)
//
//	s.Error(err)
//}
//
//func (s *AlertTestSuite) Test_ServicesRemove_RetriesRequests() {
//	attempt := 0
//	labels := make(map[string]string)
//	labels["com.df.notify"] = "true"
//	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		if attempt < 2 {
//			w.WriteHeader(http.StatusNotFound)
//		} else {
//			w.WriteHeader(http.StatusOK)
//			w.Header().Set("Content-Type", "application/json")
//		}
//		attempt = attempt + 1
//	}))
//
//	n := NewAlert([]string{}, []string{httpSrv.URL})
//	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 3, 0)
//
//	s.NoError(err)
//}
//
// Util

func (s *AlertTestSuite) getTestService() swarm.Service {
	labels := make(map[string]string)
	labels["com.df.alert.name"] = "Mem"
	labels["com.df.alert.if"] = "An if statement"
	labels["com.df.alert.for"] = "For statement"
	spec := swarm.ServiceSpec{}
	spec.Name = "my_service-1"
	spec.Labels = labels
	return swarm.Service{
		Spec: spec,
	}
}

//func (s *AlertTestSuite) verifyNotifyServiceRemove(expectSent bool, expectQuery string) {
//	actualSent := false
//	actualQuery := ""
//	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		actualPath := r.URL.Path
//		if r.Method == "GET" {
//			switch actualPath {
//			case "/v1/docker-flow-proxy/remove":
//				w.WriteHeader(http.StatusOK)
//				w.Header().Set("Content-Type", "application/json")
//				actualQuery = r.URL.RawQuery
//				actualSent = true
//			default:
//				w.WriteHeader(http.StatusNotFound)
//			}
//		}
//	}))
//	defer func() { httpSrv.Close() }()
//	url := fmt.Sprintf("%s/v1/docker-flow-proxy/remove", httpSrv.URL)
//	n := NewAlert([]string{}, []string{url})
//
//	Services["my-removed-service-1"] = swarm.Service{}
//	err := n.ServicesRemove(&[]string{"my-removed-service-1"}, 1, 0)
//
//	s.NoError(err)
//	s.Equal(expectSent, actualSent)
//	if expectSent {
//		s.Equal(expectQuery, actualQuery)
//		s.NotContains(Services, "my-removed-service-1")
//	}
//}
