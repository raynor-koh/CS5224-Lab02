package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"
)

type UserStats struct {
	UserID    string `json:"userId" dynamodbav:"userId"`
	Followers int64  `json:"followers" dynamodbav:"followers"`
	Followees int64  `json:"followees" dynamodbav:"followees"`
}

func main() {
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

	if err := router.Run(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("start API server: %v", err)
	}
}

func localFrontendCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Origin") == "http://localhost:3000" {
			c.Header("Access-Control-Allow-Origin", "http://localhost:3000")
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
