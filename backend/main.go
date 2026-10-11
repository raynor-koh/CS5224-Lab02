package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type UserStats struct {
	UserID    string `json:"userId" dynamodbav:"userId"`
	Followers int64  `json:"followers" dynamodbav:"followers"`
	Followees int64  `json:"followees" dynamodbav:"followees"`
}

type userIDRecord struct {
	UserID string `dynamodbav:"userId"`
}

func main() {
	// Support starting the backend from either the repository root or backend/.
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Load ../.env: %v", err)
	}

	ctx := context.Background()

	region := os.Getenv("AWS_DEFAULT_REGION")
	if region == "" {
		region = "us-east-1"
	}

	awsConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		log.Fatalf("load AWS configuration: %v", err)
	}

	dynamoOptions := []func(*dynamodb.Options){}
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		dynamoOptions = append(dynamoOptions, func(options *dynamodb.Options) {
			options.BaseEndpoint = aws.String(endpoint)
		})
	}
	db := dynamodb.NewFromConfig(awsConfig, dynamoOptions...)

	tableName := os.Getenv("DYNAMODB_TABLE")
	if tableName == "" {
		tableName = "UserStats"
	}

	router := gin.Default()
	router.Use(localFrontendCORS())
	router.OPTIONS("/*path", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	router.GET("/api/users", func(c *gin.Context) {
		limit := int64(100)
		if rawLimit := c.Query("limit"); rawLimit != "" {
			parsedLimit, err := strconv.ParseInt(rawLimit, 10, 32)
			if err != nil || parsedLimit < 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
				return
			}
			if parsedLimit > 250 {
				parsedLimit = 250
			}
			limit = parsedLimit
		}

		input := &dynamodb.ScanInput{
			TableName:            aws.String(tableName),
			Limit:                aws.Int32(int32(limit)),
			ProjectionExpression: aws.String("#userId"),
			ExpressionAttributeNames: map[string]string{
				"#userId": "userId",
			},
		}
		if cursor := c.Query("cursor"); cursor != "" {
			decodedCursor, err := base64.RawURLEncoding.DecodeString(cursor)
			if err != nil || len(decodedCursor) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor"})
				return
			}
			input.ExclusiveStartKey = map[string]types.AttributeValue{
				"userId": &types.AttributeValueMemberS{Value: string(decodedCursor)},
			}
		}

		result, err := db.Scan(c.Request.Context(), input)
		if err != nil {
			log.Printf("list user IDs: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
			return
		}

		var records []userIDRecord
		if err := attributevalue.UnmarshalListOfMaps(result.Items, &records); err != nil {
			log.Printf("decode user IDs: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "stored user IDs are invalid"})
			return
		}
		userIDs := make([]string, 0, len(records))
		for _, record := range records {
			userIDs = append(userIDs, record.UserID)
		}

		nextCursor := ""
		if len(result.LastEvaluatedKey) > 0 {
			if lastID, ok := result.LastEvaluatedKey["userId"].(*types.AttributeValueMemberS); ok {
				nextCursor = base64.RawURLEncoding.EncodeToString([]byte(lastID.Value))
			}
		}
		c.JSON(http.StatusOK, gin.H{"userIds": userIDs, "nextCursor": nextCursor})
	})

	router.POST("/api/users/stats", func(c *gin.Context) {
		var request struct {
			UserIDs []string `json:"userIds" binding:"required,min=1,max=100,dive,required"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "provide between 1 and 100 user IDs"})
			return
		}

		keys := make([]map[string]types.AttributeValue, 0, len(request.UserIDs))
		uniqueIDs := make([]string, 0, len(request.UserIDs))
		seen := make(map[string]struct{}, len(request.UserIDs))
		for _, userID := range request.UserIDs {
			userID = strings.TrimSpace(userID)
			if userID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "user IDs cannot be empty"})
				return
			}
			if _, exists := seen[userID]; exists {
				continue
			}
			seen[userID] = struct{}{}
			uniqueIDs = append(uniqueIDs, userID)
			keys = append(keys, map[string]types.AttributeValue{
				"userId": &types.AttributeValueMemberS{Value: userID},
			})
		}

		result, err := db.BatchGetItem(c.Request.Context(), &dynamodb.BatchGetItemInput{
			RequestItems: map[string]types.KeysAndAttributes{
				tableName: {Keys: keys},
			},
		})
		if err != nil {
			log.Printf("get selected user stats: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve selected user stats"})
			return
		}
		if len(result.UnprocessedKeys[tableName].Keys) > 0 {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "some user lookups were not processed; please retry"})
			return
		}

		items := result.Responses[tableName]
		var records []UserStats
		if err := attributevalue.UnmarshalListOfMaps(items, &records); err != nil {
			log.Printf("decode selected user stats: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "stored user statistics are invalid"})
			return
		}

		statsByID := make(map[string]UserStats, len(records))
		for _, record := range records {
			statsByID[record.UserID] = record
		}
		stats := make([]UserStats, 0, len(records))
		missingUserIDs := make([]string, 0)
		for _, userID := range uniqueIDs {
			if record, found := statsByID[userID]; found {
				stats = append(stats, record)
			} else {
				missingUserIDs = append(missingUserIDs, userID)
			}
		}
		c.JSON(http.StatusOK, gin.H{"users": stats, "missingUserIds": missingUserIDs})
	})

	router.GET("/api/users/:userID/stats", func(c *gin.Context) {
		userID := strings.TrimSpace(c.Param("userID"))
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
			return
		}

		result, err := db.GetItem(c.Request.Context(), &dynamodb.GetItemInput{
			TableName: aws.String(tableName),
			Key: map[string]types.AttributeValue{
				"userId": &types.AttributeValueMemberS{Value: userID},
			},
		})
		if err != nil {
			log.Printf("get stats for user %s: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user stats"})
			return
		}
		if len(result.Item) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "user stats not found"})
			return
		}

		var stats UserStats
		if err := attributevalue.UnmarshalMap(result.Item, &stats); err != nil {
			log.Printf("decode stats for user %s: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "stored user stats are invalid"})
			return
		}
		c.JSON(http.StatusOK, stats)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("start API server: %v", err)
	}
}

func localFrontendCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Origin") == "http://localhost:3000" {
			c.Header("Access-Control-Allow-Origin", "http://localhost:3000")
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
