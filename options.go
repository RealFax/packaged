package packaged

import "time"

type (
	NewService  func(g Group) Service
	KitOptions  func(*Kit)
	UnitOptions func(*Unit)
)

// WithSetup
//
// Set the unit type to setup, OnInstall and OnStart will be executed immediately without delay
func WithSetup() UnitOptions { return func(u *Unit) { u.Setup = true } }

func WithIndex(idx int32) UnitOptions                  { return func(u *Unit) { u.Index = idx } }
func WithMaxRetry(retry int32) UnitOptions             { return func(u *Unit) { u.MaxRetry = retry } }
func WithGroup(name string) UnitOptions                { return func(u *Unit) { u.GroupName = name } }
func WithDescription(desc string) UnitOptions          { return func(u *Unit) { u.Description = desc } }
func WithRestartPolicy(rp Restart) UnitOptions         { return func(u *Unit) { u.RestartPolicy = rp } }
func WithRestartDelay(delay time.Duration) UnitOptions { return func(u *Unit) { u.RetryDelay = delay } }

func WithLogger(logger Logger) KitOptions { return func(kit *Kit) { kit.Logger = logger } }
