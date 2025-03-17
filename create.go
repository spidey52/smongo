package smongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (m *Model[T]) Create(ctx context.Context, doc T) (primitive.ObjectID, error) {
	res, err := m.Collection.InsertOne(ctx, doc)
	if err != nil {
		return primitive.NilObjectID, err
	}
	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("failed to convert ID")
	}
	return id, nil
}

func (m *Model[T]) UpdateByID(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	filter := bson.M{"_id": id}
	updateDoc := bson.M{"$set": update}
	_, err := m.Collection.UpdateOne(ctx, filter, updateDoc)
	return err
}

func (m *Model[T]) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	_, err := m.Collection.DeleteOne(ctx, filter)
	return err
}
