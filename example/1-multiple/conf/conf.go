package conf

import (
	"encoding/json"
	"github.com/RealFax/packaged"
)

const config = `{"addr": "127.0.0.1:8080"}`

type ConfigStruct struct {
	Addr string `json:"addr"`
}

type service struct {
	packaged.Unimplemented
	g packaged.Group
	c ConfigStruct
}

func (s *service) Type() packaged.ServiceType { return packaged.ServiceTypeBlocking }

func (s *service) OnInstall() error {
	// read config & stored
	return json.Unmarshal([]byte(config), &s.c)
}

func (s *service) OnStart() error {
	// setup config in group
	s.g.Set("addr", s.c.Addr)
	return nil
}

func NewEntry(g packaged.Group) packaged.Service {
	return &service{g: g}
}
