package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Name         string             `bson:"name"`
	Email        string             `bson:"email"`
	PasswordHash string             `bson:"passwordHash"`
	CreatedAt    time.Time          `bson:"createdAt"`
}

type Option struct {
	ID    string `bson:"id" json:"id"`
	Label string `bson:"label" json:"label"`
}

type Poll struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	OwnerID   primitive.ObjectID `bson:"ownerId"`
	Slug      string             `bson:"slug"`
	Question  string             `bson:"question"`
	Options   []Option           `bson:"options"`
	CreatedAt time.Time          `bson:"createdAt"`
	ClosesAt  *time.Time         `bson:"closesAt,omitempty"`
	ClosedAt  *time.Time         `bson:"closedAt,omitempty"`
}

type Vote struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	PollID    primitive.ObjectID `bson:"pollId"`
	OptionID  string             `bson:"optionId"`
	VoterID   string             `bson:"voterId"`
	CreatedAt time.Time          `bson:"createdAt"`
}

type PublicUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type PublicOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Votes int64  `json:"votes"`
}

type PublicPoll struct {
	ID         string         `json:"id"`
	Slug       string         `json:"slug"`
	Question   string         `json:"question"`
	Options    []PublicOption `json:"options"`
	TotalVotes int64          `json:"totalVotes"`
	CreatedAt  time.Time      `json:"createdAt"`
	ClosesAt   *time.Time     `json:"closesAt"`
	IsClosed   bool           `json:"isClosed"`
	IsOwner    bool           `json:"isOwner"`
}

func (u User) Public() PublicUser {
	return PublicUser{ID: u.ID.Hex(), Name: u.Name, Email: u.Email}
}
