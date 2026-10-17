# Frontend

This Next.js application displays user IDs and follower/followee statistics
from the Go API.

## Run locally

From the repository root, use Docker Compose to start the frontend, API, and
local AWS emulator together:

```sh
docker compose up --build
```

Open <http://localhost:3000>. To run the frontend directly, install the
workspace dependencies and set the API URL before starting Next.js:

```sh
pnpm install
```

Create `frontend/.env.local` with:

```text
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

Then run `pnpm --dir frontend dev` from the repository root (or `pnpm dev`
from this directory). The Go API must be reachable at that URL, and its
`FRONTEND_ORIGIN` must match the page origin for browser requests to pass CORS.

## Deploy to ECS

The frontend Docker image runs a production Next.js build and listens on port
`3000`. `NEXT_PUBLIC_API_BASE_URL` is embedded into the browser JavaScript at
build time. Pass the backend's HTTPS Application URL when building the image;
setting it only in the ECS runtime environment does not change the client
bundle.

From the repository root, build, tag, and push the image (replace the account
ID, region, and backend URL as needed):

```powershell
$Region = "ap-southeast-1"
$AccountId = "YOUR_ACCOUNT_ID"
$BackendUrl = "https://YOUR-BACKEND-APPLICATION-URL"
$Registry = "$AccountId.dkr.ecr.$Region.amazonaws.com"

docker build --build-arg "NEXT_PUBLIC_API_BASE_URL=$BackendUrl" `
  -t twitter-frontend:latest ./frontend
docker tag twitter-frontend:latest "$Registry/twitter-frontend:latest"
docker push "$Registry/twitter-frontend:latest"
```

Deploy the pushed image as an ECS Express Mode service on port `3000` with
health-check path `/`. Use the task execution role for ECR image pulls and
logging; the frontend does not need an application task role. After deploying,
set the backend's `FRONTEND_ORIGIN` to the frontend HTTPS Application URL and
deploy a new backend revision. If the backend URL changes, rebuild and
redeploy the frontend image.
