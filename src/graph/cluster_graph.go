package graph

import (
	"fmt"
	"path"
)

type ClusterGraph struct {
	Overlay OverlayGraph
	Cluster []Graph
}

func OpenClusterGraph(base string) (*ClusterGraph, error) {
	overlay, err := OpenOverlay(path.Join(base, "/overlay"), false /* ignoreErrors */)
	if err != nil {
		return nil, err
	}

	cluster := make([]Graph, overlay.ClusterCount())
	for i := range cluster {
		clusterDir := fmt.Sprintf("/cluster%d", i+1)
		g, err := OpenGraphFile(path.Join(base, clusterDir), false /* ignoreErrors */)
		if err != nil {
			return nil, err
		}
		cluster[i] = g
	}
	return &ClusterGraph{Overlay: overlay, Cluster: cluster}, nil
}
