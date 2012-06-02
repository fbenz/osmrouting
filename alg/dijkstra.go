package alg

import (
	"container/list"
	"../graph"
	"../pq"
)


// A slightly optimized version of dijkstras algorithm
// Takes an graph as argument and returns an list of vertices in order
// of the path
func Dijkstra (g *graph.Graph,s,t uint) (int,*list.List){
	d:=make(map[uint]int) //I assume distance is an integer
	p:=make(map[uint]uint) //Predecessor list
	q :=pq.New(100) //100 is just a first guess
	for q.Len()!=0 {
		currelem := (q.Pop()).(pq.Element) //Get the first element
		curr := currelem.Value.(uint) //Unbox the id
		if curr == t { // If we remove t from the queue we can stop since dist(x)>=dist(t) for all x in q
			break
		}
		currdist := d[curr]
		for _,e := range (*g).Outgoing(curr) {
			n:=e.Endpoint()
			if dist,ok:= d[n];ok {
				if tmpdist:= currdist+e.Weight();tmpdist<dist{
					q.ChangePriority(&currelem,tmpdist)
					p[n]=curr
				}
			} else {
				d[n]=currdist + e.Weight()
				p[n]=curr
				elem := pq.NewElement(n,currdist)
				q.Push(elem)
			}
		}
	}
	path := list.New() 
	// Construct the list by moving from t to s
	for curr:=t;curr!=s; {
		path.PushFront(curr)
		curr = p[curr]
	}
	path.PushFront(s)
	return d[t],path
}