package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Up1774000000(ctx context.Context, db *mongo.Database) error {
	// Remove enum constraint on message kind, replace with bsonType string
	messagesValidator := bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"conversation_id", "sender", "kind", "created_at"},
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
					"bsonType": "string",
				},
				"content": bson.M{
					"bsonType": "string",
				},
				"attachments": bson.M{
					"anyOf": []bson.M{
						{
							"bsonType": "array",
							"items": bson.M{
								"bsonType": "object",
								"required": []string{"kind"},
								"properties": bson.M{
									"kind": bson.M{
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
						{"bsonType": "null"},
					},
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

	return db.RunCommand(ctx, bson.D{
		{Key: "collMod", Value: "messages"},
		{Key: "validator", Value: messagesValidator},
	}).Err()
}

func Down1774000000(ctx context.Context, db *mongo.Database) error {
	// Restore enum constraint on message kind
	messagesValidator := bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"conversation_id", "sender", "kind", "created_at"},
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
				"attachments": bson.M{
					"anyOf": []bson.M{
						{
							"bsonType": "array",
							"items": bson.M{
								"bsonType": "object",
								"required": []string{"kind"},
								"properties": bson.M{
									"kind": bson.M{
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
						{"bsonType": "null"},
					},
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

	return db.RunCommand(ctx, bson.D{
		{Key: "collMod", Value: "messages"},
		{Key: "validator", Value: messagesValidator},
	}).Err()
}
