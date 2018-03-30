package crawler

type Options struct {
	SameHostOnly bool
	MaxVisits int
}

func NewOptions() *Options {
	// Use defaults except for Extender
	return &Options{
		true,
		0,
	}
}