package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestScanPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name   string
		query  string
		limit  int
		id     string
		status int
	}{
		{name: "defaults", limit: 100, status: http.StatusOK},
		{name: "limit and cursor", query: "limit=25&cursor=" + base64.RawURLEncoding.EncodeToString([]byte("214328887")), limit: 25, id: "214328887", status: http.StatusOK},
		{name: "limit capped", query: "limit=1000", limit: 250, status: http.StatusOK},
		{name: "invalid limit", query: "limit=0", status: http.StatusBadRequest},
		{name: "invalid cursor", query: "cursor=invalid!", status: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(writer)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/users?"+tt.query, nil)
			limit, cursor, ok := scanPage(c)
			if tt.status != http.StatusOK {
				if ok || writer.Code != tt.status {
					t.Fatalf("got ok=%t, status=%d", ok, writer.Code)
				}
				return
			}
			if !ok || limit != tt.limit {
				t.Fatalf("got ok=%t, limit=%d", ok, limit)
			}
			if tt.id != "" {
				if cursor != tt.id || pageCursor(cursor) != base64.RawURLEncoding.EncodeToString([]byte(tt.id)) {
					t.Fatalf("cursor round trip failed: %q", cursor)
				}
			}
		})
	}
}

func TestJobToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name   string
		secret string
		header string
		status int
	}{
		{name: "valid", secret: "secret", header: "Bearer secret", status: http.StatusNoContent},
		{name: "wrong token", secret: "secret", header: "Bearer wrong", status: http.StatusUnauthorized},
		{name: "missing bearer", secret: "secret", header: "secret", status: http.StatusUnauthorized},
		{name: "unconfigured", header: "Bearer secret", status: http.StatusServiceUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/", requireJobToken(tt.secret), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Authorization", tt.header)
			writer := httptest.NewRecorder()
			router.ServeHTTP(writer, request)
			if writer.Code != tt.status {
				t.Fatalf("got status %d, want %d", writer.Code, tt.status)
			}
		})
	}
}
