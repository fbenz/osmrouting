package graph

import (
	"fmt"
	"path"
)

type ClusterGraph struct {
	Overlay OverlayGraph
	Cluster []Graph
}

// OpenGraphFile(path string, ignoreErrors bool) (*GraphFile, error) 
func OpenClusterGraph(baseDir string) (*ClusterGraph, error) {
	overlayGraphFile, err := OpenGraphFile(path.Join(baseDir, "/overlay"), false /* ignoreErrors */)
	if err != nil {
		return nil, err
	}
	// load cluster file for overlay graph
	// overlayGraph := OverlayGraphFile{GraphFile: overlayGraphFile, Cluster: cluster}

	// TODO the following three lines are just a hack to get it running
	hack := make([]OverlayGraph, 1)
	overlayGraph := hack[0]
	_ = overlayGraphFile

	cluster := make([]Graph, overlayGraph.ClusterCount())
	for i, _ := range cluster {
		clusterDir := fmt.Sprintf("/cluster%d", i+1)
		g, err := OpenGraphFile(path.Join(baseDir, clusterDir), false /* ignoreErrors */)
		if err != nil {
			return nil, err
		}
		cluster[i] = g
	}
	return &ClusterGraph{Overlay: overlayGraph, Cluster: cluster}, nil
}
