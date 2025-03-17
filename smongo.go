package smongo

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PopulateField struct {
	From         string
	LocalField   string
	ForeignField string
	As           string
	IsList       bool
}

type PopulateFieldQuery struct {
	PopulateField
	Filter     bson.M
	Opts       *options.FindOptions
	FindOneOpt *options.FindOneOptions
	Populates  []PopulateFieldQuery
}

type Query[T any] struct {
	model      *Model[T]
	filter     bson.M
	opts       *options.FindOptions
	findOneOpt *options.FindOneOptions
	populates  []PopulateFieldQuery
}

type Constraint struct {
	Field     string
	Unique    bool
	Lowercase bool
	Uppercase bool
	Trim      bool
	Min       int
	Max       int
	Required  bool
	Enum      []string
	Default   string
}

var (
	collectionLock      sync.Mutex
	collections         = map[string]*mongo.Collection{}
	populateCollections = map[string]bool{}
)

func GetCollection(collectionName string) *mongo.Collection {
	collectionLock.Lock()
	defer collectionLock.Unlock()

	if _, ok := collections[collectionName]; !ok {
		return nil
	}

	return collections[collectionName]
}

func SetCollection(collectionName string, collection *mongo.Collection) {
	collectionLock.Lock()
	defer collectionLock.Unlock()

	collections[collectionName] = collection

}

func IsValidCollection(collectionName string) bool {
	collectionLock.Lock()
	defer collectionLock.Unlock()

	if _, ok := collections[collectionName]; !ok {
		return false
	}

	return true
}

// TODO: check if LocalField, ForeignField, As are valid fields in the struct
func ValidatePopulatedCollections() {
	for collectionName := range populateCollections {
		if !IsValidCollection(collectionName) {
			panic("Collection not found: " + collectionName)
		}
	}
}

func (q *Query[T]) Sort(sort bson.M) *Query[T] {
	q.opts.SetSort(sort)
	q.findOneOpt.SetSort(sort)
	return q
}

func (q *Query[T]) Select(fields ...string) *Query[T] {

	if len(fields) == 0 {
		return q
	}

	if fields[0] == "" {
		return q
	}

	currentProjection := bson.M{}

	if q.opts.Projection != nil {
		currentProjection = q.opts.Projection.(bson.M)
	}

	newProjection := StrToProjection(currentProjection, fields...)
	q.opts.SetProjection(newProjection)

	return q
}

func (q *Query[T]) Limit(limit int64) *Query[T] {
	q.opts.SetLimit(limit)
	return q
}

func (q *Query[T]) Skip(skip int64) *Query[T] {
	q.opts.SetSkip(skip)
	return q
}

func (q *Query[T]) IsPopulateAdded(key string) bool {
	for _, populate := range q.populates {
		if key == populate.LocalField {
			return true
		}
	}
	return false
}

func (q *Query[T]) Populate(key string, selectFields string) *Query[T] {
	if !q.model.IsValidPopulateKey(key) {
		fmt.Println("Populate field not found", key)
		return q
	}

	if q.IsPopulateAdded(key) {
		fmt.Println("Populate field already added", key)
		return q
	}

	// poplate details are separated by space

	q.populates = append(q.populates, PopulateFieldQuery{
		Opts:          options.Find().SetProjection(StrToProjection(bson.M{}, selectFields)),
		PopulateField: q.model.GetPopulateField(key),
	})

	return q
}

func (q *Query[T]) PopulateAdv(keys ...PopulateFieldQuery) *Query[T] {

	for _, key := range keys {
		if !q.model.IsValidPopulateKey(key.LocalField) {
			fmt.Println("Populate field not found", key.LocalField)
			continue
		}

		if q.IsPopulateAdded(key.LocalField) {
			fmt.Println("Populate field already added", key.LocalField)
			continue
		}

		q.populates = append(q.populates, key)

	}

	return q
}

// Executes a findOne query
func (q *Query[T]) FindOne(ctx context.Context) (T, error) {
	var result T
	err := q.model.Collection.FindOne(ctx, q.filter, q.findOneOpt).Decode(&result)
	if err != nil {
		var empty T
		return empty, err
	}
	return result, nil
}

func (q *Query[T]) AddPopulateProjection() {
	if len(q.populates) == 0 {
		fmt.Println("No populates")
		return
	}

	if q.opts.Projection == nil {
		fmt.Println("Projection is nil")
		return
	}

	// check if exclusive projection is set
	oldProjection := q.opts.Projection.(bson.M)

	for _, p := range oldProjection {
		if p == 0 {
			return
		}
	}

	for _, populate := range q.populates {
		oldProjection := q.opts.Projection.(bson.M)

		if _, ok := oldProjection[populate.LocalField]; !ok {
			oldProjection[populate.LocalField] = 1
		}

		q.opts.SetProjection(oldProjection)

	}

}

func GG(wg *sync.WaitGroup, results []bson.M, populate PopulateFieldQuery, lock *sync.Mutex) {
	defer wg.Done()

	filters := make([]any, 0)
	lock.Lock()
	for _, result := range results {
		fieldName := result[populate.LocalField]

		if fieldName != nil {
			filters = append(filters, fieldName)
		}

	}

	lock.Unlock()

	// now get the populated results
	populatedResults, err := GetPopulatedResults1(filters, GetCollection(populate.From), populate)

	if err != nil {
		return
	}

	if len(populate.Populates) > 0 {
		for _, pop := range populate.Populates {
			wg.Add(1)
			go GG(wg, populatedResults, pop, lock)
		}
	}

	lock.Lock()

	for idx := range results {
		localField := results[idx][populate.LocalField]

		if localField == nil {
			continue
		}

		// find the populated value
		for _, populatedValue := range populatedResults {
			if populatedValue[populate.ForeignField] == localField {
				results[idx][populate.As] = populatedValue
				break
			}
		}

	}

	lock.Unlock()

}

func (q *Query[T]) PopulateResult(resultsPtr *[]bson.M) error {
	results := *resultsPtr

	var wg sync.WaitGroup
	var localLock sync.Mutex

	// Populate fields
	for _, populate := range q.populates {
		wg.Add(1)
		go GG(&wg, results, populate, &localLock)
	}
	wg.Wait()
	return nil
}
