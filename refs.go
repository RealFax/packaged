package packaged

import (
	"context"
	"errors"
	"sort"
	"sync"
)

var (
	_ Service        = &UnimplementedHandler{}
	_ Namespace      = &namespace{}
	_ sort.Interface = &unitSorter{}
)

type (
	ServiceType int32
	Restart     int32

	Logger interface {
		Debug(msg string, args ...any)
		Info(msg string, args ...any)
		Warn(msg string, args ...any)
		Error(msg string, args ...any)
	}

	Service interface {
		mustEmbedUnimplemented()
		Name() string
		Type() ServiceType
		OnInstall() error
		OnStart() error
		OnStop() error
	}

	Namespace interface {
		Name() string
		Set(key string, value any)
		Del(key string)
		Get(key string) (any, bool)
		Entries() []Service
		EnvManager
	}

	Unit struct {
		Index, MaxRetry int32
		Namespace       string
		Description     string
		Entry           Service
		RestartPolicy   Restart
	}

	Units []*Unit

	Context struct {
		context.Context
		cancelFunc context.CancelCauseFunc
		Namespace

		stopOnce sync.Once
	}

	UnimplementedHandler struct{}

	namespace struct {
		name     string
		rw       sync.RWMutex
		values   map[string]any
		services []Service
		EnvManager
	}

	unitSorter struct {
		units Units
		desc  bool
	}
)

var (
	ErrQuitUnexpectedly = errors.New("quit unexpectedly")
)

const PublicNamespace = "__public_namespace__"

const (
	ServiceTypeIgnore ServiceType = iota
	ServiceTypeBlocking
	ServiceTypeAsync
)

const (
	RestartIgnore Restart = iota
	RestartAlways
	RestartRetry
)

func (h UnimplementedHandler) mustEmbedUnimplemented() {}
func (h UnimplementedHandler) Name() string            { return "unimplemented" }
func (h UnimplementedHandler) Type() ServiceType       { return ServiceTypeIgnore }
func (h UnimplementedHandler) OnInstall() error        { return errors.New("unimplemented OnInstall") }
func (h UnimplementedHandler) OnStart() error          { return errors.New("unimplemented OnStart") }
func (h UnimplementedHandler) OnStop() error           { return errors.New("unimplemented OnStop") }

func (n *namespace) Name() string { return n.name }

func (n *namespace) setValueWithLock(action func()) {
	n.rw.Lock()
	defer n.rw.Unlock()
	action()
}

func (n *namespace) Set(key string, value any) {
	n.setValueWithLock(func() {
		n.values[key] = value
	})
}

func (n *namespace) Del(key string) {
	n.setValueWithLock(func() {
		delete(n.values, key)
	})
}

func (n *namespace) Get(key string) (any, bool) {
	n.rw.RLock()
	defer n.rw.RUnlock()
	v, ok := n.values[key]
	return v, ok
}

func (n *namespace) Entries() []Service {
	return n.services
}

func (c *Context) Stop() {
	c.stopOnce.Do(func() {
		c.cancelFunc(ErrQuitUnexpectedly)
	})
}

func newNamespace(name string) Namespace {
	var env EnvManager
	if name == PublicNamespace {
		env = lookupPrefix("")
	} else {
		env = lookupPrefix(name)
	}
	return &namespace{
		name:       name,
		values:     make(map[string]any, 16),
		services:   make([]Service, 0, 8),
		EnvManager: env,
	}
}

func (s unitSorter) Len() int { return len(s.units) }
func (s unitSorter) Less(i, j int) bool {
	return (s.units[i].Index > s.units[j].Index) == s.desc
}

func (s unitSorter) Swap(i, j int) {
	s.units[i], s.units[j] = s.units[j], s.units[i]
}

func (u Units) Sort(desc bool) {
	sort.Sort(unitSorter{units: u, desc: desc})
}
