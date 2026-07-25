package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cindle0826/line-bot-ledger/internal/ledger"
)

// FirestoreStore stores entries under users/{lineUserID}/entries/{entryID},
// matching the data model in docs/architecture.html.
type FirestoreStore struct {
	client *firestore.Client
}

// NewFirestoreStore opens a Firestore client for the given GCP project.
func NewFirestoreStore(ctx context.Context, projectID string) (*FirestoreStore, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("store: open firestore client: %w", err)
	}
	return &FirestoreStore{client: client}, nil
}

func (s *FirestoreStore) AddEntry(ctx context.Context, lineUserID string, entry ledger.Entry) error {
	_, _, err := s.client.Collection("users").Doc(lineUserID).Collection("entries").Add(ctx, entry)
	if err != nil {
		return fmt.Errorf("store: add entry: %w", err)
	}
	return nil
}

func (s *FirestoreStore) ListEntries(ctx context.Context, lineUserID string, from, to time.Time) ([]ledger.Entry, error) {
	iter := s.client.Collection("users").Doc(lineUserID).Collection("entries").
		Where("createdAt", ">=", from).
		Where("createdAt", "<", to).
		OrderBy("createdAt", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	var entries []ledger.Entry
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("store: list entries: %w", err)
		}
		var e ledger.Entry
		if err := doc.DataTo(&e); err != nil {
			return nil, fmt.Errorf("store: decode entry: %w", err)
		}
		e.ID = doc.Ref.ID
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *FirestoreStore) UpdateEntry(ctx context.Context, lineUserID, entryID string, entry ledger.Entry) error {
	_, err := s.client.Collection("users").Doc(lineUserID).Collection("entries").Doc(entryID).Update(ctx, []firestore.Update{
		{Path: "amount", Value: entry.Amount},
		{Path: "category", Value: entry.Category},
		{Path: "note", Value: entry.Note},
	})
	if err != nil {
		return fmt.Errorf("store: update entry: %w", err)
	}
	return nil
}

func (s *FirestoreStore) DeleteEntry(ctx context.Context, lineUserID, entryID string) error {
	_, err := s.client.Collection("users").Doc(lineUserID).Collection("entries").Doc(entryID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("store: delete entry: %w", err)
	}
	return nil
}

// userDoc is users/{lineUserID} itself — previously never written (an
// implicit path container for the entries subcollection), now holding each
// user's subscription settings.
type userDoc struct {
	Subscriptions []ledger.Frequency `firestore:"subscriptions"`
}

func (s *FirestoreStore) ListSubscribers(ctx context.Context, frequency ledger.Frequency) ([]string, error) {
	iter := s.client.Collection("users").
		Where("subscriptions", "array-contains", frequency).
		Documents(ctx)
	defer iter.Stop()

	var userIDs []string
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("store: list subscribers: %w", err)
		}
		userIDs = append(userIDs, doc.Ref.ID)
	}
	return userIDs, nil
}

func (s *FirestoreStore) GetSubscriptions(ctx context.Context, lineUserID string) ([]ledger.Frequency, error) {
	doc, err := s.client.Collection("users").Doc(lineUserID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get subscriptions: %w", err)
	}
	var u userDoc
	if err := doc.DataTo(&u); err != nil {
		return nil, fmt.Errorf("store: decode subscriptions: %w", err)
	}
	return u.Subscriptions, nil
}

func (s *FirestoreStore) SetSubscriptions(ctx context.Context, lineUserID string, frequencies []ledger.Frequency) error {
	// MergeAll requires map data, not a struct — Merge with an explicit
	// field path works with either, and here we only ever want to touch
	// the one field anyway.
	_, err := s.client.Collection("users").Doc(lineUserID).Set(ctx, userDoc{Subscriptions: frequencies}, firestore.Merge([]string{"subscriptions"}))
	if err != nil {
		return fmt.Errorf("store: set subscriptions: %w", err)
	}
	return nil
}

func (s *FirestoreStore) Close() error {
	return s.client.Close()
}
