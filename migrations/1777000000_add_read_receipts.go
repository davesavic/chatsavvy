package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Up1777000000(ctx context.Context, db *mongo.Database) error {
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
							"last_read_message_id": bson.M{
								"anyOf": []bson.M{
									{"bsonType": "objectId"},
									{"bsonType": "null"},
								},
							},
							"last_read_at": bson.M{
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

	return db.RunCommand(ctx, bson.D{
		{Key: "collMod", Value: "conversations"},
		{Key: "validator", Value: conversationsValidator},
	}).Err()
}

func Down1777000000(ctx context.Context, db *mongo.Database) error {
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

	return db.RunCommand(ctx, bson.D{
		{Key: "collMod", Value: "conversations"},
		{Key: "validator", Value: conversationsValidator},
	}).Err()
}
