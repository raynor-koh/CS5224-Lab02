package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"
)

// statsStore keeps the HTTP API independent of the cloud database provider.
type statsStore interface {
	ListUserIDs(context.Context, int, string) ([]string, string, error)
	ListStats(context.Context, int, string) ([]UserStats, string, error)
	GetStats(context.Context, []string) ([]UserStats, error)
	GetStat(context.Context, string) (UserStats, bool, error)
}

type dynamoDBStore struct {
	client *dynamodb.Client
	table  string
}

func dynamoDBTable() string {
	if table := os.Getenv("DYNAMODB_TABLE"); table != "" {
		return table
	}
	return "userStats"
}

func awsRegion() string {
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}
	return "ap-southeast-1"
}

func newStatsStore(ctx context.Context) (statsStore, func() error, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(awsRegion()))
	if err != nil {
		return nil, nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	endpoint := awsEndpointURL()
	client := dynamodb.NewFromConfig(cfg, func(options *dynamodb.Options) {
		if endpoint != "" {
			options.BaseEndpoint = aws.String(endpoint)
		}
	})
	return &dynamoDBStore{client: client, table: dynamoDBTable()}, func() error { return nil }, nil
}

func awsEndpointURL() string {
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		return endpoint
	}
	return os.Getenv("DYNAMODB_ENDPOINT_URL")
}

// scanPage validates shared list-query parameters. DynamoDB cursors are
// encoded primary keys rather than user-facing identifiers.
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

func (s *dynamoDBStore) ListUserIDs(ctx context.Context, limit int, cursor string) ([]string, string, error) {
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

func (s *dynamoDBStore) ListStats(ctx context.Context, limit int, cursor string) ([]UserStats, string, error) {
	return s.list(ctx, limit, cursor)
}

// DynamoDB Scan cursors are the table's primary key. A Scan has no ordering
// guarantee, so callers must treat list results and cursors as opaque.
func (s *dynamoDBStore) list(ctx context.Context, limit int, cursor string) ([]UserStats, string, error) {
	var startKey map[string]types.AttributeValue
	if cursor != "" {
		startKey = map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: cursor}}
	}
	output, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:         aws.String(s.table),
		Limit:             aws.Int32(int32(limit)),
		ExclusiveStartKey: startKey,
	})
	if err != nil {
		return nil, "", fmt.Errorf("scan DynamoDB table: %w", err)
	}
	records := make([]UserStats, 0, len(output.Items))
	for _, item := range output.Items {
		var record UserStats
		if err := attributevalue.UnmarshalMap(item, &record); err != nil {
			return nil, "", fmt.Errorf("decode DynamoDB user statistics: %w", err)
		}
		records = append(records, record)
	}
	if key, ok := output.LastEvaluatedKey["userId"].(*types.AttributeValueMemberS); ok {
		return records, pageCursor(key.Value), nil
	}
	return records, "", nil
}

func (s *dynamoDBStore) GetStats(ctx context.Context, ids []string) ([]UserStats, error) {
	keys := make([]map[string]types.AttributeValue, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: id}})
	}
	output, err := s.client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{s.table: {Keys: keys}},
	})
	if err != nil {
		return nil, fmt.Errorf("batch get DynamoDB user statistics: %w", err)
	}
	items := output.Responses[s.table]
	records := make([]UserStats, 0, len(items))
	for _, item := range items {
		var record UserStats
		if err := attributevalue.UnmarshalMap(item, &record); err != nil {
			return nil, fmt.Errorf("decode DynamoDB user statistics: %w", err)
		}
		records = append(records, record)
	}
	return records, nil
}

func (s *dynamoDBStore) GetStat(ctx context.Context, id string) (UserStats, bool, error) {
	output, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key:       map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: id}},
	})
	if err != nil {
		return UserStats{}, false, fmt.Errorf("get DynamoDB user statistic: %w", err)
	}
	if len(output.Item) == 0 {
		return UserStats{}, false, nil
	}
	var record UserStats
	if err := attributevalue.UnmarshalMap(output.Item, &record); err != nil {
		return UserStats{}, false, fmt.Errorf("decode DynamoDB user statistic: %w", err)
	}
	return record, true, nil
}
