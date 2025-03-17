package smongo

import (
	"context"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

func (q *Query[T]) FindRaw(ctx context.Context) ([]bson.M, error) {
	// add projection of fields to be populated
	q.AddPopulateProjection()

	cursor, err := q.model.Collection.Find(ctx, q.filter, q.opts)
	if err != nil {
		fmt.Println("error in find", err.Error())
		return nil, err
	}
	defer cursor.Close(ctx)

	var results = make([]bson.M, 0)
	if err := cursor.All(ctx, &results); err != nil {
		fmt.Println("error in cursor all", err.Error())
		return nil, err
	}

	q.PopulateResult(&results)

	return results, nil
}

func (q *Query[T]) Find(ctx context.Context) ([]T, error) {

	rawResult, err := q.FindRaw(ctx)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := json.Marshal(rawResult)

	if err != nil {
		fmt.Println("error in marshalling", err.Error())
		return nil, err
	}

	var results []T

	if err := json.Unmarshal(jsonBytes, &results); err != nil {
		return nil, err
	}

	return results, nil

}
