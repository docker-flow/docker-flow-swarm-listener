package service

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type NotifierTestSuite struct {
	suite.Suite
	Logger   *log.Logger
	LogBytes *bytes.Buffer
	Params   string
}

func TestNotifierUnitTestSuite(t *testing.T) {
	suite.Run(t, new(NotifierTestSuite))
}

func (s *NotifierTestSuite) SetupSuite() {
	s.LogBytes = new(bytes.Buffer)
	s.Logger = log.New(s.LogBytes, "", 0)

	cParams := url.Values{}
	cParams.Add("serviceName", "hello")
	s.Params = cParams.Encode()

}

func (s *NotifierTestSuite) TearDownTest() {
	s.LogBytes.Reset()
}

// Create

func (s *NotifierTestSuite) Test_Create_SendsRequests() {

	var query1 string
	createMethod := http.MethodPost
	httpSrv := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter, r *http.Request) {
		if r.Method == createMethod {
			switch r.URL.Path {
			case "/v1/docker-flow-proxy/reconfigure":
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				query1 = r.URL.Query().Encode()
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer httpSrv.Close()

	url1 := fmt.Sprintf("%s/v1/docker-flow-proxy/reconfigure", httpSrv.URL)

	n := NewNotifier(
		url1, "", createMethod, http.MethodGet,
		"service", 5, 1, s.Logger)
	s.Equal(url1, n.GetCreateAddr())
	err := n.Create(context.Background(), s.Params)
	s.Require().NoError(err)

	s.Equal(s.Params, query1)

	urlObj1, err := url.Parse(url1)
	s.Require().NoError(err)

	urlObj1.RawQuery = s.Params

	logMsgs := s.LogBytes.String()
	s.Contains(logMsgs, fmt.Sprintf("Sending service created notification to %s", urlObj1.String()))
}

func (s *NotifierTestSuite) Test_Create_SendsRequestsWithParams() {

	var query1 string
	httpSrv := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			switch r.URL.Path {
			case "/v1/docker-flow-proxy/reconfigure":
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				query1 = r.URL.Query().Encode()
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer httpSrv.Close()

	url1 := fmt.Sprintf("%s/v1/docker-flow-proxy/reconfigure?hello=world", httpSrv.URL)

	n := NewNotifier(url1, "", http.MethodGet,
		http.MethodGet, "service", 5, 1, s.Logger)
	s.Equal(url1, n.GetCreateAddr())
	err := n.Create(context.Background(), s.Params)
	s.Require().NoError(err)

	newParams := "hello=world&serviceName=hello"

	s.Equal(newParams, query1)

	urlObj1, err := url.Parse(url1)
	s.Require().NoError(err)

	urlObj1.RawQuery = newParams

	logMsgs := s.LogBytes.String()
	s.Contains(logMsgs, fmt.Sprintf("Sending service created notification to %s", urlObj1.String()))
}

func (s *NotifierTestSuite) Test_Create_ReturnsAndLogsError_WhenUrlCannotBeParsed() {
	n := NewNotifier("%%%", "", http.MethodGet,
		http.MethodGet, "service", 5, 1, s.Logger)
	err := n.Create(context.Background(), s.Params)
	s.Error(err)

	logMsgs := s.LogBytes.String()
	s.True(strings.HasPrefix(logMsgs, "ERROR: "))
}

func (s *NotifierTestSuite) Test_Create_ReturnsAndLogsError_WhenHttpStatusIsNot200() {

	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	n := NewNotifier(
		httpSrv.URL, "", http.MethodGet,
		http.MethodGet, "node", 1, 0, s.Logger)
	err := n.Create(context.Background(), s.Params)
	s.Error(err)

	logMsgs := s.LogBytes.String()
	s.Contains(logMsgs, "ERROR: ")
}

func (s *NotifierTestSuite) Test_Create_ReturnsNoError_WhenHttpStatusIs409() {

	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))

	n := NewNotifier(
		httpSrv.URL, "", http.MethodGet,
		http.MethodGet, "node", 1, 0, s.Logger)
	err := n.Create(context.Background(), s.Params)
	s.Require().NoError(err)
}

func (s *NotifierTestSuite) Test_Create_ReturnsAndLogsError_WhenHttpRequestErrors() {
	n := NewNotifier(
		"this-does-not-exist", "", http.MethodGet,
		http.MethodGet, "node", 2, 1, s.Logger)

	err := n.Create(context.Background(), s.Params)
	s.Require().Error(err)

	logMsgs := s.LogBytes.String()
	s.Contains(logMsgs, "ERROR: ")
}

func (s *NotifierTestSuite) Test_Create_RetriesRequests() {
	attempt := 0
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempt < 1 {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
		}
		attempt++
	}))

	n := NewNotifier(
		httpSrv.URL, "", http.MethodGet,
		http.MethodGet, "service", 2, 1, s.Logger)
	n.Create(context.Background(), s.Params)

	s.Equal(2, attempt)

	logMsgs := s.LogBytes.String()
	expMsg := fmt.Sprintf("Retrying service created notification to %s", httpSrv.URL)
	s.Contains(logMsgs, expMsg)
}

func (s *NotifierTestSuite) Test_Create_Cancels() {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	n := NewNotifier(
		httpSrv.URL, "", http.MethodGet,
		http.MethodGet, "service", 2, 1, s.Logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n.Create(ctx, s.Params)

	logMsgs := s.LogBytes.String()
	expMsg := fmt.Sprintf("Canceling service create notification to %s", httpSrv.URL)
	s.Contains(logMsgs, expMsg)
}

// Remove

func (s *NotifierTestSuite) Test_Remove_SendsRequests() {
	var query1 string
	removeMethod := http.MethodDelete

	httpSrv := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter, r *http.Request) {
		if r.Method == removeMethod {
			switch r.URL.Path {
			case "/v1/docker-flow-proxy/remove":
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				query1 = r.URL.Query().Encode()
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer httpSrv.Close()

	url1 := fmt.Sprintf("%s/v1/docker-flow-proxy/remove", httpSrv.URL)

	n := NewNotifier("", url1, http.MethodGet,
		removeMethod, "node", 5, 1, s.Logger)
	s.Equal(url1, n.GetRemoveAddr())
	err := n.Remove(context.Background(), s.Params)
	s.Require().NoError(err)

	s.Equal(s.Params, query1)

	urlObj1, err := url.Parse(url1)
	s.Require().NoError(err)

	urlObj1.RawQuery = s.Params

	logMsgs := s.LogBytes.String()
	s.Contains(logMsgs, fmt.Sprintf("Sending node removed notification to %s", urlObj1.String()))
}

func (s *NotifierTestSuite) Test_Remove_ReturnsAndLogsError_WhenUrlCannotBeParsed() {
	n := NewNotifier("", "%%%", http.MethodGet,
		http.MethodGet, "node", 5, 1, s.Logger)
	err := n.Remove(context.Background(), s.Params)
	s.Error(err)

	logMsgs := s.LogBytes.String()
	s.Contains(logMsgs, "ERROR: ")
}

func (s *NotifierTestSuite) Test_Remove_ReturnsAndLogsError_WhenHttpStatusIsNot200() {

	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	n := NewNotifier(
		"", httpSrv.URL, http.MethodGet,
		http.MethodGet, "service", 1, 0, s.Logger)
	err := n.Remove(context.Background(), s.Params)
	s.Error(err)

	logMsgs := s.LogBytes.String()
	s.Contains(logMsgs, "ERROR: ")
}

func (s *NotifierTestSuite) Test_Remove_ReturnsAndLogsError_WhenHttpRequestReturnsError() {
	n := NewNotifier(
		"", "this-does-not-exist", http.MethodGet,
		http.MethodGet, "service", 2, 1, s.Logger)
	err := n.Remove(context.Background(), s.Params)
	s.Error(err)

	logMsgs := s.LogBytes.String()
	s.Contains(logMsgs, "ERROR: ")
}

func (s *NotifierTestSuite) Test_Remove_RetriesRequests() {
	attempt := 0
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempt < 1 {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
		}
		attempt++
	}))

	n := NewNotifier(
		"", httpSrv.URL, http.MethodGet,
		http.MethodGet, "node", 2, 1, s.Logger)
	err := n.Remove(context.Background(), s.Params)
	s.Require().NoError(err)

	s.Equal(2, attempt)

	logMsgs := s.LogBytes.String()
	expMsg := fmt.Sprintf("Retrying node removed notification to %s", httpSrv.URL)
	s.Contains(logMsgs, expMsg)
}

func (s *NotifierTestSuite) Test_Remove_Cancels() {

	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	n := NewNotifier(
		"", httpSrv.URL, http.MethodGet,
		http.MethodGet, "service", 2, 1, s.Logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n.Remove(ctx, s.Params)

	logMsgs := s.LogBytes.String()
	expMsg := fmt.Sprintf("Canceling service remove notification to %s", httpSrv.URL)
	s.Contains(logMsgs, expMsg)
}

func (s *NotifierTestSuite) EqualURLValues(expected, actual url.Values) {
	for k := range expected {
		expV, expA := expected[k], actual[k]
		s.Len(expV, 1)
		s.Len(expA, 1)
		s.Equal(expected.Get(k), actual.Get(k))
	}
}
