// Actually a 2-d tree for the dimensions latitude and longitude

package alg

import (
    "sort"
)

type NodeData struct {
    Lat float64
    Lng float64
}

type NodeDataSlice []NodeData

type Nodes []int

type KdTree struct {
    Nodes []int
    Data NodeDataSlice
}

func (s Nodes) Len() int      { return len(s) }
func (s Nodes) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type ByLat struct {
    Nodes
    Tree *KdTree
}
func (x ByLat) Less(i, j int) bool { return x.Tree.Data[x.Nodes[i]].Lat < x.Tree.Data[x.Nodes[j]].Lat }

type ByLng struct {
    Nodes
    Tree *KdTree
}
func (x ByLng) Less(i, j int) bool { return x.Tree.Data[x.Nodes[i]].Lng < x.Tree.Data[x.Nodes[j]].Lng }

func (t KdTree) getData(i int) NodeData {
    return t.Data[t.Nodes[i]]
}

func NewKdTree(data NodeDataSlice) KdTree {
    nodes := make([]int, DataSetSize)
    for i := 0; i < DataSetSize; i++ {
        nodes[i] = i
    }
    t := KdTree{nodes, data}
    t.create(t.Nodes, true)
    return t
}

func (t KdTree) create(nodes Nodes, compareLat bool) {
    if len(nodes) <= 1 {
        return
    }
    if compareLat {
        sort.Sort(ByLat{nodes, &t})
    } else {
        sort.Sort(ByLng{nodes, &t})
    }
    middle := len(nodes) / 2
    t.create(nodes[:middle], !compareLat) // correct without -1 as the upper bound is equal to the length
    t.create(nodes[middle+1:], !compareLat)
}

func (t KdTree) Search(x NodeData) int {
    return t.search(x, true)
}

func (t KdTree) search(x NodeData, compareLat bool) int {
    index, lineSearch := t.binarySearch(x, compareLat)
    if lineSearch {
        if x.Lat == t.getData(index).Lat {
            return t.lineSearch(x)
        }
        return t.lineSearch(x)
    }
    return t.Nodes[index]
}


func (t KdTree) binarySearch(x NodeData, compareLat bool) (int, bool) {
    if len(t.Nodes) == 0 {
        panic("KdTree.binarySearch")
    } else if len(t.Nodes) == 1 {
        return 0, false
    }
    middle := len(t.Nodes) / 2
    
    // exact hit
    if x.Lat == t.getData(middle).Lat && x.Lng == t.getData(middle).Lng {
        return middle, false
    }
    
    // If two or more nodes have lat/lng in common with the given point, 
    // we can not guarantee to hit OSM with exactly the coordinates of the given point.
    // But this is required for the project at the moment, so we switch to line search.
    if compareLat && x.Lat == t.getData(middle).Lat {
        return middle, true
    }
    if !compareLat && x.Lng == t.getData(middle).Lng {
        return middle, true
    }

    var left bool
    if compareLat {
        left = x.Lat < t.getData(middle).Lat
    } else {
        left = x.Lng < t.getData(middle).Lng
    }
    if left {
        // stop if there is nothing left of the middle
        if middle == 0 {
            return middle, false
        }
        return KdTree{t.Nodes[:middle], t.Data}.binarySearch(x, !compareLat)
    }
    // stop if there is nothing right of the middle
    if middle == len(t.Nodes) - 1 {
        return middle, false
    }
    index, lineSearch := KdTree{t.Nodes[middle+1:], t.Data}.binarySearch(x, !compareLat)
    return middle + 1 + index, lineSearch
}

func (t KdTree) lineSearch(x NodeData) int {
    for i := range t.Nodes {
        if x.Lat == t.Data[i].Lat && x.Lng == t.Data[i].Lng {
            return i
        }
    }
    panic("KdTree.lineaSearch")
}

// getCoordinate returns the requested coordinate
/*func getCoordinate(x NodeData, lat bool) float64 {
    if lat {
        return x.Lat
    }
    return x.Lng
}*/

// lineSearch performs a line search where one coordinate is fixed and the other one is minimized
// It is implement more general than needed: it minimizes the distance of the other
// coordinate instead of simply comparing. The general version is more than 10x slower
// than a line search simply comparing for an exact hit.
/*func (t KdTree) lineSearch(startIndex int, x NodeData, searchLat bool) int {
    minDist := math.Abs(getCoordinate(x, searchLat) - getCoordinate(t.getData(startIndex), searchLat))
    minDistPos := startIndex
    for i := 0; i < len(t.Nodes); i++ {
        if getCoordinate(x, !searchLat) != getCoordinate(t.getData(i), !searchLat) {
            continue
        }
        dist := math.Abs(getCoordinate(x, searchLat) - getCoordinate(t.getData(i), searchLat))
        if dist < minDist {
            minDist = dist
            minDistPos = i
        }
    }
    return minDistPos
}*/

