# Deploying the Twitter Statistics Application on AWS (Windows)

This guide deploys the application in one AWS Region using managed services:

~~~text
S3 input + PySpark script -> EMR Serverless -> DynamoDB -> Go API (App Runner) -> Next.js UI (App Runner)
~~~

It is written for **Windows PowerShell** and the AWS Console. Use the same Region for every resource; this guide uses `ap-southeast-1` (Asia Pacific — Singapore). Do not use the local Floci endpoint or local AWS credentials in a deployed service.

> **Cost note:** Set an AWS Budget before starting. EMR Serverless and App Runner incur charges while work or services are running. Keep EMR Serverless initial capacity at zero and delete demonstration resources afterwards.

## Prerequisites

- AWS account with permission to use S3, DynamoDB, IAM, ECR, EMR Serverless, and App Runner.
- Docker Desktop running on Windows.
- A current Windows Python installation with pip. Install it from
  [python.org](https://www.python.org/downloads/windows/) and select **Install
  launcher for all users**. The commands below deliberately use the `py`
  launcher rather than an MSYS2 or Git Bash `python` on `PATH`.
- AWS CLI v2 configured in PowerShell (`aws configure`) or AWS CloudShell.
- From the repository root, PowerShell is the terminal used below.

Set values once for the session. Replace the placeholders; the S3 bucket name must be globally unique and lower-case.

~~~powershell
$Region = "ap-southeast-1"
$AccountId = (aws sts get-caller-identity --query Account --output text)
$DataBucket = "YOUR-UNIQUE-twitter-data-bucket"
$TableName = "userStats"
$BackendRepository = "twitter-backend"
$FrontendRepository = "twitter-frontend"
~~~

## 1. Create DynamoDB and the data bucket

### DynamoDB

1. In the AWS Console, switch to `$Region`, then open **DynamoDB**.
2. Choose **Create table** and enter:
   - Table name: `userStats`
   - Partition key: `userId` with type **String**
   - Table settings: **On-demand** capacity
3. Choose **Create table** and wait for the table to become Active.

### S3

1. Open **S3** → **Create bucket**.
2. Name it `$DataBucket` and select `$Region`.
3. Keep **Block all public access** enabled; this bucket is private.
4. Create the bucket.

The EMR job needs the input file, the PySpark program, and a dependency archive. Create the archive in PowerShell, then upload all three files:

~~~powershell
$DependencyDir = Join-Path $env:TEMP "twitter-boto3"
$DependencyZip = Join-Path $env:TEMP "boto3.zip"
Remove-Item -Recurse -Force $DependencyDir -ErrorAction SilentlyContinue
Remove-Item -Force $DependencyZip -ErrorAction SilentlyContinue

if (-not (Get-Command py -ErrorAction SilentlyContinue)) {
  throw "Python Launcher (py) was not found. Install Python from python.org, then open a new PowerShell window."
}
& py -m pip --version
if ($LASTEXITCODE -ne 0) {
  & py -m ensurepip --upgrade
  if ($LASTEXITCODE -ne 0) { throw "pip could not be installed for the selected Python." }
}
& py -m pip install --target $DependencyDir -r backend/jobs/requirements-aws.txt
if ($LASTEXITCODE -ne 0) { throw "Failed to install the EMR Python dependencies." }
Compress-Archive -Path "$DependencyDir\*" -DestinationPath $DependencyZip -ErrorAction Stop

aws s3 cp data/twitter_combined.txt "s3://$DataBucket/data/twitter_combined.txt" --region $Region
aws s3 cp backend/jobs/process.py "s3://$DataBucket/jobs/process.py" --region $Region
aws s3 cp $DependencyZip "s3://$DataBucket/dependencies/boto3.zip" --region $Region
~~~

You can instead upload these files from the S3 console, preserving the exact object keys shown above.

## 2. Create the EMR Serverless runtime role and application

In **IAM** → **Roles** → **Create role**:

1. Choose **Custom trust policy**, enter the following, and name the role `TwitterEmrRuntimeRole`.

~~~json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {"Service": "emr-serverless.amazonaws.com"},
    "Action": "sts:AssumeRole"
  }]
}
~~~

2. Add an inline policy. Replace `ACCOUNT_ID` and `YOUR-UNIQUE-twitter-data-bucket`.

~~~json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["s3:ListBucket"],
      "Resource": "arn:aws:s3:::YOUR-UNIQUE-twitter-data-bucket"
    },
    {
      "Effect": "Allow",
      "Action": ["s3:GetObject"],
      "Resource": [
        "arn:aws:s3:::YOUR-UNIQUE-twitter-data-bucket/data/*",
        "arn:aws:s3:::YOUR-UNIQUE-twitter-data-bucket/jobs/*",
        "arn:aws:s3:::YOUR-UNIQUE-twitter-data-bucket/dependencies/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": ["s3:PutObject"],
      "Resource": "arn:aws:s3:::YOUR-UNIQUE-twitter-data-bucket/logs/*"
    },
    {
      "Effect": "Allow",
      "Action": ["dynamodb:BatchWriteItem"],
      "Resource": "arn:aws:dynamodb:ap-southeast-1:ACCOUNT_ID:table/userStats"
    }
  ]
}
~~~

Open **EMR** → **EMR Serverless**. AWS now creates and manages Serverless
applications through the EMR Studio interface, so the **Create application**
button is not shown on the initial Serverless landing page.

1. If there is no Studio in the Region, select **Get started** → **Create and
   launch Studio**. Choose the batch-job setup; do not enable or configure
   interactive workloads unless you need them.
2. If a Studio already exists, choose **Manage applications** (or open that
   Studio). The Studio opens in a new browser tab.
3. In Studio, choose **Create application** and enter:

- Name: `twitter-user-stats`
- Type: **Spark**
- **Custom settings**, then set pre-initialized capacity to **zero**
- Application limits: **4 vCPU**, **20 GB memory**, and **80 GB disk**
- Keep auto-start enabled and retain the 15-minute auto-stop timeout

The submitted job limits itself to one driver plus at most two 1-vCPU
executors. Its 4-GB driver/executor memory settings need approximately 5 GB
per worker after Spark overhead, so 3 vCPU, 15 GB, and 60 GB is the minimum.
The limits above leave a small margin without reserving or charging for idle
capacity; the zero pre-initialized capacity is what prevents idle workers.

Record its application ID. Submit a job using the application page:

- Runtime role: `TwitterEmrRuntimeRole`
- Script location: `s3://$DataBucket/jobs/process.py`
- Script arguments: the current Studio form validates this field as a JSON
  array. Enter the two arguments exactly as follows (including square brackets
  and double quotes):

  ~~~text
  ["s3://YOUR-UNIQUE-twitter-data-bucket/data/twitter_combined.txt", "userStats"]
  ~~~

- Under **Spark properties**, add one key/value row for each of the following.
  Do **not** use `spark.submit.pyFiles` for `boto3.zip`: botocore needs its AWS
  service-definition files on a real filesystem, not inside a Python zip
  importer. `spark.archives` extracts the archive into `python-deps` on both
  the driver and executors.

  | Key | Value |
  | --- | --- |
  | `spark.archives` | `s3://YOUR-UNIQUE-twitter-data-bucket/dependencies/boto3.zip#python-deps` |
  | `spark.emr-serverless.driverEnv.PYTHONPATH` | `./python-deps` |
  | `spark.executorEnv.PYTHONPATH` | `./python-deps` |
  | `spark.emr-serverless.driverEnv.AWS_REGION` | `ap-southeast-1` |
  | `spark.executorEnv.AWS_REGION` | `ap-southeast-1` |
  | `spark.executor.cores` | `1` |
  | `spark.executor.memory` | `4g` |
  | `spark.driver.cores` | `1` |
  | `spark.driver.memory` | `4g` |
  | `spark.dynamicAllocation.maxExecutors` | `2` |

- Scroll below **Job configuration** to **Additional settings**, enable S3
  application logging if required by the form, and set its S3 log location to
  `s3://YOUR-UNIQUE-twitter-data-bucket/logs/`.

Before submission, verify that the table exists in the same account and Region
as the EMR application. A missing table produces `ResourceNotFoundException`,
whereas an IAM-policy problem produces `AccessDeniedException`:

~~~powershell
aws dynamodb describe-table --table-name userStats --region $Region
~~~

Wait for a **Success** state, then confirm the `userStats` table contains items.

## 3. Create ECR repositories and push the backend image

In **ECR** → **Private repositories**, create `twitter-backend` and `twitter-frontend`. In each repository, choose **View push commands** if you need AWS-generated login instructions.

From PowerShell at the repository root, build and push the backend:

~~~powershell
aws ecr get-login-password --region $Region |
  docker login --username AWS --password-stdin "$AccountId.dkr.ecr.$Region.amazonaws.com"

docker build -t "${BackendRepository}:latest" ./backend
docker tag "${BackendRepository}:latest" "$AccountId.dkr.ecr.$Region.amazonaws.com/${BackendRepository}:latest"
docker push "$AccountId.dkr.ecr.$Region.amazonaws.com/${BackendRepository}:latest"
~~~

## 4. Deploy the Go API with ECS Express Mode

AWS App Runner no longer accepts new customers. Use **Amazon ECS Express Mode**
instead. It deploys the same ECR image to Fargate and automatically creates an
HTTPS URL, load balancer, networking, logs, and scaling configuration.

Create an IAM role named `TwitterApiTaskRole` with this trust policy:

~~~json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {"Service": "ecs-tasks.amazonaws.com"},
    "Action": "sts:AssumeRole"
  }]
}
~~~

Attach this inline policy, replacing `ACCOUNT_ID`. The EMR permissions are only needed if you will use the protected `/api/jobs` routes.

~~~json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["dynamodb:GetItem", "dynamodb:BatchGetItem", "dynamodb:Scan"],
      "Resource": "arn:aws:dynamodb:ap-southeast-1:ACCOUNT_ID:table/userStats"
    },
    {
      "Effect": "Allow",
      "Action": ["emr-serverless:StartJobRun", "emr-serverless:GetJobRun"],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": "iam:PassRole",
      "Resource": "arn:aws:iam::ACCOUNT_ID:role/TwitterEmrRuntimeRole"
    }
  ]
}
~~~

In **ECS** → **Express mode** → **Create**, configure the API service:

1. For Image URI, choose **Browse ECR images** and select
   `twitter-backend:latest` (prefer the image digest when available).
2. For **Task execution role** and **Infrastructure role**, choose **Create new
   role** if this is the first Express Mode service. ECS creates
   `ecsTaskExecutionRole` and `ecsInfrastructureRoleForExpressServices` with
   the managed policies needed to pull images and create the service resources.
3. Open **Additional configurations** and set service name `twitter-api`,
   container port `8080`, health-check path `/health`, and task role
   `TwitterApiTaskRole`.
4. Add these runtime environment variables, replacing placeholders:

~~~text
AWS_REGION=ap-southeast-1
DYNAMODB_TABLE=userStats
EMR_SERVERLESS_APPLICATION_ID=YOUR_EMR_APPLICATION_ID
EMR_SERVERLESS_EXECUTION_ROLE_ARN=arn:aws:iam::ACCOUNT_ID:role/TwitterEmrRuntimeRole
PROCESSING_ENTRY_POINT=s3://YOUR_BUCKET/jobs/process.py
PROCESSING_INPUT_URI=s3://YOUR_BUCKET/data/twitter_combined.txt
PROCESSING_LOG_URI=s3://YOUR_BUCKET/logs/
PROCESSING_ARCHIVES=s3://YOUR_BUCKET/dependencies/boto3.zip#python-deps
PROCESSING_SPARK_SUBMIT_PARAMETERS=--conf spark.executor.cores=1 --conf spark.executor.memory=4g --conf spark.driver.cores=1 --conf spark.driver.memory=4g --conf spark.dynamicAllocation.maxExecutors=2
PROCESSING_API_TOKEN=USE_A_LONG_RANDOM_SECRET
~~~

5. Set compute to 0.25 vCPU and 0.5 GB memory for this coursework
   demonstration, and set minimum tasks to `1` and maximum tasks to `2`.
6. Create the service. Once deployment is active, copy its **Application URL**;
   it is the API URL used in the frontend build.

The API includes a `/health` endpoint specifically for ECS health checks. Build
and push the backend image again after pulling the current repository changes.

## 5. Build and deploy the frontend with ECS Express Mode

The checked-in frontend Dockerfile starts `next dev`, which is a development server. Before deploying, update it to make a production build and run `pnpm start`. It must also receive `NEXT_PUBLIC_API_BASE_URL` **when the image is built**, since Next.js exposes this value to browser code at build time.

Use this `frontend/Dockerfile`:

~~~dockerfile
FROM node:22-bookworm-slim

WORKDIR /app
RUN corepack enable
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .

ARG NEXT_PUBLIC_API_BASE_URL
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL
RUN pnpm build

EXPOSE 3000
CMD ["pnpm", "start", "--hostname", "0.0.0.0"]
~~~

Set the API URL to the backend ECS Express Mode Application URL, build, and
push the frontend image:

~~~powershell
$BackendUrl = "https://YOUR-BACKEND-DEFAULT-DOMAIN"

docker build --build-arg "NEXT_PUBLIC_API_BASE_URL=$BackendUrl" `
  -t "${FrontendRepository}:latest" ./frontend
docker tag "${FrontendRepository}:latest" "$AccountId.dkr.ecr.$Region.amazonaws.com/${FrontendRepository}:latest"
docker push "$AccountId.dkr.ecr.$Region.amazonaws.com/${FrontendRepository}:latest"
~~~

Create another **ECS Express Mode** service from `twitter-frontend:latest`:

- Service name: `twitter-frontend`
- Port: `3000`
- Health-check path: `/`
- Task execution and infrastructure roles: use the roles created for the API
- No task role is required.
- Compute: 0.25 vCPU and 0.5 GB memory; minimum tasks `1`, maximum tasks `2`

When it is active, copy its Application URL. Return to the `twitter-api` ECS
Express Mode service, edit its environment variables, and add:

~~~text
FRONTEND_ORIGIN=https://YOUR-FRONTEND-DEFAULT-DOMAIN
~~~

Deploy the updated API revision. This matches the Go API's CORS policy to the
frontend origin.

## 6. Verify and capture submission evidence

1. Open the frontend ECS Express Mode Application URL and load data from the UI.
2. Test the API directly:

   ~~~powershell
   Invoke-RestMethod "$BackendUrl/api/users?limit=10"
   ~~~

3. Capture screenshots of:
   - S3 objects (input, script, dependency archive, and logs)
   - the EMR Serverless application and successful job run
   - Spark logs or Spark UI
   - DynamoDB `userStats` items, including verification IDs
   - the ECS Express Mode frontend calling the deployed API

## Cleanup

After demonstrating the application, delete the ECS Express Mode services, EMR Serverless application, ECR images/repositories, S3 objects/bucket, and the DynamoDB table if they are no longer required. Delete IAM roles only after the services using them have been removed.
