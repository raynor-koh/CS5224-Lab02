#!/bin/sh
set -eu

endpoint="http://floci:4566"

until aws dynamodb describe-table --endpoint-url "$endpoint" --table-name userStats >/dev/null 2>&1
do
  aws dynamodb create-table --endpoint-url "$endpoint" --table-name userStats --attribute-definitions AttributeName=userId,AttributeType=S --key-schema AttributeName=userId,KeyType=HASH --billing-mode PAY_PER_REQUEST >/dev/null 2>&1 || true
  sleep 1
done

aws s3 mb "s3://twitter-data" --endpoint-url "$endpoint" >/dev/null 2>&1 || true
aws s3 cp /input/twitter_combined.txt "s3://twitter-data/data/twitter_combined.txt" --endpoint-url "$endpoint" >/dev/null
