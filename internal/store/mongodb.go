package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"media-crawler-go/internal/config"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	mongoOnce sync.Once
	mongoCli  *mongo.Client
	mongoErr  error
)

func mongoURI() string {
	return strings.TrimSpace(config.AppConfig.MongoURI)
}

func mongoDBName() string {
	v := strings.TrimSpace(config.AppConfig.MongoDB)
	if v == "" {
		return "media_crawler"
	}
	return v
}

func mongoClient() (*mongo.Client, error) {
	if backendKind() != backendMongoDB {
		return nil, errors.New("mongodb backend disabled")
	}
	mongoOnce.Do(func() {
		uri := mongoURI()
		if uri == "" {
			mongoErr = errors.New("MONGO_URI is empty")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cli, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err != nil {
			mongoErr = err
			return
		}
		if err := cli.Ping(ctx, readpref.Primary()); err != nil {
			_ = cli.Disconnect(ctx)
			mongoErr = err
			return
		}
		if err := initMongoSchema(ctx, cli); err != nil {
			_ = cli.Disconnect(ctx)
			mongoErr = err
			return
		}
		mongoCli = cli
	})
	return mongoCli, mongoErr
}

func initMongoSchema(ctx context.Context, cli *mongo.Client) error {
	db := cli.Database(mongoDBName())

	notes := db.Collection("notes")
	_, err := notes.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "platform", Value: 1}, {Key: "note_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_platform_note"),
		},
	})
	if err != nil {
		return fmt.Errorf("mongo create indexes notes: %w", err)
	}

	creators := db.Collection("creators")
	_, err = creators.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "platform", Value: 1}, {Key: "creator_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_platform_creator"),
		},
	})
	if err != nil {
		return fmt.Errorf("mongo create indexes creators: %w", err)
	}

	comments := db.Collection("comments")
	_, err = comments.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "platform", Value: 1}, {Key: "comment_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_platform_comment"),
		},
		{
			Keys:    bson.D{{Key: "platform", Value: 1}, {Key: "note_id", Value: 1}},
			Options: options.Index().SetName("idx_platform_note"),
		},
	})
	if err != nil {
		return fmt.Errorf("mongo create indexes comments: %w", err)
	}
	return nil
}

func mongoUpsertNote(noteID string, note any) error {
	cli, err := mongoClient()
	if err != nil {
		return err
	}
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return errors.New("note_id is empty")
	}
	b, err := json.Marshal(note)
	if err != nil {
		return err
	}
	platform := strings.TrimSpace(config.AppConfig.Platform)
	if platform == "" {
		platform = "xhs"
	}
	now := time.Now().Unix()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := cli.Database(mongoDBName()).Collection("notes")
	filter := bson.D{{Key: "platform", Value: platform}, {Key: "note_id", Value: noteID}}
	update := bson.D{{Key: "$set", Value: bson.M{
		"platform":    platform,
		"note_id":     noteID,
		"data_json":   string(b),
		"updated_at":  now,
		"updated_iso": time.Now().UTC().Format(time.RFC3339Nano),
	}}}
	_, err = coll.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func mongoUpsertCreator(creatorID string, data any) error {
	cli, err := mongoClient()
	if err != nil {
		return err
	}
	creatorID = strings.TrimSpace(creatorID)
	if creatorID == "" {
		return errors.New("creator_id is empty")
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	platform := strings.TrimSpace(config.AppConfig.Platform)
	if platform == "" {
		platform = "xhs"
	}
	now := time.Now().Unix()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := cli.Database(mongoDBName()).Collection("creators")
	filter := bson.D{{Key: "platform", Value: platform}, {Key: "creator_id", Value: creatorID}}
	update := bson.D{{Key: "$set", Value: bson.M{
		"platform":    platform,
		"creator_id":  creatorID,
		"data_json":   string(b),
		"updated_at":  now,
		"updated_iso": time.Now().UTC().Format(time.RFC3339Nano),
	}}}
	_, err = coll.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func mongoInsertComments(noteID string, items []any, keyFn func(any) (string, error)) error {
	cli, err := mongoClient()
	if err != nil {
		return err
	}
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return errors.New("note_id is empty")
	}
	if len(items) == 0 {
		return nil
	}
	platform := strings.TrimSpace(config.AppConfig.Platform)
	if platform == "" {
		platform = "xhs"
	}
	now := time.Now().Unix()

	models := make([]mongo.WriteModel, 0, len(items))
	for _, item := range items {
		id, err := keyFn(item)
		if err != nil {
			return err
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		b, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("marshal comment %s: %w", id, err)
		}
		filter := bson.D{{Key: "platform", Value: platform}, {Key: "comment_id", Value: id}}
		update := bson.D{{Key: "$setOnInsert", Value: bson.M{
			"platform":     platform,
			"comment_id":   id,
			"note_id":      noteID,
			"data_json":    string(b),
			"created_at":   now,
			"created_iso":  time.Now().UTC().Format(time.RFC3339Nano),
			"note_id_norm": noteID,
		}}}
		m := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models = append(models, m)
	}
	if len(models) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	coll := cli.Database(mongoDBName()).Collection("comments")
	_, err = coll.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
	return err
}

