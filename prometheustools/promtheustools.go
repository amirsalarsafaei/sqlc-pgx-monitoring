package prometheustools

type Observer interface {
	Observe(float64)
	With(labels map[string]string) Observer
}
