package sakayukari

type ActorKey = string

type GraphStateMap = map[ActorKey]int

type Actor2 struct {
	DependsOn   []ActorKey
	UpdateFunc  func(self *Actor, gsm GraphStateMap, gs GraphState) (updated Value)
	RecvChan    chan Value
	SideEffects bool
	Comment     string
}

type Graph2 struct {
	Actors map[ActorKey]Actor2
}

func (g2 *Graph2) Convert() *Graph {
	actors := make([]Actor, 0, len(g2.Actors))
	indexes := map[string]int{}
	for key, actor := range g2.Actors {
		indexes[key] = len(actors)
		actor := actor
		actor2 := Actor{
			RecvChan:    actor.RecvChan,
			SideEffects: actor.SideEffects,
			Comment:     actor.Comment,
		}
		if actor.UpdateFunc != nil {
			actor2.UpdateFunc = func(self *Actor, gs GraphState) Value { return actor.UpdateFunc(self, indexes, gs) }
		}
		actors = append(actors, actor2)
	}
	for key, actor := range g2.Actors {
		i := indexes[key]
		dependsOn := make([]ActorIndex, 0, len(actor.DependsOn))
		for _, depKey := range actor.DependsOn {
			dependsOn = append(dependsOn, indexes[depKey])
		}
		actors[i].DependsOn = dependsOn
	}
	return NewGraph(actors)
}
