package server

import (
	"context"
	"errors"
	"github.com/RealFax/packaged"
	"log"
	"net/http"
	"time"
)

type service struct {
	packaged.Unimplemented
	g packaged.Group
	s *http.Server
}

func (s *service) Type() packaged.ServiceType { return packaged.ServiceTypeAsync }

func (s *service) Name() string {
	return "http-server"
}

func (s *service) OnInstall() error {
	addr, found := s.g.GetString("addr")
	if !found {
		return errors.New("addr not found")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("Hello, packaged."))
	})

	s.s = &http.Server{Addr: addr}
	return nil
}

func (s *service) OnStart() (err error) {
	log.Println("http server starting.", "listen_addr", s.s.Addr)
	if err = s.s.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
	}
	return
}

func (s *service) OnStop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.s.Shutdown(ctx)
}

func NewEntry(g packaged.Group) packaged.Service {
	return &service{g: g}
}
