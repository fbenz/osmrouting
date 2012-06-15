// Wrap a file + filter in a nicer interface.

package pbf

import (
	"os"
)

type Graph struct {
	file   *os.File
	access AccessType
}

// I wonder if this is really such a good idea...
func NewGraph(file *os.File, access AccessType) Graph {
	return Graph{
		file:   file,
		access: access,
	}
}

func (g Graph) Traverse(client Visitor) error {
	_, err := g.file.Seek(0, 0)
	if err != nil {
		return err
	}
	
	filter := NewRoutingFilter(client, g.access)
	return VisitGraph(g.file, filter)
}
