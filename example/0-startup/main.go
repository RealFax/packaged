package main

import (
	"context"
	"github.com/RealFax/packaged"
	"log"
	"net/http"
)

type httpServer struct {
	*packaged.UnimplementedHandler
	ctx packaged.Namespace
	svc *http.Server
}

func (s httpServer) Name() string               { return "http-server" }
func (s httpServer) Type() packaged.ServiceType { return packaged.ServiceTypeAsync }
func (s httpServer) OnInstall() error {
	addr, _ := s.ctx.GetEnv("HTTP_ADDR")
	s.svc = &http.Server{Addr: addr}
	return nil
}
func (s httpServer) OnStart() error {
	return s.svc.ListenAndServe()
}
func (s httpServer) OnStop() error {
	return s.svc.Shutdown(context.Background())
}

func main() {
	packaged.Register(func(ns packaged.Namespace) packaged.Service {
		return &httpServer{ctx: ns}
	})

	if err := packaged.Run(); err != nil {
		log.Fatal(err)
	}

	packaged.Wait()
}
