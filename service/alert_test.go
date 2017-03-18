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
	labels := make(map[string]string)
	labels["com.df.alert"] = "true"
	labels["com.df.alert.name"] = "Mem"
	spec1 := swarm.ServiceSpec{}
	spec1.Name = "my_service-1"
	spec1.Labels = labels
	service1 := swarm.Service{
		Spec: spec1,
	}
	services := []swarm.Service{service1}
	actualMethod := ""
	actualBody := []AlertBody{}
	expectedBody := AlertBody{
		Name: "myservice1Mem",
	}
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		actualMethod = r.Method
		body, _ := ioutil.ReadAll(r.Body)
		alertBody := AlertBody{}
		json.Unmarshal(body, &alertBody)
		actualBody = append(actualBody, alertBody)

	}))
	defer func() { httpSrv.Close() }()
	addr := fmt.Sprintf("%s/v1/docker-flow-proxy/reconfigure", httpSrv.URL)

	l := NewAlert([]string{addr}, []string{})
	l.ServicesCreate(&services, 0, 0)

	s.Equal("PUT", actualMethod)
	s.Contains(actualBody, expectedBody)
}

//func (s *AlertTestSuite) Test_ServicesCreate_ReturnsError_WhenUrlCannotBeParsed() {
//	labels := make(map[string]string)
//	labels["com.df.notify"] = "true"
//	n := NewAlert([]string{"%%%"}, []string{})
//	err := n.ServicesCreate(s.getSwarmServices(labels), 1, 0)
//
//	s.Error(err)
//}
//
//func (s *AlertTestSuite) Test_ServicesCreate_DoesNotSendRequest_WhenDfNotifyIsNotDefined() {
//	labels := make(map[string]string)
//	labels["DF_key1"] = "value1"
//
//	s.verifyNotifyServiceCreate(labels, false, "")
//}
//
//func (s *AlertTestSuite) Test_ServicesCreate_ReturnsError_WhenHttpStatusIsNot200() {
//	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		w.WriteHeader(http.StatusNotFound)
//	}))
//	labels := make(map[string]string)
//	labels["com.df.notify"] = "true"
//
//	n := NewAlert([]string{httpSrv.URL}, []string{})
//	err := n.ServicesCreate(s.getSwarmServices(labels), 1, 0)
//
//	s.Error(err)
//}
//
//func (s *AlertTestSuite) Test_ServicesCreate_ReturnsError_WhenHttpRequestReturnsError() {
//	labels := make(map[string]string)
//	labels["com.df.notify"] = "true"
//
//	n := NewAlert([]string{"this-does-not-exist"}, []string{})
//	err := n.ServicesCreate(s.getSwarmServices(labels), 1, 0)
//
//	s.Error(err)
//}
//
//func (s *AlertTestSuite) Test_ServicesCreate_RetriesRequests() {
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
//	n := NewAlert([]string{httpSrv.URL}, []string{})
//	err := n.ServicesCreate(s.getSwarmServices(labels), 3, 0)
//
//	s.NoError(err)
//}
//
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
//// Util
//
//func (s *AlertTestSuite) getSwarmServices(labels map[string]string) *[]swarm.Service {
//	ann := swarm.Annotations{
//		Name:   "my-service",
//		Labels: labels,
//	}
//	spec := swarm.ServiceSpec{
//		Annotations: ann,
//	}
//	serv := swarm.Service{
//		Spec: spec,
//	}
//	return &[]swarm.Service{serv}
//}

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
