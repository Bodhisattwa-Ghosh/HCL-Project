package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/domain"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrNotFound      = errors.New("record not found")
	ErrDuplicate     = errors.New("duplicate record")
	ErrNotAuthorized = errors.New("not authorized")
)

// Store is the MongoDB source of truth for accounts, polls and the durable
// vote audit trail. Redis only owns the hot, realtime projection.
type Store struct {
	users *mongo.Collection
	polls *mongo.Collection
	votes *mongo.Collection
}

func New(client *mongo.Client, database string) *Store {
	db := client.Database(database)
	return &Store{
		users: db.Collection("users"),
		polls: db.Collection("polls"),
		votes: db.Collection("votes"),
	}
}

func (s *Store) EnsureIndexes(ctx context.Context) error {
	indexes := []struct {
		collection *mongo.Collection
		model      mongo.IndexModel
	}{
		{s.users, mongo.IndexModel{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true).SetName("unique_user_email")}},
		{s.polls, mongo.IndexModel{Keys: bson.D{{Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true).SetName("unique_poll_slug")}},
		{s.polls, mongo.IndexModel{Keys: bson.D{{Key: "ownerId", Value: 1}, {Key: "createdAt", Value: -1}}, Options: options.Index().SetName("owner_polls")}},
		{s.votes, mongo.IndexModel{Keys: bson.D{{Key: "pollId", Value: 1}, {Key: "voterId", Value: 1}}, Options: options.Index().SetUnique(true).SetName("one_vote_per_browser")}},
		{s.votes, mongo.IndexModel{Keys: bson.D{{Key: "pollId", Value: 1}, {Key: "optionId", Value: 1}}, Options: options.Index().SetName("poll_vote_totals")}},
	}
	for _, item := range indexes {
		if _, err := item.collection.Indexes().CreateOne(ctx, item.model); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	result, err := s.users.InsertOne(ctx, user)
	if mongo.IsDuplicateKeyError(err) {
		return domain.User{}, ErrDuplicate
	}
	if err != nil {
		return domain.User{}, err
	}
	user.ID = result.InsertedID.(primitive.ObjectID)
	return user, nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User
	err := s.users.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.User{}, ErrNotFound
	}
	return user, err
}

func (s *Store) UserByID(ctx context.Context, id primitive.ObjectID) (domain.User, error) {
	var user domain.User
	err := s.users.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.User{}, ErrNotFound
	}
	return user, err
}

func (s *Store) CreatePoll(ctx context.Context, poll domain.Poll) (domain.Poll, error) {
	result, err := s.polls.InsertOne(ctx, poll)
	if mongo.IsDuplicateKeyError(err) {
		return domain.Poll{}, ErrDuplicate
	}
	if err != nil {
		return domain.Poll{}, err
	}
	poll.ID = result.InsertedID.(primitive.ObjectID)
	return poll, nil
}

func (s *Store) PollBySlug(ctx context.Context, slug string) (domain.Poll, error) {
	var poll domain.Poll
	err := s.polls.FindOne(ctx, bson.M{"slug": slug}).Decode(&poll)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.Poll{}, ErrNotFound
	}
	return poll, err
}

func (s *Store) PollsByOwner(ctx context.Context, ownerID primitive.ObjectID) ([]domain.Poll, error) {
	cursor, err := s.polls.Find(ctx, bson.M{"ownerId": ownerID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var polls []domain.Poll
	if err := cursor.All(ctx, &polls); err != nil {
		return nil, err
	}
	return polls, nil
}

func (s *Store) InsertVote(ctx context.Context, vote domain.Vote) error {
	_, err := s.votes.InsertOne(ctx, vote)
	if mongo.IsDuplicateKeyError(err) {
		return ErrDuplicate
	}
	return err
}

// VoteCounts is only used to rebuild a missing Redis projection. It is never
// on the normal hot path after a poll has been initialized.
func (s *Store) VoteCounts(ctx context.Context, pollID primitive.ObjectID) (map[string]int64, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{{Key: "pollId", Value: pollID}}}},
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$optionId"}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
	}
	cursor, err := s.votes.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	counts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		counts[result.ID] = result.Count
	}
	return counts, cursor.Err()
}

func (s *Store) ClosePoll(ctx context.Context, slug string, ownerID primitive.ObjectID) (domain.Poll, error) {
	now := time.Now().UTC()
	result := s.polls.FindOneAndUpdate(
		ctx,
		bson.M{"slug": slug, "ownerId": ownerID, "closedAt": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"closedAt": now}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var poll domain.Poll
	err := result.Decode(&poll)
	if errors.Is(err, mongo.ErrNoDocuments) {
		// Distinguish a missing/foreign poll from an already-closed poll without
		// exposing existence to unauthenticated users (this route is authenticated).
		original, lookupErr := s.PollBySlug(ctx, slug)
		if lookupErr != nil {
			return domain.Poll{}, lookupErr
		}
		if original.OwnerID != ownerID {
			return domain.Poll{}, ErrNotAuthorized
		}
		return original, nil
	}
	return poll, err
}

func NewPoll(ownerID primitive.ObjectID, question string, labels []string, closesAt *time.Time) domain.Poll {
	options := make([]domain.Option, 0, len(labels))
	for _, label := range labels {
		options = append(options, domain.Option{ID: uuid.NewString(), Label: label})
	}
	return domain.Poll{
		OwnerID:   ownerID,
		Slug:      uuid.NewString(),
		Question:  question,
		Options:   options,
		CreatedAt: time.Now().UTC(),
		ClosesAt:  closesAt,
	}
}
