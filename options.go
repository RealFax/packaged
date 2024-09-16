package packaged

type (
	NewService  func(ns Namespace) Service
	KitOptions  func(*Kit)
	UnitOptions func(*Unit)
)

func WithIndex(idx int32) UnitOptions          { return func(u *Unit) { u.Index = idx } }
func WithMaxRetry(retry int32) UnitOptions     { return func(u *Unit) { u.MaxRetry = retry } }
func WithNamespace(ns string) UnitOptions      { return func(u *Unit) { u.Namespace = ns } }
func WithDescription(desc string) UnitOptions  { return func(u *Unit) { u.Description = desc } }
func WithRestartPolicy(rp Restart) UnitOptions { return func(u *Unit) { u.RestartPolicy = rp } }
func WithLogger(logger Logger) KitOptions      { return func(kit *Kit) { kit.Logger = logger } }
