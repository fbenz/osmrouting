package route

import (
	"graph"
)

type Config struct {
	Transport graph.Transport
	Metric    graph.Metric
	AvoidFerries bool
}
