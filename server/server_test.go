package server_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/albertodonato/h2static/server"
	"github.com/albertodonato/h2static/testhelpers"
)

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

type ServerTestSuite struct {
	testhelpers.TempDirTestSuite
}

// If the specified Dir doesn't exist, an error is returned.
func (s *ServerTestSuite) TestValidateConfigDirNotExists() {
	serv, err := server.NewStaticServer(server.StaticServerConfig{Dir: "/not/here"})
	s.Nil(serv)
	s.NotNil(err)
	s.Contains(err.Error(), "/not/here: no such file or directory")
}

// If the specified Dir exists but is not a directory, an error is returned.
func (s *ServerTestSuite) TestValidateConfigDirNotDir() {
	path := s.WriteFile("foo", "bar")
	serv, err := server.NewStaticServer(server.StaticServerConfig{Dir: path})
	s.Nil(serv)
	s.NotNil(err)
	s.Equal(fmt.Sprintf("not a directory: %s", path), err.Error())
}

// If the TLS certificate file doesn't exist, an error is returned.
func (s *ServerTestSuite) TestValidateConfigTLSCertFileNotExists() {
	tlsKey := s.WriteFile("foo", "bar")
	serv, err := server.NewStaticServer(server.StaticServerConfig{
		Dir:     s.TempDir,
		TLSCert: "/not/here",
		TLSKey:  tlsKey,
	})
	s.Nil(serv)
	s.NotNil(err)
	s.Contains(err.Error(), "/not/here: no such file or directory")
}

// If the TLS key file doesn't exist, an error is returned.
func (s *ServerTestSuite) TestValidateConfigTLSKeyFileNotExists() {
	tlsCert := s.WriteFile("foo", "bar")
	serv, err := server.NewStaticServer(
		server.StaticServerConfig{
			Dir:     s.TempDir,
			TLSCert: tlsCert,
			TLSKey:  "/not/here",
		})
	s.Nil(serv)
	s.NotNil(err)
	s.Contains(err.Error(), "/not/here: no such file or directory")
}

// If the passwords file doesn't exist, an error is returned.
func (s *ServerTestSuite) TestValidateConfigPasswordFileNotExists() {
	serv, err := server.NewStaticServer(
		server.StaticServerConfig{
			Dir:          s.TempDir,
			PasswordFile: "/not/here",
		})
	s.Nil(serv)
	s.NotNil(err)
	s.Contains(err.Error(), "/not/here: no such file or directory")
}

// If no invalid file is passed, ValidateConfig returns nil.
func (s *ServerTestSuite) TestValidateConfigNoError() {
	serv, err := server.NewStaticServer(server.StaticServerConfig{Dir: s.TempDir})
	s.NotNil(serv)
	s.Nil(err)
}

// IsHTTPS returns true if certificates are set.
func (s *ServerTestSuite) TestEnableTLSTrue() {
	tlsCert := s.WriteFile("cert.pem", "cert")
	tlsKey := s.WriteFile("key.pem", "key")
	serv, err := server.NewStaticServer(server.StaticServerConfig{
		Dir:     s.TempDir,
		TLSCert: tlsCert,
		TLSKey:  tlsKey,
	})
	s.Nil(err)
	s.True(serv.IsHTTPS())
}

// IsHTTPS returns false if certificates are not set.
func (s *ServerTestSuite) TestEnableTLSFalse() {
	serv := server.StaticServer{}
	s.False(serv.IsHTTPS())
}

// getServer returns static file handlers for a path.
func (s *ServerTestSuite) TestGetServerDefaultStaticServe() {
	serv, err := server.NewStaticServer(server.StaticServerConfig{})
	s.Nil(err)
	httpServer, err := server.GetServer(serv)
	s.Nil(err)
	s.Nil(httpServer.TLSNextProto)
	s.Equal("", httpServer.Addr)
	h := httpServer.Handler.(*server.CommonHeadersHandler)
	mux := h.Handler.(*http.ServeMux)
	fh, pattern := mux.Handler(httptest.NewRequest("GET", "/foo", nil))
	s.Equal("/", pattern)
	s.IsType(&server.FileHandler{}, fh)
	curdir, err := os.Getwd()
	s.Nil(err)
	s.Equal(curdir, fh.(*server.FileHandler).FileSystem.Root)
}

// getServer returns a configured http.Server with the specified dir
func (s *ServerTestSuite) TestSetupServerSpecifyDir() {
	serv, err := server.NewStaticServer(server.StaticServerConfig{Dir: s.TempDir})
	s.Nil(err)
	httpServer, err := server.GetServer(serv)
	s.Nil(err)
	h := httpServer.Handler.(*server.CommonHeadersHandler)
	mux := h.Handler.(*http.ServeMux)
	fh, _ := mux.Handler(httptest.NewRequest("GET", "/foo", nil))
	s.Equal(s.TempDir, fh.(*server.FileHandler).FileSystem.Root)
}

// getServer returns a configured http.Server which allows outside symlink access
func (s *ServerTestSuite) TestSetupServerSpecifyAllowOutsideSymlinks() {
	serv, err := server.NewStaticServer(server.StaticServerConfig{
		Dir:                  s.TempDir,
		AllowOutsideSymlinks: true,
	})
	s.Nil(err)
	httpServer, err := server.GetServer(serv)
	s.Nil(err)
	h := httpServer.Handler.(*server.CommonHeadersHandler)
	mux := h.Handler.(*http.ServeMux)
	fh, _ := mux.Handler(httptest.NewRequest("GET", "/foo", nil))
	s.True(fh.(*server.FileHandler).FileSystem.AllowOutsideSymlinks)
}

// getServer returns a configured http.Server with logging
func (s *ServerTestSuite) TestSetupServerLog() {
	serv, err := server.NewStaticServer(server.StaticServerConfig{
		Dir: s.TempDir,
		Log: true,
	})
	s.Nil(err)
	httpServer, err := server.GetServer(serv)
	s.Nil(err)
	h := httpServer.Handler.(*server.CommonHeadersHandler)
	s.IsType(&server.LoggingHandler{}, h.Handler)
}

// getServer returns a configured http.Server without HTTP/2
func (s *ServerTestSuite) TestSetupServerNoH2() {
	serv, err := server.NewStaticServer(server.StaticServerConfig{
		Dir:       s.TempDir,
		DisableH2: true,
	})
	s.Nil(err)
	httpServer, err := server.GetServer(serv)
	s.Nil(err)
	s.NotNil(httpServer.TLSNextProto)
}

// getServer returns a configured http.Server with Basic-Auth
func (s *ServerTestSuite) TestSetupServerBasicAuth() {
	absPath := s.WriteFile("basic-auth", "")
	serv, err := server.NewStaticServer(server.StaticServerConfig{
		Dir:          s.TempDir,
		PasswordFile: absPath,
	})
	s.Nil(err)
	httpServer, err := server.GetServer(serv)
	s.Nil(err)
	h := httpServer.Handler.(*server.CommonHeadersHandler)
	s.IsType(&server.BasicAuthHandler{}, h.Handler)
}
