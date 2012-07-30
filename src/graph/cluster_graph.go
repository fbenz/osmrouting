package graph

import (
	"fmt"
	"path"
)

type ClusterGraph struct {
	Overlay *OverlayGraphFile
	Cluster []*GraphFile
}

func OpenClusterGraph(base string, loadMatrices bool) (*ClusterGraph, error) {
	overlay, err := OpenOverlay(base, loadMatrices, false /* ignoreErrors */)
	if err != nil {
		return nil, err
	}

	cluster := make([]*GraphFile, overlay.ClusterCount())
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
