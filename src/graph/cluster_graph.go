/*
 * Copyright 2014 Florian Benz, Steven Schäfer, Bernhard Schommer
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
