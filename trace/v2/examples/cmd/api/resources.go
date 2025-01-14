package main

import (
	"net/http"

	"github.com/arquivei/foundationkit/app"
	"github.com/arquivei/foundationkit/gokitmiddlewares/loggingmiddleware"
	"github.com/arquivei/foundationkit/httpmiddlewares/enrichloggingmiddleware"
	"github.com/arquivei/foundationkit/trace/v2"
	"github.com/arquivei/foundationkit/trace/v2/examples/services/ping"
	"github.com/arquivei/foundationkit/trace/v2/examples/services/ping/apiping"
	"github.com/arquivei/foundationkit/trace/v2/examples/services/ping/implping"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
)

func setupTrace() {
	traceShutdown := trace.Setup(config.Trace)
	app.RegisterShutdownHandler(
		&app.ShutdownHandler{
			Name:     "opentelemetry_trace",
			Priority: app.ShutdownPriority(shutdownPriorityTrace),
			Handler:  traceShutdown,
			Policy:   app.ErrorPolicyAbort,
		})
}

func getEndpoint() endpoint.Endpoint {
	loggingConfig := loggingmiddleware.NewDefaultConfig("ping")

	pongAdapter := implping.NewHTTPPongAdapter(
		&http.Client{Timeout: config.Pong.HTTP.Timeout},
		config.Pong.HTTP.URL,
	)

	pingEndpoint := endpoint.Chain(
		loggingmiddleware.MustNew(loggingConfig),
	)(apiping.MakeAPIPingEndpoint(
		ping.NewService(pongAdapter),
	))

	return pingEndpoint
}

func getHTTPServer() *http.Server {
	r := mux.NewRouter()

	r.PathPrefix("/ping/").Handler(apiping.MakeHTTPHandler(getEndpoint()))

	r.Use(enrichloggingmiddleware.New)

	httpAddr := ":" + config.HTTP.Port
	httpServer := &http.Server{Addr: httpAddr, Handler: r}

	app.RegisterShutdownHandler(
		&app.ShutdownHandler{
			Name:     "http_server",
			Priority: shutdownPriorityHTTP,
			Handler:  httpServer.Shutdown,
			Policy:   app.ErrorPolicyAbort,
		})

	return httpServer
}
