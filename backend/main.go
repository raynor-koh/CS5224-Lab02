package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type UserStats struct {
	UserID    string `json:"userId"`
	Followers int64  `json:"followers"`
	Followees int64  `json:"followees"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/users/{userID}/stats", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userID")
		if strings.TrimSpace(userID) == "" {
			http.Error(w, `{"error":"userId is required"}`, http.StatusBadRequest)
			return
		}

		// Temporary data; replace this lookup with DynamoDB.
		stats := UserStats{
			UserID:    userID,
			Followers: 123,
			Followees: 456,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	log.Println("API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
