package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Up1770967803(ctx context.Context, db *mongo.Database) error {
	// Add attachments to messages collection validator
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

	err := db.RunCommand(ctx, bson.D{
		{Key: "collMod", Value: "messages"},
		{Key: "validator", Value: messagesValidator},
	}).Err()
	if err != nil {
		return err
	}

	// Add attachments to conversations last_message validator
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

func Down1770967803(ctx context.Context, db *mongo.Database) error {
	// Revert messages validator (remove attachments)
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

	err := db.RunCommand(ctx, bson.D{
		{Key: "collMod", Value: "messages"},
		{Key: "validator", Value: messagesValidator},
	}).Err()
	if err != nil {
		return err
	}

	// Revert conversations validator (remove attachments from last_message)
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

	return db.RunCommand(ctx, bson.D{
		{Key: "collMod", Value: "conversations"},
		{Key: "validator", Value: conversationsValidator},
	}).Err()
}
