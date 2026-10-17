package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type UserStats struct {
	UserID    string `json:"userId" dynamodbav:"userId"`
	Followers int64  `json:"followers" dynamodbav:"followers"`
	Followees int64  `json:"followees" dynamodbav:"followees"`
}

func main() {
	loadEnvironment()

	store, closeStore, err := newStatsStore(context.Background())
	if err != nil {
		log.Fatalf("create statistics store: %v", err)
	}
	defer closeStore()

	router := gin.Default()
	router.Use(frontendCORS())
	router.OPTIONS("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	registerUserRoutes(router, store)
	registerJobRoutes(router, processingConfigFromEnvironment())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("start API server: %v", err)
	}
}

func loadEnvironment() {
	// Support starting the backend from either the repository root or backend/.
	if err := godotenv.Load(".env"); err != nil {
		if err := godotenv.Load("../.env"); err != nil {
			log.Printf("load .env or ../.env: %v", err)
		}
	}
}

func registerUserRoutes(router *gin.Engine, store statsStore) {
	router.GET("/api/users", func(c *gin.Context) {
		limit, cursor, ok := scanPage(c)
		if !ok {
			return
		}
		ids, nextCursor, err := store.ListUserIDs(c.Request.Context(), limit, cursor)
		if err != nil {
			log.Printf("list user IDs: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"userIds": ids, "nextCursor": nextCursor})
	})

	router.GET("/api/users/stats", func(c *gin.Context) {
		limit, cursor, ok := scanPage(c)
		if !ok {
			return
		}
		users, nextCursor, err := store.ListStats(c.Request.Context(), limit, cursor)
		if err != nil {
			log.Printf("list all user stats: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list user stats"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"users": users, "nextCursor": nextCursor})
	})

	router.POST("/api/users/stats", func(c *gin.Context) {
		var request struct {
			UserIDs []string `json:"userIds" binding:"required,min=1,max=100,dive,required"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "provide between 1 and 100 user IDs"})
			return
		}

		uniqueIDs := uniqueUserIDs(request.UserIDs)
		if len(uniqueIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user IDs cannot be empty"})
			return
		}
		records, err := store.GetStats(c.Request.Context(), uniqueIDs)
		if err != nil {
			log.Printf("get selected user stats: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user stats"})
			return
		}

		statsByID := make(map[string]UserStats, len(records))
		for _, record := range records {
			statsByID[record.UserID] = record
		}
		stats := make([]UserStats, 0, len(records))
		missing := make([]string, 0)
		for _, id := range uniqueIDs {
			if record, found := statsByID[id]; found {
				stats = append(stats, record)
			} else {
				missing = append(missing, id)
			}
		}
		c.JSON(http.StatusOK, gin.H{"users": stats, "missingUserIds": missing})
	})

	router.GET("/api/users/:userID/stats", func(c *gin.Context) {
		userID := strings.TrimSpace(c.Param("userID"))
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
			return
		}
		stats, found, err := store.GetStat(c.Request.Context(), userID)
		if err != nil {
			log.Printf("get stats for user %s: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user stats"})
			return
		}
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "user stats not found"})
			return
		}
		c.JSON(http.StatusOK, stats)
	})
}

func uniqueUserIDs(ids []string) []string {
	unique := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func frontendCORS() gin.HandlerFunc {
	allowedOrigin := os.Getenv("FRONTEND_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}
	return func(c *gin.Context) {
		if c.GetHeader("Origin") == allowedOrigin {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
