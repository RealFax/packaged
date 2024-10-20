package packaged

import (
	"context"
	"errors"
	"maps"
	"sort"
	"sync"
	"time"
)

var (
	_ Service        = &Unimplemented{}
	_ Group          = &group{}
	_ sort.Interface = &entriesSorter{}
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

	Group interface {
		context.Context
		Name() string
		Set(key string, value any)
		Del(key string)
		Get(key string) (any, bool)
		GetString(key string) (string, bool)
		Values() map[string]any
		Entries() []Service
		EnvManager
	}

	Unit struct {
		Setup           bool
		Index, MaxRetry int32
		RetryDelay      time.Duration
		GroupName       string
		Description     string
		Entry           Service
		RestartPolicy   Restart
	}

	Units []*Unit

	Context struct {
		context.Context
		cancelFunc context.CancelCauseFunc
		Group

		stopOnce sync.Once
	}

	Unimplemented struct{}

	group struct {
		context.Context
		name     string
		rw       sync.RWMutex
		values   map[string]any
		services []Service
		EnvManager
	}

	entriesSorter struct {
		entries Units
		desc    bool
	}
)

var (
	ErrQuitUnexpectedly = errors.New("quit unexpectedly")
)

const DefaultGroupName = "__group__"

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

func (h Unimplemented) mustEmbedUnimplemented() {}
func (h Unimplemented) Name() string            { return "Unnamed-Service" }
func (h Unimplemented) Type() ServiceType       { return ServiceTypeIgnore }
func (h Unimplemented) OnInstall() error        { return nil }
func (h Unimplemented) OnStart() error          { return errors.New("unimplemented OnStart") }
func (h Unimplemented) OnStop() error           { return nil }

func (n *group) Name() string { return n.name }

func (n *group) setValueWithLock(action func()) {
	n.rw.Lock()
	defer n.rw.Unlock()
	action()
}

func (n *group) Set(key string, value any) {
	n.setValueWithLock(func() {
		n.values[key] = value
	})
}

func (n *group) Del(key string) {
	n.setValueWithLock(func() {
		delete(n.values, key)
	})
}

func (n *group) Get(key string) (any, bool) {
	n.rw.RLock()
	defer n.rw.RUnlock()
	v, ok := n.values[key]
	return v, ok
}

func (n *group) GetString(key string) (string, bool) {
	return Assert[string](key, n)
}

func (n *group) Values() map[string]any {
	n.rw.RLock()
	defer n.rw.RUnlock()
	return maps.Clone(n.values)
}

func (n *group) Entries() []Service {
	return n.services
}

func (c *Context) Stop() {
	c.stopOnce.Do(func() {
		c.cancelFunc(ErrQuitUnexpectedly)
	})
}

func newGroup(ctx context.Context, name string) Group {
	var env EnvManager
	if name == DefaultGroupName {
		env = lookupPrefix("")
	} else {
		env = lookupPrefix(name)
	}
	return &group{
		Context:    ctx,
		name:       name,
		values:     make(map[string]any, 16),
		services:   make([]Service, 0, 8),
		EnvManager: env,
	}
}

func (s entriesSorter) Len() int { return len(s.entries) }
func (s entriesSorter) Less(i, j int) bool {
	return (s.entries[i].Index > s.entries[j].Index) == s.desc
}

func (s entriesSorter) Swap(i, j int) {
	s.entries[i], s.entries[j] = s.entries[j], s.entries[i]
}

func (u Units) Sort(desc bool) {
	fastForward := true
	for _, unit := range u {
		if unit.Index != 0 && fastForward {
			fastForward = false
		}
	}

	if fastForward {
		// keep raw order
		return
	}

	sort.Sort(entriesSorter{entries: u, desc: desc})
}
