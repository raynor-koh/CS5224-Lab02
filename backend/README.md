# Backend API and processing job

The Go API reads precomputed user statistics from DynamoDB. An EMR Serverless
PySpark job reads the Twitter edge list from S3, removes duplicate edges,
counts followers and followees, and writes one item per user to DynamoDB.

```text
S3 input -> EMR Serverless (PySpark) -> DynamoDB -> Go API
```

## API routes

| Method | Path | Result |
| --- | --- | --- |
| GET | `/health` | HTTP 200 for container and load balancer health checks |
| GET | `/api/users?limit=100&cursor=...` | Paginated user IDs |
| GET | `/api/users/stats?limit=100&cursor=...` | Paginated statistics |
| POST | `/api/users/stats` | Statistics for 1–100 user IDs |
| GET | `/api/users/:userID/stats` | One user's statistics |
| POST | `/api/jobs` | Starts an EMR Serverless Spark run |
| GET | `/api/jobs/:jobID` | Returns the EMR Serverless job state |

List routes use DynamoDB Scan. Result order is not guaranteed; treat cursors as
opaque. Job routes require `Authorization: Bearer <token>` matching
`PROCESSING_API_TOKEN`.

## Run locally

From the repository root, start Floci, initialize its DynamoDB table and S3
input, then start the API and frontend:

```sh
docker compose up --build
```

Open <http://localhost:3000>. The API listens on <http://localhost:8080> and
Floci is available at <http://localhost:4566>. To run the local Spark
processor after initialization:

```sh
docker compose --profile processing run --rm spark-job
```

The processor reads the mounted dataset and writes statistics to Floci
DynamoDB. EMR Serverless is not emulated locally.

For running the API outside Compose, copy `.env.example` to `.env` in the
repository root or `backend/`. Set `AWS_ENDPOINT_URL=http://localhost:4566`
for Floci and provide local emulator credentials. Never put real AWS keys in
`.env` or source control. For deployed services, use an ECS task role instead
of long-lived access keys. See [`AWS_DEPLOYMENT.md`](../AWS_DEPLOYMENT.md).

## Runtime configuration

The API defaults to port `8080`, region `ap-southeast-1`, and DynamoDB table
`userStats`. Set `FRONTEND_ORIGIN` to the exact browser origin, including
scheme, for CORS. In ECS, set `AWS_REGION`, `DYNAMODB_TABLE`, and
`FRONTEND_ORIGIN` as service environment variables. The ECS task role needs
DynamoDB read permissions. The separate task execution role needs permission
to pull the image and write logs.

Set `GIN_MODE=release` in ECS for production logging. The `/health` endpoint
returns HTTP 200, which matches the default Application Load Balancer success
code. If health checks fail, verify the configured container port is `8080`,
the path is `/health`, and the task is staying running; inspect stopped reason
and container logs for startup errors.
