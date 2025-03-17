package smongo

import (
	"context"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetFieldNameByTag(tag string, model any) string {
	t := reflect.TypeOf(model)

	for i := range t.NumField() {
		field := t.Field(i)

		// check if the field  name is json tag
		splitted := strings.Split(field.Tag.Get("json"), ",")

		if len(splitted) > 0 && splitted[0] == tag {
			return field.Name
		}

		// check if the field name is bson tag
		bsonSplitted := strings.Split(field.Tag.Get("bson"), ",")

		if len(bsonSplitted) > 0 && bsonSplitted[0] == tag {
			return field.Name
		}

		// check if the field name is tag
		if field.Name == tag {
			return field.Name
		}

	}

	return ""
}

func StrToProjection(projection bson.M, fields ...string) bson.M {

	// create a map of fields
	fieldMap := make(map[string]bool)

	for _, field := range fields {
		splitted := strings.Split(field, " ")

		for _, f := range splitted {
			trimmedVal := strings.TrimSpace(f)

			// check if first character is a minus sign
			if trimmedVal[0] == '-' {
				fieldMap[trimmedVal[1:]] = false
			} else {
				fieldMap[trimmedVal] = true
			}

		}

	}

	for key, val := range fieldMap {
		if val {
			projection[key] = 1
		} else {
			projection[key] = 0
		}

	}

	return projection

}

func GetPopulatedResults(filters []any, collection *mongo.Collection, details PopulateFieldQuery) (map[any]any, error) {
	result := make(map[any]any)

	filter := bson.M{details.ForeignField: bson.M{"$in": filters}}

	cursor, err := collection.Find(context.Background(), filter, details.Opts)
	if err != nil {
		return result, err
	}
	defer cursor.Close(context.Background())

	var results []bson.M
	if err := cursor.All(context.Background(), &results); err != nil {
		return result, err
	}

	for _, res := range results {

		key := res[details.ForeignField]

		if details.IsList {
			if _, ok := result[key]; !ok {
				result[key] = make([]bson.M, 0)
			}

			result[key] = append(result[key].([]bson.M), res)
		} else {
			result[key] = res
		}
	}

	return result, nil

}

func GetPopulatedResults1(filters []any, collection *mongo.Collection, details PopulateFieldQuery) ([]bson.M, error) {
	result := make([]bson.M, 0)

	filter := bson.M{details.ForeignField: bson.M{"$in": filters}}

	cursor, err := collection.Find(context.Background(), filter, details.Opts)
	if err != nil {
		return result, err
	}
	defer cursor.Close(context.Background())

	// var results []bson.M
	if err := cursor.All(context.Background(), &result); err != nil {
		return result, err
	}

	return result, nil

}

func IsValidPopulateField(populateField *PopulateFieldQuery) bool {
	if populateField == nil {
		return false
	}

	populateField.From = strings.TrimSpace(populateField.From)
	populateField.LocalField = strings.TrimSpace(populateField.LocalField)
	populateField.ForeignField = strings.TrimSpace(populateField.ForeignField)
	populateField.As = strings.TrimSpace(populateField.As)

	if populateField.From == "" || populateField.LocalField == "" || populateField.ForeignField == "" || populateField.As == "" {
		return false
	}

	for _, pop := range populateField.Populates {
		if !IsValidPopulateField(&pop) {
			return false
		}
	}

	return true

}
