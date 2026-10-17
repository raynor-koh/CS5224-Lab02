# AWS backend and Spark processing

The Go API reads precomputed statistics from DynamoDB. An EMR Serverless
PySpark job reads the Twitter edge list from S3, removes duplicate edges,
counts followers and followees, and writes one item per user to DynamoDB.

~~~text
S3 -> EMR Serverless (PySpark) -> DynamoDB -> Go API
~~~

## API

| Method | Path | Result |
| --- | --- | --- |
| GET | /api/users?limit=100&cursor=... | paginated user IDs |
| GET | /api/users/stats?limit=100&cursor=... | paginated statistics |
| POST | /api/users/stats | selected statistics for 1–100 user IDs |
| GET | /api/users/:userID/stats | one user's statistics |
| POST | /api/jobs | starts an EMR Serverless Spark run |
| GET | /api/jobs/:jobID | returns the EMR Serverless job state |

The list endpoints use DynamoDB Scan, so their order is not guaranteed and
the returned cursor must be treated as opaque. The job routes require an
Authorization Bearer token configured as PROCESSING_API_TOKEN.

## Run locally

Docker Compose starts Floci's local AWS emulator, creates a DynamoDB
userStats table, uploads the dataset to local S3, then starts the Go API and
Next.js frontend:

~~~sh
docker compose up --build
~~~

Open http://localhost:3000. The API runs at http://localhost:8080; Floci is
exposed at http://localhost:4566.

Run the one-shot Spark processor after Compose is ready:

~~~sh
docker compose --profile processing run --rm spark-job
~~~

It reads the mounted local input file and writes userStats to Floci DynamoDB.
The initialiser also uploads the input to Floci S3 at
s3://twitter-data/data/twitter_combined.txt, allowing the application's S3
and DynamoDB setup to be inspected locally. EMR Serverless itself is not
emulated; use the AWS deployment for cloud-run evidence.

## AWS configuration

Copy .env.example to .env. The backend uses the AWS SDK default credential
chain. For any deployed backend, attach an IAM role rather than distributing
long-lived AWS keys. See AWS_DEPLOYMENT.md for the S3, EMR Serverless,
DynamoDB, IAM, and CloudFront setup.
