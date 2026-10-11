package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var jobIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,61}[a-z0-9]$`)

type processingConfig struct {
	token string
}

type processingJob struct {
	ID      string
	State   string
	Details string
}

type jobRunner interface {
	Configured() bool
	Start(context.Context) (processingJob, error)
	Get(context.Context, string) (processingJob, error)
}

func processingConfigFromEnvironment() processingConfig {
	return processingConfig{token: os.Getenv("PROCESSING_API_TOKEN")}
}

func registerJobRoutes(router *gin.Engine, config processingConfig) {
	runner := newGCPJobRunner()

	jobs := router.Group("/api/jobs", requireJobToken(config.token))
	jobs.POST("", func(c *gin.Context) {
		if !runner.Configured() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "processing job is not configured"})
			return
		}
		job, startErr := runner.Start(c.Request.Context())
		if startErr != nil {
			log.Printf("start processing job: %v", startErr)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to start processing job"})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"jobId": job.ID, "state": job.State, "stateDetails": job.Details})
	})

	jobs.GET("/:jobID", func(c *gin.Context) {
		if !runner.Configured() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "processing job is not configured"})
			return
		}
		jobID := c.Param("jobID")
		if !jobIDPattern.MatchString(jobID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
			return
		}
		job, getErr := runner.Get(c.Request.Context(), jobID)
		if getErr != nil {
			log.Printf("get processing job %s: %v", jobID, getErr)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to retrieve processing job"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"jobId": job.ID, "state": job.State, "stateDetails": job.Details})
	})
}

func requireJobToken(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expected == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "processing API token is not configured"})
			c.Abort()
			return
		}
		provided, bearer := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		expectedHash := sha256.Sum256([]byte(expected))
		providedHash := sha256.Sum256([]byte(provided))
		if !bearer || provided == "" || subtle.ConstantTimeCompare(expectedHash[:], providedHash[:]) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// gcpJobRunner submits PySpark batches to Serverless for Apache Spark. The
// backend image must contain the gcloud CLI; Cloud Run service-account identity
// supplies credentials instead of a downloaded key file.
type gcpJobRunner struct {
	projectID     string
	region        string
	entryPoint    string
	inputURI      string
	stagingBucket string
	pyFiles       string
	collection    string
}

func newGCPJobRunner() jobRunner {
	region := os.Getenv("GCP_REGION")
	if region == "" {
		region = "asia-southeast1"
	}
	return &gcpJobRunner{
		projectID: os.Getenv("GCP_PROJECT_ID"), region: region,
		entryPoint: os.Getenv("PROCESSING_ENTRY_POINT"), inputURI: os.Getenv("PROCESSING_INPUT_URI"),
		stagingBucket: os.Getenv("GCP_DATAPROC_STAGING_BUCKET"), pyFiles: os.Getenv("PROCESSING_PY_FILES"), collection: firestoreCollection(),
	}
}

func (r *gcpJobRunner) Configured() bool {
	return r.projectID != "" && r.entryPoint != "" && r.inputURI != "" && r.stagingBucket != "" && r.pyFiles != ""
}

func (r *gcpJobRunner) Start(ctx context.Context) (processingJob, error) {
	token := make([]byte, 8)
	if _, err := rand.Read(token); err != nil {
		return processingJob{}, fmt.Errorf("generate Dataproc batch ID: %w", err)
	}
	batchID := "twitter-" + hex.EncodeToString(token)
	var response struct {
		State        string `json:"state"`
		StateMessage string `json:"stateMessage"`
	}
	err := runGCloud(ctx, &response, "dataproc", "batches", "submit", "pyspark", r.entryPoint,
		"--batch", batchID, "--project", r.projectID, "--region", r.region,
		"--deps-bucket", r.stagingBucket, "--py-files", r.pyFiles, "--", r.inputURI, r.projectID, r.collection)
	if err != nil {
		return processingJob{}, err
	}
	return processingJob{ID: batchID, State: response.State, Details: response.StateMessage}, nil
}

func (r *gcpJobRunner) Get(ctx context.Context, jobID string) (processingJob, error) {
	var response struct {
		State        string `json:"state"`
		StateMessage string `json:"stateMessage"`
	}
	if err := runGCloud(ctx, &response, "dataproc", "batches", "describe", jobID,
		"--project", r.projectID, "--region", r.region); err != nil {
		return processingJob{}, err
	}
	return processingJob{ID: jobID, State: response.State, Details: response.StateMessage}, nil
}

func runGCloud(ctx context.Context, response any, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	args = append(args, "--format=json", "--quiet")
	output, err := exec.CommandContext(ctx, "gcloud", args...).CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.New("gcloud timed out")
		}
		return fmt.Errorf("gcloud failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return json.Unmarshal(output, response)
}
