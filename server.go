package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/netutil"

	"github.com/imgproxy/imgproxy/v3/config"
	"github.com/imgproxy/imgproxy/v3/errorreport"
	"github.com/imgproxy/imgproxy/v3/ierrors"
	"github.com/imgproxy/imgproxy/v3/reuseport"
	"github.com/imgproxy/imgproxy/v3/router"
)

var (
	imgproxyIsRunningMsg = []byte("imgproxy is running")

	errInvalidSecret = ierrors.New(403, "Invalid secret", "Forbidden")
)

func buildRouter() *router.Router {
	r := router.New(config.PathPrefix)

	r.PanicHandler = handlePanic

	r.GET("/", handleLanding, true)
	r.GET("/health", handleHealth, true)
	r.GET("/favicon.ico", handleFavicon, true)
	r.GET("/", withCORS(withSecret(handleProcessing)), false)
	r.HEAD("/", withCORS(handleHead), false)
	r.OPTIONS("/", withCORS(handleHead), false)

	return r
}

func startServer(cancel context.CancelFunc) (*http.Server, error) {
	l, err := reuseport.Listen(config.Network, config.Bind)
	if err != nil {
		return nil, fmt.Errorf("Can't start server: %s", err)
	}
	l = netutil.LimitListener(l, config.MaxClients)

	s := &http.Server{
		Handler:        buildRouter(),
		ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if config.KeepAliveTimeout > 0 {
		s.IdleTimeout = time.Duration(config.KeepAliveTimeout) * time.Second
	} else {
		s.SetKeepAlivesEnabled(false)
	}

	go func() {
		log.Infof("Starting server at %s", config.Bind)
		if err := s.Serve(l); err != nil && err != http.ErrServerClosed {
			log.Error(err)
		}
		cancel()
	}()

	return s, nil
}

func shutdownServer(s *http.Server) {
	log.Info("Shutting down the server...")

	ctx, close := context.WithTimeout(context.Background(), 5*time.Second)
	defer close()

	s.Shutdown(ctx)
}

func withCORS(h router.RouteHandler) router.RouteHandler {
	return func(reqID string, rw http.ResponseWriter, r *http.Request) {
		if len(config.AllowOrigin) > 0 {
			rw.Header().Set("Access-Control-Allow-Origin", config.AllowOrigin)
			rw.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		}

		h(reqID, rw, r)
	}
}

func withSecret(h router.RouteHandler) router.RouteHandler {
	if len(config.Secret) == 0 {
		return h
	}

	authHeader := []byte(fmt.Sprintf("Bearer %s", config.Secret))

	return func(reqID string, rw http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), authHeader) == 1 {
			h(reqID, rw, r)
		} else {
			panic(errInvalidSecret)
		}
	}
}

func handlePanic(reqID string, rw http.ResponseWriter, r *http.Request, err error) {
	ierr := ierrors.Wrap(err, 3)

	if ierr.Unexpected {
		errorreport.Report(err, r)
	}

	router.LogResponse(reqID, r, ierr.StatusCode, ierr)

	rw.WriteHeader(ierr.StatusCode)

	if config.DevelopmentErrorsMode {
		rw.Write([]byte(ierr.Message))
	} else {
		rw.Write([]byte(ierr.PublicMessage))
	}
}

func handleHealth(reqID string, rw http.ResponseWriter, r *http.Request) {
	router.LogResponse(reqID, r, 200, nil)
	rw.WriteHeader(200)
	rw.Write(imgproxyIsRunningMsg)
}

func handleHead(reqID string, rw http.ResponseWriter, r *http.Request) {
	router.LogResponse(reqID, r, 200, nil)
	rw.WriteHeader(200)
}

func handleFavicon(reqID string, rw http.ResponseWriter, r *http.Request) {
	router.LogResponse(reqID, r, 200, nil)
	// TODO: Add a real favicon maybe?
	rw.WriteHeader(200)
}
