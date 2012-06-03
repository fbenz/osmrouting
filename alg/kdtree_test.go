
package alg

import (
    "math/rand"
    "testing"
    "time"
)

const (
    DataSetSize = 100000 // more important to the performance
    Repeats = 10000
)

var nodeData = createData(false)
var repeatsNodeData = createData(true)

func createData(repeats bool) NodeDataSlice {
    rnd := rand.New(rand.NewSource(time.Now().Unix()))

    nodeData := make(NodeDataSlice, DataSetSize)
    for i := 0; i < DataSetSize; i++ {
        lat, lng := rnd.Float64(), rnd.Float64()
        nodeData[i] = NodeData{lat, lng}
        
        // insert coordinates that have the same lat/lng (ugly corner case)
        if repeats {
            if rnd.Intn(5) == 0 {
                up := i + 2 + rnd.Intn(100)
                for i++; i < up && i < DataSetSize; i++ {
                    nodeData[i] = NodeData{lat, rnd.Float64()}
                }
                i--
            }
            if rnd.Intn(5) == 0 {
                up := i + 2 + rnd.Intn(100)
                for i++; i < up && i < DataSetSize; i++ {
                    nodeData[i] = NodeData{rnd.Float64(), lng}
                }
                i--
            }
        }
    }
    return nodeData
}

func TestKdTree(t *testing.T) {
    tree := NewKdTree(nodeData)
    
    rnd := rand.New(rand.NewSource(time.Now().Unix()))
    for i := 0; i < Repeats; i++ {
        refIndex := rnd.Intn(DataSetSize)
        x := nodeData[refIndex]
        
        index := tree.Search(x)
        
        if index != refIndex {
            t.Fatalf("Returned wrong index: expected %v but was %v", refIndex, index)
        }
    }
}

func TestKdTreeRepeats(t *testing.T) {
    tree := NewKdTree(nodeData)
    
    rnd := rand.New(rand.NewSource(time.Now().Unix()))
    for i := 0; i < Repeats; i++ {
        refIndex := rnd.Intn(DataSetSize)
        x := nodeData[refIndex]
        
        index := tree.Search(x)
        
        if index != refIndex {
            t.Fatalf("Returned wrong index: expected %v but was %v", refIndex, index)
        }
    }
}
