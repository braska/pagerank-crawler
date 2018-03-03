package crawler

type Options struct {
	SameHostOnly bool
}

func NewOptions() *Options {
	// Use defaults except for Extender
	return &Options{
		true,
	}
}