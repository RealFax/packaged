package packaged

func zero[T any]() (zero T) { return }

func Assert[T any](key string, g Group) (T, bool) {
	value, found := g.Get(key)
	if !found {
		return zero[T](), false
	}
	as, ok := value.(T)
	return as, ok
}
