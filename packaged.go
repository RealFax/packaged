package packaged

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Kit struct {
	context.Context
	cancelFunc context.CancelCauseFunc
	namespaces map[string]Namespace
	units      Units
	stopOnce   sync.Once
	Logger
}

func (k *Kit) stopServices() {
	k.Warn("packaged: stop services.", "total_entries", len(k.units))

	k.units.Sort(true)

	for _, unit := range k.units {
		if err := unit.Entry.OnStop(); err != nil {
			k.Logger.Error(
				"packaged: failed to call on_stop.",
				"entry_name", unit.Entry.Name(),
				"cause", err.Error(),
			)
		}
	}

	k.Info("packaged: all services stopped.")
}

func (k *Kit) runEntry(u *Unit) error {
	switch u.Entry.Type() {
	case ServiceTypeIgnore:
		k.Warn("packaged: ignoring service.", "name", u.Entry.Name())
		return nil
	case ServiceTypeBlocking:
		return runBlocking(u.MaxRetry, u.RestartPolicy, u.Entry)
	case ServiceTypeAsync:
		runAsync(k.Context, u.MaxRetry, u.RestartPolicy, u.Entry, k.Logger)
		return nil
	default:
		k.Warn("packaged: unknown unit type, ignored.", "name", u.Entry.Name(), "type", u.Entry.Type())
		return nil
	}
}

func (k *Kit) Register(newFc NewService, opts ...UnitOptions) {
	unit := &Unit{Namespace: PublicNamespace}
	for _, opt := range opts {
		opt(unit)
	}

	ns, found := k.namespaces[unit.Namespace]
	if !found {
		ns = newNamespace(unit.Namespace)
		k.namespaces[unit.Namespace] = ns
	}

	unit.Entry = newFc(ns)

	k.units = append(k.units, unit)
}

func (k *Kit) Stop() {
	k.stopOnce.Do(func() {
		k.cancelFunc(ErrQuitUnexpectedly)
		k.stopServices()
	})
	return
}

func (k *Kit) Run() error {
	if len(k.units) == 0 {
		k.Warn("packaged: no unit require run.")
		return nil
	}

	k.units.Sort(false)

	k.Warn("packaged: run units.", "total_entries", len(k.units))

	for _, unit := range k.units {
		if err := unit.Entry.OnInstall(); err != nil {
			return fmt.Errorf("packaged: failed to call on_install. name: %s, reason: %s", unit.Entry.Name(), err)
		}
		if err := k.runEntry(unit); err != nil {
			return err
		}
	}
	return nil
}

func (k *Kit) Wait() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	select {
	case <-k.Context.Done():
		return
	case <-sig:
		k.Stop()
		return
	}
}

func New(opts ...KitOptions) *Kit {
	kit := &Kit{
		namespaces: make(map[string]Namespace),
		units:      make(Units, 0, 32),
		Logger:     slog.Default(),
	}

	for _, opt := range opts {
		opt(kit)
	}

	kit.Context, kit.cancelFunc = context.WithCancelCause(context.Background())

	return kit
}
