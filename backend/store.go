package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"unicode/utf8"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// statsStore keeps the HTTP API independent of the cloud database provider.
type statsStore interface {
	ListUserIDs(context.Context, int, string) ([]string, string, error)
	ListStats(context.Context, int, string) ([]UserStats, string, error)
	GetStats(context.Context, []string) ([]UserStats, error)
	GetStat(context.Context, string) (UserStats, bool, error)
}

func newStatsStore(ctx context.Context) (statsStore, func() error, error) {
	return newFirestoreStore(ctx, gcpProjectID(), firestoreCollection())
}

func firestoreCollection() string {
	if collection := os.Getenv("FIRESTORE_COLLECTION"); collection != "" {
		return collection
	}
	return "userStats"
}

// A cursor is a user ID encoded for safe use in a URL. Stores sort by user ID.
func scanPage(c *gin.Context) (int, string, bool) {
	limit := 100
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
			return 0, "", false
		}
		if parsedLimit > 250 {
			parsedLimit = 250
		}
		limit = parsedLimit
	}
	if cursor := c.Query("cursor"); cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || len(decoded) == 0 || !utf8.Valid(decoded) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor"})
			return 0, "", false
		}
		return limit, string(decoded), true
	}
	return limit, "", true
}

func pageCursor(lastID string) string {
	if lastID == "" {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(lastID))
}

type firestoreStore struct {
	client     *firestore.Client
	collection *firestore.CollectionRef
}

func gcpProjectID() string {
	return os.Getenv("GCP_PROJECT_ID")
}

func newFirestoreStore(ctx context.Context, projectID, collection string) (statsStore, func() error, error) {
	if projectID == "" {
		return nil, nil, fmt.Errorf("GCP_PROJECT_ID is required")
	}
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("create Firestore client: %w", err)
	}
	store := &firestoreStore{client: client, collection: client.Collection(collection)}
	return store, client.Close, nil
}

func (s *firestoreStore) ListUserIDs(ctx context.Context, limit int, cursor string) ([]string, string, error) {
	records, next, err := s.list(ctx, limit, cursor)
	if err != nil {
		return nil, "", err
	}
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.UserID)
	}
	return ids, next, nil
}

func (s *firestoreStore) ListStats(ctx context.Context, limit int, cursor string) ([]UserStats, string, error) {
	return s.list(ctx, limit, cursor)
}

func (s *firestoreStore) list(ctx context.Context, limit int, cursor string) ([]UserStats, string, error) {
	query := s.collection.OrderBy("userId", firestore.Asc).Limit(limit + 1)
	if cursor != "" {
		query = query.StartAfter(cursor)
	}
	documents, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, "", err
	}
	hasNext := len(documents) > limit
	if hasNext {
		documents = documents[:limit]
	}
	records := make([]UserStats, 0, len(documents))
	for _, document := range documents {
		var record UserStats
		if err := document.DataTo(&record); err != nil {
			return nil, "", fmt.Errorf("decode Firestore user statistics: %w", err)
		}
		records = append(records, record)
	}
	if hasNext && len(records) > 0 {
		return records, pageCursor(records[len(records)-1].UserID), nil
	}
	return records, "", nil
}

func (s *firestoreStore) GetStats(ctx context.Context, ids []string) ([]UserStats, error) {
	references := make([]*firestore.DocumentRef, 0, len(ids))
	for _, id := range ids {
		references = append(references, s.collection.Doc(id))
	}
	documents, err := s.client.GetAll(ctx, references)
	if err != nil {
		return nil, err
	}
	records := make([]UserStats, 0, len(documents))
	for _, document := range documents {
		if !document.Exists() {
			continue
		}
		var record UserStats
		if err := document.DataTo(&record); err != nil {
			return nil, fmt.Errorf("decode Firestore user statistics: %w", err)
		}
		records = append(records, record)
	}
	return records, nil
}

func (s *firestoreStore) GetStat(ctx context.Context, id string) (UserStats, bool, error) {
	document, err := s.collection.Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return UserStats{}, false, nil
	}
	if err != nil {
		return UserStats{}, false, err
	}
	var record UserStats
	if err := document.DataTo(&record); err != nil {
		return UserStats{}, false, fmt.Errorf("decode Firestore user statistics: %w", err)
	}
	return record, true, nil
}
