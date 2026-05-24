package normalization

// Normalizable is an interface that requires a Normalize method which returns a value of type T and an error.
type Normalizable[T any] interface {
	Normalize() (T, error)
}

// Apply takes a Normalizable object and calls its Normalize method, returning the normalized value and any error that occurs during normalization.
func Apply[T any](t Normalizable[T]) (T, error) {
	return t.Normalize()
}