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
	Groups     map[string]Group
	Entries    Units
	stopOnce   sync.Once
	Logger
}

func (k *Kit) stopServices() {
	k.Warn("packaged: stop services.", "total_entries", len(k.Entries))

	k.Entries.Sort(true)

	for _, entry := range k.Entries {
		if err := entry.Entry.OnStop(); err != nil {
			k.Logger.Error(
				"packaged: failed to call on_stop.",
				"entry_name", entry.Entry.Name(),
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
		return runBlocking(u.MaxRetry, u.RetryDelay, u.RestartPolicy, u.Entry)
	case ServiceTypeAsync:
		runAsync(k.Context, u.MaxRetry, u.RetryDelay, u.RestartPolicy, u.Entry, k.Logger)
		return nil
	default:
		k.Warn("packaged: unknown unit type, ignored.", "name", u.Entry.Name(), "type", u.Entry.Type())
		return nil
	}
}

func (k *Kit) Register(newEntry NewService, opts ...UnitOptions) {
	entry := &Unit{GroupName: DefaultGroupName}
	for _, opt := range opts {
		opt(entry)
	}

	g, found := k.Groups[entry.GroupName]
	if !found {
		g = newGroup(k.Context, entry.GroupName)
		k.Groups[entry.GroupName] = g
	}

	entry.Entry = newEntry(g)

	k.Entries = append(k.Entries, entry)
}

func (k *Kit) Stop() {
	k.stopOnce.Do(func() {
		k.cancelFunc(ErrQuitUnexpectedly)
		k.stopServices()
	})
	return
}

func (k *Kit) Run() error {
	if len(k.Entries) == 0 {
		k.Warn("packaged: no unit require run.")
		return nil
	}

	k.Entries.Sort(false)

	k.Warn("packaged: run Entries.", "total_entries", len(k.Entries))

	setupEntries := make(map[int]struct{})
	for i, entry := range k.Entries {
		if err := entry.Entry.OnInstall(); err != nil {
			return fmt.Errorf("packaged: failed to call on_install. name: %s, reason: %s", entry.Entry.Name(), err)
		}
		if entry.Setup {
			if err := k.runEntry(entry); err != nil {
				return err
			}
			setupEntries[i] = struct{}{}
		}
	}

	for i, entry := range k.Entries {
		if _, found := setupEntries[i]; found {
			// ignore setup unit
			continue
		}
		if err := k.runEntry(entry); err != nil {
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
		Groups:  make(map[string]Group),
		Entries: make(Units, 0, 32),
		Logger:  slog.Default(),
	}

	for _, opt := range opts {
		opt(kit)
	}

	kit.Context, kit.cancelFunc = context.WithCancelCause(context.Background())

	return kit
}
