package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/emrserverless"
	"github.com/aws/aws-sdk-go-v2/service/emrserverless/types"
	"github.com/gin-gonic/gin"
)

var jobIDPattern = regexp.MustCompile(`^[a-zA-Z0-9-]{1,64}$`)

type processingConfig struct{ token string }
type processingJob struct{ ID, State, Details string }

type jobRunner interface {
	Configured() bool
	Start(context.Context) (processingJob, error)
	Get(context.Context, string) (processingJob, error)
}

func processingConfigFromEnvironment() processingConfig {
	return processingConfig{token: os.Getenv("PROCESSING_API_TOKEN")}
}

func registerJobRoutes(router *gin.Engine, processing processingConfig) {
	runner := newEMRServerlessJobRunner()
	jobs := router.Group("/api/jobs", requireJobToken(processing.token))
	jobs.POST("", func(c *gin.Context) {
		if !runner.Configured() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "processing job is not configured"})
			return
		}
		job, err := runner.Start(c.Request.Context())
		if err != nil {
			log.Printf("start processing job: %v", err)
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
		job, err := runner.Get(c.Request.Context(), jobID)
		if err != nil {
			log.Printf("get processing job %s: %v", jobID, err)
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

type emrServerlessJobRunner struct {
	applicationID, entryPoint, inputURI, logURI, executionRole, table, pyFiles, sparkSubmitParameters string
}

func newEMRServerlessJobRunner() jobRunner {
	return &emrServerlessJobRunner{
		applicationID: os.Getenv("EMR_SERVERLESS_APPLICATION_ID"),
		entryPoint:    os.Getenv("PROCESSING_ENTRY_POINT"),
		inputURI:      os.Getenv("PROCESSING_INPUT_URI"), logURI: os.Getenv("PROCESSING_LOG_URI"),
		executionRole: os.Getenv("EMR_SERVERLESS_EXECUTION_ROLE_ARN"), table: dynamoDBTable(),
		pyFiles: os.Getenv("PROCESSING_PY_FILES"), sparkSubmitParameters: os.Getenv("PROCESSING_SPARK_SUBMIT_PARAMETERS"),
	}
}

func (r *emrServerlessJobRunner) Configured() bool {
	return r.applicationID != "" && r.entryPoint != "" && r.inputURI != "" && r.logURI != "" && r.executionRole != ""
}

func (r *emrServerlessJobRunner) client(ctx context.Context) (*emrserverless.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(awsRegion()))
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	return emrserverless.NewFromConfig(cfg), nil
}

func (r *emrServerlessJobRunner) Start(ctx context.Context) (processingJob, error) {
	client, err := r.client(ctx)
	if err != nil {
		return processingJob{}, err
	}
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return processingJob{}, fmt.Errorf("generate EMR client token: %w", err)
	}
	sparkParameters := r.sparkSubmitParameters
	if r.pyFiles != "" {
		sparkParameters = strings.TrimSpace(sparkParameters + " --py-files " + r.pyFiles)
	}
	output, err := client.StartJobRun(ctx, &emrserverless.StartJobRunInput{
		ApplicationId: aws.String(r.applicationID), ClientToken: aws.String(hex.EncodeToString(token)),
		ExecutionRoleArn: aws.String(r.executionRole), Name: aws.String("twitter-user-stats"),
		JobDriver: &types.JobDriverMemberSparkSubmit{Value: types.SparkSubmit{
			EntryPoint: aws.String(r.entryPoint), EntryPointArguments: []string{r.inputURI, r.table},
			SparkSubmitParameters: aws.String(sparkParameters),
		}},
		ConfigurationOverrides: &types.ConfigurationOverrides{MonitoringConfiguration: &types.MonitoringConfiguration{
			S3MonitoringConfiguration: &types.S3MonitoringConfiguration{LogUri: aws.String(r.logURI)},
		}},
	})
	if err != nil {
		return processingJob{}, fmt.Errorf("start EMR Serverless job: %w", err)
	}
	return processingJob{ID: aws.ToString(output.JobRunId), State: "SUBMITTED"}, nil
}

func (r *emrServerlessJobRunner) Get(ctx context.Context, jobID string) (processingJob, error) {
	client, err := r.client(ctx)
	if err != nil {
		return processingJob{}, err
	}
	output, err := client.GetJobRun(ctx, &emrserverless.GetJobRunInput{ApplicationId: aws.String(r.applicationID), JobRunId: aws.String(jobID)})
	if err != nil {
		return processingJob{}, fmt.Errorf("get EMR Serverless job: %w", err)
	}
	return processingJob{ID: jobID, State: string(output.JobRun.State), Details: aws.ToString(output.JobRun.StateDetails)}, nil
}
