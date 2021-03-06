package api

import (
	"encoding/json"

	"github.com/baidu/openedge/config"
	"github.com/baidu/openedge/logger"
	"github.com/baidu/openedge/trans/http"
	"github.com/baidu/openedge/utils"
	"github.com/juju/errors"
	"github.com/sirupsen/logrus"
)

// Engine engine
type Engine interface {
	Start(module config.Module) error
	Stop(moduleName string) error
	Authenticate(username, password string) bool
}

// Server api server to start/stop modules
type Server struct {
	*http.Server
	engine Engine
	log    *logrus.Entry
}

// NewServer creates a new server
func NewServer(e Engine, c http.ServerConfig) (*Server, error) {
	svr, err := http.NewServer(c)
	if err != nil {
		return nil, errors.Trace(err)
	}
	s := &Server{
		Server: svr,
		engine: e,
		log:    logger.WithFields("api", "http"),
	}
	s.Handle(s.getPort, "GET", "/ports/available", "host", "{host}")
	s.Handle(s.startModule, "PUT", "/modules/{name}/start")
	s.Handle(s.stopModule, "PUT", "/modules/{name}/stop")
	return s, nil
}

func (s *Server) startModule(params http.Params, headers http.Headers, reqBody []byte) ([]byte, error) {
	if !s.engine.Authenticate(headers.Get("x-iot-edge-username"), headers.Get("x-iot-edge-password")) {
		account := headers.Get("x-iot-edge-username")
		s.log.Errorf("Unauthorized to start module (%s) by account (%s)", params["name"], account)
		return nil, errors.Errorf("Account (%s) unauthorized", account)
	}
	if reqBody == nil {
		return nil, errors.Errorf("Request body missing")
	}
	var m config.Module
	err := utils.UnmarshalJSON(reqBody, &m)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if err = s.engine.Start(m); err != nil {
		s.log.WithError(err).Errorf("Failed to start module (%s)", m.Name)
		return nil, errors.Trace(err)
	}
	return nil, nil
}

func (s *Server) stopModule(params http.Params, headers http.Headers, reqBody []byte) ([]byte, error) {
	if !s.engine.Authenticate(headers.Get("x-iot-edge-username"), headers.Get("x-iot-edge-password")) {
		account := headers.Get("x-iot-edge-username")
		s.log.Errorf("Unauthorized to stop module (%s) by account (%s)", params["name"], account)
		return nil, errors.Errorf("Account (%s) unauthorized", account)
	}
	if err := s.engine.Stop(params["name"]); err != nil {
		s.log.WithError(err).Errorf("Failed to stop module (%s)", params["name"])
		return nil, errors.Trace(err)
	}
	return nil, nil
}

func (s *Server) getPort(params http.Params, headers http.Headers, reqBody []byte) ([]byte, error) {
	if !s.engine.Authenticate(headers.Get("x-iot-edge-username"), headers.Get("x-iot-edge-password")) {
		account := headers.Get("x-iot-edge-username")
		s.log.Errorf("Unauthorized to get port by account (%s)", account)
		return nil, errors.Errorf("Account (%s) unauthorized", account)
	}
	host, ok := params["host"]
	if !ok {
		host = "127.0.0.1"
	}
	port, err := utils.GetPortAvailable(host)
	if err != nil {
		return nil, errors.Trace(err)
	}
	data := map[string]int{"port": port}
	resBody, err := json.Marshal(&data)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return resBody, nil
}
