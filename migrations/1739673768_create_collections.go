package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Up1739673768(ctx context.Context, db *mongo.Database) error {
	conversationsValidator := bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"participants", "created_at", "updated_at"},
			"properties": bson.M{
				"participants": bson.M{
					"bsonType": "array",
					"items": bson.M{
						"bsonType": "object",
						"required": []string{"participant_id"},
						"properties": bson.M{
							"participant_id": bson.M{
								"bsonType": "string",
							},
							"metadata": bson.M{
								"anyOf": []bson.M{
									{"bsonType": "object"},
									{"bsonType": "null"},
								},
							},
							"deleted_at": bson.M{
								"anyOf": []bson.M{
									{"bsonType": "date"},
									{"bsonType": "null"},
								},
							},
						},
					},
				},
				"last_message": bson.M{
					"bsonType": "object",
					"properties": bson.M{
						"sender": bson.M{
							"bsonType": "object",
							"required": []string{"participant_id"},
							"properties": bson.M{
								"participant_id": bson.M{
									"bsonType": "string",
								},
								"metadata": bson.M{
									"anyOf": []bson.M{
										{"bsonType": "object"},
										{"bsonType": "null"},
									},
								},
							},
						},
						"content": bson.M{
							"bsonType": "string",
						},
					},
				},
				"metadata": bson.M{
					"anyOf": []bson.M{
						{"bsonType": "object"},
						{"bsonType": "null"},
					},
				},
				"created_at": bson.M{
					"bsonType": "date",
				},
				"updated_at": bson.M{
					"bsonType": "date",
				},
			},
		},
	}

	if err := db.CreateCollection(ctx, "conversations", options.CreateCollection().SetValidator(conversationsValidator)); err != nil {
		return err
	}

	db.Collection("conversations").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "participants.participant_id", Value: 1},
			{Key: "updated_at", Value: -1},
		},
	})

	messagesValidator := bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"conversation_id", "sender", "kind", "content", "created_at"},
			"properties": bson.M{
				"conversation_id": bson.M{
					"bsonType": "string",
				},
				"sender": bson.M{
					"bsonType": "object",
					"required": []string{"participant_id"},
					"properties": bson.M{
						"participant_id": bson.M{
							"bsonType": "string",
						},
						"metadata": bson.M{
							"anyOf": []bson.M{
								{"bsonType": "object"},
								{"bsonType": "null"},
							},
						},
					},
				},
				"kind": bson.M{
					"enum": []string{"general", "system"},
				},
				"content": bson.M{
					"bsonType": "string",
				},
				"reactions": bson.M{
					"bsonType": "array",
					"items": bson.M{
						"bsonType": "object",
						"required": []string{"participants", "emoji"},
						"properties": bson.M{
							"emoji": bson.M{
								"bsonType": "string",
							},
							"participants": bson.M{
								"bsonType": "array",
								"items": bson.M{
									"bsonType": "object",
									"properties": bson.M{
										"participant_id": bson.M{
											"bsonType": "string",
										},
										"metadata": bson.M{
											"anyOf": []bson.M{
												{"bsonType": "object"},
												{"bsonType": "null"},
											},
										},
									},
								},
							},
						},
					},
				},
				"created_at": bson.M{
					"bsonType": "date",
				},
			},
		},
	}

	if err := db.CreateCollection(ctx, "messages", options.CreateCollection().SetValidator(messagesValidator)); err != nil {
		return err
	}

	db.Collection("messages").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "conversation_id", Value: 1},
			{Key: "created_at", Value: -1},
		},
	})

	return nil
}

func Down1739673768(ctx context.Context, db *mongo.Database) error {
	if err := db.Collection("messages").Drop(ctx); err != nil {
		return err
	}

	if err := db.Collection("conversations").Drop(ctx); err != nil {
		return err
	}

	return nil
}
