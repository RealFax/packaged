package packaged

var internal = New()

func Register(newFc NewService, opts ...UnitOptions) { internal.Register(newFc, opts...) }
func Stop()                                          { internal.Stop() }
func Run() error                                     { return internal.Run() }
func Wait()                                          { internal.Wait() }
