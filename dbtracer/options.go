package dbtracer

type optionCtx struct {
	name    string
	help    string
	buckets []float64
}

type Option func(*optionCtx)

func WithTimeBuckets(buckets ...float64) Option {
	return func(optCtx *optionCtx) {
		optCtx.buckets = buckets
	}
}

func WithName(name string) Option {
	return func(optCtx *optionCtx) {
		optCtx.name = name
	}
}

func WithHelp(help string) Option {
	return func(optCtx *optionCtx) {
		optCtx.help = help
	}
}
