# Twitter Follower and Followee Statistics

CS5224 Lab 02 project for calculating follower and followee counts from a Twitter relationship dataset and browsing the results in a web application.

## Architecture

```text
Online: Browser → Next.js frontend → Go API → DynamoDB userStats
Batch:  Twitter edge list in S3 → EMR Serverless PySpark job → DynamoDB userStats
```

Each line of the dataset represents a directed follower-to-followee relationship. The Spark job removes invalid and duplicate edges, counts incoming and outgoing relationships for each user, and writes the results to DynamoDB. The Go API serves those precomputed counts to the Next.js frontend. The frontend and backend use separate Docker images in Amazon ECR.

## Run locally

From the repository root, start the frontend, backend, and local AWS emulator with Docker Compose:

```sh
docker compose up --build
```

Open <http://localhost:3000> for the frontend. The API is at <http://localhost:8080>; its health endpoint is <http://localhost:8080/health>. To run the local Spark processing job after the services are initialized:

```sh
docker compose --profile processing run --rm spark-job
```

The local setup uses Floci and development-only credentials defined in `compose.yaml`. It does not launch EMR Serverless. See the [backend guide](backend/README.md) and [frontend guide](frontend/README.md) for API routes, configuration, and alternative local startup steps.

## Deploy on AWS

Follow the [AWS deployment guide](AWS_DEPLOYMENT.md) for the S3 dataset and script, EMR Serverless job, DynamoDB table, ECR images, ECS Express Mode services, and IAM roles. The frontend's `NEXT_PUBLIC_API_BASE_URL` must be set to the backend's HTTPS URL **when building the frontend image**. Set the backend's `FRONTEND_ORIGIN` to the deployed frontend origin for browser requests.

Use IAM roles for deployed services and keep real AWS credentials and the administrative `PROCESSING_API_TOKEN` out of source control. The optional `/api/jobs` endpoints use that token to start and inspect EMR Serverless runs.

## Project files

- [`backend/`](backend/) contains the Go API and the [PySpark processing job](backend/jobs/process.py).
- [`frontend/`](frontend/) contains the Next.js application.
- [`data/`](data/) contains the input dataset and data validation utilities.
