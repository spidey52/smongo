package smongo

import (
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Model[T any] struct {
	mu         sync.Mutex
	Collection *mongo.Collection
	Populates  []PopulateField
	Constaints []Constraint
}

func NewModel[T any](db *mongo.Database, collectionName string) *Model[T] {
	SetCollection(collectionName, db.Collection(collectionName))
	return &Model[T]{Collection: db.Collection(collectionName)}
}

func (m *Model[T]) Where(filter bson.M) *Query[T] {
	return &Query[T]{
		model:      m,
		filter:     filter,
		opts:       options.Find(),
		findOneOpt: options.FindOne(),
	}
}

func (m *Model[T]) AddPopulates(populates ...PopulateField) *Model[T] {
	m.mu.Lock()
	defer m.mu.Unlock()

	added := map[string]bool{}

	for _, populate := range m.Populates {
		added[populate.LocalField] = true
	}

	//  Add new populates
	for _, populate := range populates {
		if _, ok := added[populate.LocalField]; !ok {
			m.Populates = append(m.Populates, populate)
			added[populate.LocalField] = true

			if _, ok := populateCollections[populate.From]; !ok {
				populateCollections[populate.From] = true
			}

		}
	}

	return m
}

func (m *Model[T]) IsValidPopulateKey(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, populate := range m.Populates {
		if key == populate.LocalField {
			return true
		}
	}
	return false
}

func (m *Model[T]) AddConstraints(constraints ...Constraint) *Model[T] {
	m.Constaints = append(m.Constaints, constraints...)
	return m
}

func (m *Model[T]) GetPopulateField(key string) PopulateField {

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, populate := range m.Populates {
		if key == populate.LocalField {
			return populate
		}
	}

	return PopulateField{}
}
