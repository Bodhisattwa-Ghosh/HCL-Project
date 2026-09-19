package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/domain"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/realtime"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/repository"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/validation"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrPollNotFound        = errors.New("poll not found")
	ErrPollClosed          = errors.New("this poll is closed")
	ErrInvalidOption       = errors.New("that option does not belong to this poll")
	ErrAlreadyVoted        = errors.New("this browser has already voted on this poll")
	ErrRealtimeUnavailable = errors.New("live results are temporarily unavailable")
)

const counterReadyField = "__pulsepoll_ready"

// incrementIfReady is deliberately atomic. A Redis key can disappear between
// the pre-vote read and the post-Mongo write (for example after an eviction).
// Checking the projection marker and incrementing in one Lua operation avoids
// creating a partial counter that a cache rebuild could later overwrite.
var incrementIfReady = redis.NewScript(`
if redis.call('HGET', KEYS[1], ARGV[1]) == '1' then
  return redis.call('HINCRBY', KEYS[1], ARGV[2], 1)
end
return 0
`)

type PollService struct {
	store  *repository.Store
	redis  *redis.Client
	broker *realtime.Broker
	now    func() time.Time
}

func NewPolls(store *repository.Store, redisClient *redis.Client, broker *realtime.Broker) *PollService {
	return &PollService{store: store, redis: redisClient, broker: broker, now: func() time.Time { return time.Now().UTC() }}
}

func (s *PollService) Create(ctx context.Context, ownerID primitive.ObjectID, question string, options []string, closesAt *time.Time) (domain.PublicPoll, error) {
	question, options, err := validation.Poll(question, options)
	if err != nil {
		return domain.PublicPoll{}, err
	}
	if closesAt != nil {
		utc := closesAt.UTC()
		if utc.Before(s.now().Add(time.Minute)) || utc.After(s.now().AddDate(1, 0, 0)) {
			return domain.PublicPoll{}, errors.New("closing time must be between one minute and one year from now")
		}
		closesAt = &utc
	}

	var poll domain.Poll
	for attempt := 0; attempt < 3; attempt++ {
		poll, err = s.store.CreatePoll(ctx, repository.NewPoll(ownerID, question, options, closesAt))
		if !errors.Is(err, repository.ErrDuplicate) {
			break
		}
	}
	if err != nil {
		return domain.PublicPoll{}, err
	}

	// Mark the cache as initialized at zero. Any later Redis eviction can be
	// rebuilt from Mongo's immutable vote records by countsFor.
	fields := make(map[string]any, len(poll.Options)+1)
	fields[counterReadyField] = "1"
	for _, option := range poll.Options {
		fields[option.ID] = "0"
	}
	if err := s.redis.HSet(ctx, realtime.CounterKey(poll.ID), fields).Err(); err != nil {
		// The poll is still safe in Mongo. A later read will hydrate the cache,
		// so avoid reporting a false failure after persisting it.
		return s.toPublic(poll, map[string]int64{}, ownerID), nil
	}
	return s.toPublic(poll, map[string]int64{}, ownerID), nil
}

func (s *PollService) ListForOwner(ctx context.Context, ownerID primitive.ObjectID) ([]domain.PublicPoll, error) {
	polls, err := s.store.PollsByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.PublicPoll, 0, len(polls))
	for _, poll := range polls {
		counts, err := s.countsFor(ctx, poll)
		if err != nil {
			return nil, ErrRealtimeUnavailable
		}
		result = append(result, s.toPublic(poll, counts, ownerID))
	}
	return result, nil
}

func (s *PollService) Get(ctx context.Context, slug string, viewerID *primitive.ObjectID) (domain.PublicPoll, error) {
	poll, err := s.rawBySlug(ctx, slug)
	if err != nil {
		return domain.PublicPoll{}, err
	}
	counts, err := s.countsFor(ctx, poll)
	if err != nil {
		return domain.PublicPoll{}, ErrRealtimeUnavailable
	}
	var viewer primitive.ObjectID
	if viewerID != nil {
		viewer = *viewerID
	}
	return s.toPublic(poll, counts, viewer), nil
}

func (s *PollService) Vote(ctx context.Context, slug, optionID, voterID string) (domain.PublicPoll, error) {
	poll, err := s.rawBySlug(ctx, slug)
	if err != nil {
		return domain.PublicPoll{}, err
	}
	if isClosed(poll, s.now()) {
		return domain.PublicPoll{}, ErrPollClosed
	}
	if !hasOption(poll, optionID) {
		return domain.PublicPoll{}, ErrInvalidOption
	}
	if _, err := s.countsFor(ctx, poll); err != nil {
		return domain.PublicPoll{}, ErrRealtimeUnavailable
	}

	err = s.store.InsertVote(ctx, domain.Vote{
		PollID: poll.ID, OptionID: optionID, VoterID: voterID, CreatedAt: s.now(),
	})
	if errors.Is(err, repository.ErrDuplicate) {
		return domain.PublicPoll{}, ErrAlreadyVoted
	}
	if err != nil {
		return domain.PublicPoll{}, err
	}

	updated, err := incrementIfReady.Run(ctx, s.redis, []string{realtime.CounterKey(poll.ID)}, counterReadyField, optionID).Result()
	if err != nil {
		// Mongo has the durable vote. Returning an error tells the UI to refetch;
		// the next cache miss/eviction rebuilds the projection from Mongo.
		return domain.PublicPoll{}, ErrRealtimeUnavailable
	}
	updatedCount, isCount := updated.(int64)
	if !isCount || updatedCount == 0 {
		// The durable Mongo insert is already visible. Do not increment after a
		// rebuild: the rebuild's aggregate includes this exact vote. If another
		// request is rebuilding now, hydrateCounter waits for it and then performs
		// a fresh aggregate itself when needed.
		if err := s.hydrateCounter(ctx, poll); err != nil {
			return domain.PublicPoll{}, ErrRealtimeUnavailable
		}
	}
	counts, err := s.countsFor(ctx, poll)
	if err != nil {
		return domain.PublicPoll{}, ErrRealtimeUnavailable
	}
	result := s.toPublic(poll, counts, primitive.NilObjectID)
	// A response still succeeds if Pub/Sub briefly fails: the durable count is
	// correct and reconnecting SSE clients receive the latest snapshot.
	_ = s.broker.Publish(context.WithoutCancel(ctx), result)
	return result, nil
}

func (s *PollService) Close(ctx context.Context, slug string, ownerID primitive.ObjectID) (domain.PublicPoll, error) {
	poll, err := s.rawBySlug(ctx, slug)
	if err != nil {
		return domain.PublicPoll{}, err
	}
	if poll.OwnerID != ownerID {
		return domain.PublicPoll{}, repository.ErrNotAuthorized
	}
	poll, err = s.store.ClosePoll(ctx, slug, ownerID)
	if err != nil {
		return domain.PublicPoll{}, err
	}
	counts, err := s.countsFor(ctx, poll)
	if err != nil {
		return domain.PublicPoll{}, ErrRealtimeUnavailable
	}
	result := s.toPublic(poll, counts, ownerID)
	_ = s.broker.Publish(context.WithoutCancel(ctx), result)
	return result, nil
}

func (s *PollService) rawBySlug(ctx context.Context, slug string) (domain.Poll, error) {
	if _, err := uuid.Parse(slug); err != nil {
		return domain.Poll{}, ErrPollNotFound
	}
	poll, err := s.store.PollBySlug(ctx, slug)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.Poll{}, ErrPollNotFound
	}
	return poll, err
}

func (s *PollService) countsFor(ctx context.Context, poll domain.Poll) (map[string]int64, error) {
	key := realtime.CounterKey(poll.ID)
	values, err := s.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if values[counterReadyField] != "1" {
		if err := s.hydrateCounter(ctx, poll); err != nil {
			return nil, err
		}
		values, err = s.redis.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
	}
	counts := make(map[string]int64, len(poll.Options))
	for _, option := range poll.Options {
		if raw, exists := values[option.ID]; exists {
			count, parseErr := strconv.ParseInt(raw, 10, 64)
			if parseErr != nil || count < 0 {
				return nil, fmt.Errorf("invalid Redis vote count")
			}
			counts[option.ID] = count
		}
	}
	return counts, nil
}

// hydrateCounter makes Redis a recoverable realtime projection. A short Redis
// lock prevents two API instances from overwriting a just-incremented counter
// during cache warm-up.
func (s *PollService) hydrateCounter(ctx context.Context, poll domain.Poll) error {
	key := realtime.CounterKey(poll.ID)
	lockKey := key + ":hydrate-lock"
	acquired, err := s.redis.SetNX(ctx, lockKey, "1", 5*time.Second).Result()
	if err != nil {
		return err
	}
	if acquired {
		defer s.redis.Del(context.WithoutCancel(ctx), lockKey)
		counts, err := s.store.VoteCounts(ctx, poll.ID)
		if err != nil {
			return err
		}
		fields := make(map[string]any, len(poll.Options)+1)
		fields[counterReadyField] = "1"
		for _, option := range poll.Options {
			fields[option.ID] = strconv.FormatInt(counts[option.ID], 10)
		}
		return s.redis.HSet(ctx, key, fields).Err()
	}

	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("timed out waiting for Redis counter hydration")
		case <-ticker.C:
			ready, getErr := s.redis.HGet(ctx, key, counterReadyField).Result()
			if getErr == nil && ready == "1" {
				return nil
			}
			if getErr != nil && getErr != redis.Nil {
				return getErr
			}
		}
	}
}

func (s *PollService) toPublic(poll domain.Poll, counts map[string]int64, viewerID primitive.ObjectID) domain.PublicPoll {
	options := make([]domain.PublicOption, 0, len(poll.Options))
	var total int64
	for _, option := range poll.Options {
		votes := counts[option.ID]
		options = append(options, domain.PublicOption{ID: option.ID, Label: option.Label, Votes: votes})
		total += votes
	}
	return domain.PublicPoll{
		ID:         poll.ID.Hex(),
		Slug:       poll.Slug,
		Question:   poll.Question,
		Options:    options,
		TotalVotes: total,
		CreatedAt:  poll.CreatedAt,
		ClosesAt:   poll.ClosesAt,
		IsClosed:   isClosed(poll, s.now()),
		IsOwner:    viewerID != primitive.NilObjectID && poll.OwnerID == viewerID,
	}
}

func hasOption(poll domain.Poll, optionID string) bool {
	for _, option := range poll.Options {
		if option.ID == optionID {
			return true
		}
	}
	return false
}

func isClosed(poll domain.Poll, now time.Time) bool {
	return poll.ClosedAt != nil || (poll.ClosesAt != nil && !poll.ClosesAt.After(now))
}
