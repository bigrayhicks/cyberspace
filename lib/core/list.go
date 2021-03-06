package core

import (
	"github.com/stojg/vivere/lib/components"
	"math"
	"sync"
)

// List is the primary resource for adding, removing and changing GameObjects and their components.
var List *ObjectList

func init() {
	List = &ObjectList{
		entities:    make(map[components.Entity]*GameObject),
		graphics:    make(map[components.Entity]*Graphic),
		bodies:      make(map[components.Entity]*Body),
		collisions:  make(map[components.Entity]*Collision),
		agents:      make(map[components.Entity]*Agent),
		inventories: make(map[components.Entity]*Inventory),
		deleted:     make([]components.Entity, 0),
	}
}

// ObjectList is a struct that contains a list of GameObjects and their components. All creation,
// removal and changes should be handled by this list so they don't get lost or out of sync.
type ObjectList struct {
	sync.Mutex
	nextID      components.Entity
	entities    map[components.Entity]*GameObject
	graphics    map[components.Entity]*Graphic
	bodies      map[components.Entity]*Body
	collisions  map[components.Entity]*Collision
	agents      map[components.Entity]*Agent
	inventories map[components.Entity]*Inventory
	deleted     []components.Entity
}

// Add a GameObject to this list and assign it an unique ID
func (l *ObjectList) Add(g *GameObject) {
	l.Lock()
	defer l.Unlock()
	if l.nextID == math.MaxUint32 {
		panic("Out of entity ids, implement GC")
	}
	l.nextID++
	g.id = l.nextID
	l.entities[g.id] = g
}

// Remove a GameObject and all of it's components
func (l *ObjectList) Remove(g *GameObject) {
	l.Lock()
	defer l.Unlock()
	if _, found := l.graphics[g.id]; found {
		delete(l.graphics, g.id)
	}
	if _, found := l.bodies[g.id]; found {
		delete(l.bodies, g.id)
	}
	if _, found := l.agents[g.id]; found {
		delete(l.agents, g.id)
	}
	if _, found := l.collisions[g.id]; found {
		delete(l.collisions, g.id)
	}
	if _, found := l.entities[g.id]; found {
		delete(l.entities, g.id)
	}
	if _, found := l.inventories[g.id]; found {
		delete(l.inventories, g.id)
	}
	l.deleted = append(l.deleted, g.id)
}

// All returns all GameObjects in this list
func (l *ObjectList) All() []*GameObject {
	l.Lock()
	var result []*GameObject
	for i := range l.entities {
		result = append(result, l.entities[i])
	}
	l.Unlock()
	return result
}

// FindWithTag returns all GameObjects tagged with tag.
func (l *ObjectList) FindWithTag(tag string) []*GameObject {
	l.Lock()
	var result []*GameObject
	for i := range l.entities {
		if l.entities[i].CompareTag(tag) {
			result = append(result, l.entities[i])
		}
	}
	l.Unlock()
	return result
}

// AddGraphic adds a Graphic component to a GameObject
func (l *ObjectList) AddGraphic(id components.Entity, graphic *Graphic) {
	l.Lock()
	graphic.gameObject = l.entities[id]
	graphic.transform = l.entities[id].transform
	l.graphics[id] = graphic
	l.Unlock()
}

// Graphic returns the Graphic component for a GameObject
func (l *ObjectList) Graphic(id components.Entity) *Graphic {
	return l.graphics[id]
}

// Graphics returns all Graphic components
func (l *ObjectList) Graphics() []*Graphic {
	l.Lock()
	var result []*Graphic
	for i := range l.graphics {
		result = append(result, l.graphics[i])
	}
	l.Unlock()
	return result
}

// AddBody adds a Body component to a GameObject
func (l *ObjectList) AddBody(id components.Entity, body *Body) {
	l.Lock()
	body.gameObject = l.entities[id]
	body.transform = l.entities[id].transform
	l.bodies[id] = body
	l.Unlock()
}

// Body returns the Body component for a GameObject
func (l *ObjectList) Body(id components.Entity) *Body {
	return l.bodies[id]
}

// Bodies returns all Body components
func (l *ObjectList) Bodies() []*Body {
	l.Lock()
	var result []*Body
	for i := range l.bodies {
		result = append(result, l.bodies[i])
	}
	l.Unlock()
	return result
}

// AddCollision adds a Collision component to a GameObject
func (l *ObjectList) AddCollision(id components.Entity, collision *Collision) {
	l.Lock()
	collision.gameObject = l.entities[id]
	collision.transform = l.entities[id].transform
	l.collisions[id] = collision
	l.Unlock()
}

// Collision returns the Collision component for a GameObject
func (l *ObjectList) Collision(id components.Entity) *Collision {
	return l.collisions[id]
}

// Collisions returns all registered Collision components
func (l *ObjectList) Collisions() []*Collision {
	l.Lock()
	var result []*Collision
	for i := range l.collisions {
		result = append(result, l.collisions[i])
	}
	l.Unlock()
	return result
}

// AddAgent adds an Agent component to a GameObject
func (l *ObjectList) AddAgent(id components.Entity, agent *Agent) {
	l.Lock()
	agent.gameObject = l.entities[id]
	l.agents[id] = agent
	l.Unlock()
}

// Agent returns the Agent component for a GameObject
func (l *ObjectList) Agent(id components.Entity) *Agent {
	return l.agents[id]
}

// Agents returns all registered Agent components
func (l *ObjectList) Agents() []*Agent {
	l.Lock()
	var result []*Agent
	for i := range l.agents {
		result = append(result, l.agents[i])
	}
	l.Unlock()
	return result
}

// Deleted returns a list of GameObject IDs that has been deleted/removed
func (l *ObjectList) Deleted() []components.Entity {
	l.Lock()
	defer l.Unlock()
	return l.deleted
}

// ClearDeleted clears the list of deleted GameObjects
func (l *ObjectList) ClearDeleted() {
	l.Lock()
	defer l.Unlock()
	l.deleted = make([]components.Entity, 0)
}
