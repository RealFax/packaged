package packaged

// internal holds the service manager instance.
var internal = New()

// Register adds a new service to the internal service manager.
func Register(newFc NewService, opts ...UnitOptions) {
	internal.Register(newFc, opts...)
}

// Stop gracefully stops all services managed by the internal service manager.
func Stop() {
	internal.Stop()
}

// Run starts the internal service manager and launches all registered services.
func Run() error {
	return internal.Run()
}

// Wait blocks execution until all services have completed or stopped.
func Wait() {
	internal.Wait()
}
