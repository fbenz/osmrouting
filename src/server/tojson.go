package main

import (
	"graph"
	"container/list"
	"fmt"
)

func StepToPolyline (steps []graph.Step, u, v graph.Node) Polyline {
	polyline := make([]Point,len(steps)+2)
	first:=make([]float64,2)
	lat1,long1 := u.LatLng()
	first[0]=lat1
	first[1]=long1
	polyline[0]=first
	for i,s:=range steps {
		point:= make([]float64,2)
		point[0]=s.Lat
		point[1]=s.Lng
		polyline[i+1]=point
	}
	last:=make([]float64,2)
	lat2,long2 := v.LatLng()
	last[0]=lat2
	last[1]=long2
	polyline[len(steps)+1]=last
	return polyline
}

func WayToStep(w graph.Way,u,v graph.Node) (Step){
	dist := Distance{fmt.Sprintf("%.2f m", w.Length),int(w.Length)}
	dur := Duration{"? s",42}
	start:=make([]float64,2)
	lat1,long1:=u.LatLng()
	start[0]=lat1
	start[1]=long1
	end :=make([]float64,2)
	lat2,long2:=v.LatLng()
	end[0]=lat2
	end[1]=long2
	poly:=StepToPolyline(w.Steps,u,v)
	instruction:="TODO"
	return Step{dist,dur,start,end,poly,instruction}
}

func EdgeToStep (e graph.Edge,u,v graph.Node) (Step){
	dist := Distance{fmt.Sprintf("%.2f m", e.Length()),int(e.Length())}
	dur := Duration{"? s",42}
	start:=make([]float64,2)
	lat1,long1:=u.LatLng()
	start[0]=lat1
	start[1]=long1
	end :=make([]float64,2)
	lat2,long2:=v.LatLng()
	end[0]=lat2
	end[1]=long2
	poly:=StepToPolyline(e.Steps(),u,v)
	instruction:= fmt.Sprintf("(%.4f, %.4f) -> (%.4f, %.4f)", lat1, long1, lat2, long2)
	return Step{dist,dur,start,end,poly,instruction}
}

func PathToLeg (dist float64, vertex, edge *list.List,startway,endway graph.Way) (Leg) {
	distance := Distance{fmt.Sprintf("%.2f m", dist),int(dist)}
	dur := Duration{"? s",42}
	steps:=make([]Step,edge.Len()+2)
	startvertex := vertex.Front().Value.(graph.Node)
	steps[0]=WayToStep(startway,startway.Node,startvertex)
	for v,e,i:=vertex.Front(),edge.Front(),1;e!=edge.Back();v,e,i=v.Next(),e.Next(),i+1 {
		fmt.Printf("v: %v\n", v)
		fmt.Printf("e: %v\n", e)
		fmt.Printf("i: %v\n", i)
		ue:=e.Value.(graph.Edge)
		uv:=v.Value.(graph.Node)
		nuv:=v.Next().Value.(graph.Node)
		steps[i]=EdgeToStep(ue,uv,nuv)
	}
	steps[len(steps)-1]=WayToStep(endway,vertex.Back().Value.(graph.Node),endway.Node)
	return Leg{distance,dur,steps[0].StartLocation,steps[len(steps)-1].EndLocation,steps}
}
