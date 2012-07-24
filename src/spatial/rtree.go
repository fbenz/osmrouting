
package spatial

import (
	"mm"
	"geo"
)

const (
	Degree = 4
)

type RTreeNode struct {
	// encoded bbox
	Bounds   [4]int32
	// An index into an rtree has to distinguish three cases:
	// * i == 0: child does not exist
	// * i  > 0: child is another RTreeNode with index i - 1
	// * i  < 0: child is a leaf with index -i - i (= ^i)
	Children [Degree]int
}

type RTree struct {
	// tree index over the element array
	Nodes    []RTreeNode
	// file mapping with encoded bboxes in hilbert order
	Elements []int32
}

// Number of interior nodes of a d-ary tree with n leafs.
func TreeSize(n, d int) int {
	m := 0
	for n > 1 {
		n = (n + d - 1) / d
		m += n
	}
	return m
}

func (r *RTree) ElementBBox(i int) geo.BBox {
	return geo.DecodeBBox(r.Elements[4 * i:])
}

func (r *RTree) NodeBBox(i int) geo.BBox {
	return geo.DecodeBBox(r.Nodes[i].Bounds[:])
}

func (r *RTree) BBox(i int) geo.BBox {
	if i > 0 {
		return r.NodeBBox(i - 1)
	} else if i < 0 {
		return r.ElementBBox(^i)
	}
	return geo.EmptyBBox()
}

func (r *RTree) Query(q geo.Coordinate) []int {
	if len(r.Nodes) == 0 {
		return nil
	}
	result := make([]int, 0)
	stack  := make([]int, 1)
	stack[0] = 1 // <- root node
	
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, t := range r.Nodes[s - 1].Children {
			if t == 0 {
				continue
			}
			
			if r.BBox(t).Contains(q) {
				if t < 0 {
					result = append(result, ^t)
				} else {
					stack = append(stack, t)
				}
			}
		}
	}
	
	return result
}

// Pack the RTree - that is, generate the tree nodes.
// We do this in two steps; first we count the number of nodes to create,
// then we create the nodes bottom up.
func PackRTree(r *RTree) {
	r.Nodes := make([]RTreeNode, TreeSize(len(r.Elements), Degree))
	
	levelSize   := (len(r.Elements) + Degree - 1) / Degree
	levelOffset := len(r.Nodes) - levelSize
	level := r.Elements[levelOffset:]
	
	// Pack leaf nodes
	for i, k := 0, 0; i < levelSize; i, k = i + 1, k + Degree {
		// Compute the degree of this node
		d := len(r.Elements) - k
		if d > Degree {
			d = Degree
		}
		// Compute the children/bounds of the current node
		level[i].Children[0] = ^k
		bounds := r.ElementBBox(k)
		for j := 1; j < d; j++ {
			bounds = bounds.Union(r.ElementBBox(k + j))
			level[i].Children[j] = ^(k + j)
		}
		level[i].Bounds = bounds.Encode()
	}
	
	// Pack internal nodes
	for levelOffset > 1 {
		packSize := levelSize
		packOffset := levelOffset
		levelSize = (levelSize + Degree - 1) / Degree
		levelOffset -= levelSize
		level = r.Elements[levelOffset:]
		
		for i, k := 0, 0; i < levelSize; i, k = i + 1, k + Degree {
			// Compute the degree of this node
			d := packSize - k
			if d > Degree {
				d = Degree
			}
			// Compute the children/bounds of the current node
			level[i].Children[0] = 1 + packOffset + k
			bounds := r.NodeBBox(packOffset + k)
			for j := 1; j < d; j++ {
				bounds = bounds.Union(r.ElementBBox(packOffset + k + j))
				level[i].Children[j] = 1 + packOffset + k + j
			}
			level[i].Bounds = bounds.Encode()
		}
	}
}

func LoadRTree(file string) (*RTree, error) {
	// The elements array is stored on disk in Hilbert order.
	// Actually, the order can be arbitrary, it's just that the index will
	// be worthless unless there is some kind of locality preserving ordering.
	r := &RTree{}
	err := mm.Open(file, &r.Elements)
	if err != nil {
		return nil, err
	}
	PackRTree(r)
	return r, nil
}
