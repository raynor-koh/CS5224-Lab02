"""EMR Serverless PySpark job for Twitter follower/followee counts.

Usage:
    spark-submit process.py s3://BUCKET/data/twitter_combined.txt userStats

The EMR Serverless runtime role needs S3 read access to the input and script,
S3 write access for logs, and DynamoDB BatchWriteItem on the target table.
"""

import os
import sys

import boto3
from pyspark.sql import SparkSession


def parse_edge(line):
    fields = line.split()
    if len(fields) != 2 or not all(field.isdecimal() for field in fields):
        return None
    return fields[0], fields[1]


def write_partition(records, table_name):
    """Write one DynamoDB item per user, retrying unprocessed batch entries."""
    endpoint_url = os.environ.get("AWS_ENDPOINT_URL") or os.environ.get(
        "DYNAMODB_ENDPOINT_URL"
    )
    region_name = os.environ.get("AWS_REGION") or os.environ.get(
        "AWS_DEFAULT_REGION"
    )
    table = boto3.resource(
        "dynamodb", endpoint_url=endpoint_url, region_name=region_name
    ).Table(table_name)
    with table.batch_writer(overwrite_by_pkeys=["userId"]) as batch:
        for user_id, counts in records:
            followers, followees = counts
            batch.put_item(
                Item={
                    "userId": user_id,
                    "followers": int(followers or 0),
                    "followees": int(followees or 0),
                }
            )


def main():
    if len(sys.argv) != 3:
        raise SystemExit("usage: process.py INPUT_S3_URI DYNAMODB_TABLE")
    input_uri, table_name = sys.argv[1:]
    spark = SparkSession.builder.appName("TwitterUserStats").getOrCreate()
    sc = spark.sparkContext
    edges = (
        sc.textFile(input_uri)
        .map(parse_edge)
        .filter(lambda edge: edge is not None)
        .filter(lambda edge: edge[0] != edge[1])
        .distinct()
    )
    followers = edges.map(lambda edge: (edge[1], 1)).reduceByKey(lambda left, right: left + right)
    followees = edges.map(lambda edge: (edge[0], 1)).reduceByKey(lambda left, right: left + right)
    stats = followers.fullOuterJoin(followees)
    for user_id, counts in sorted(stats.filter(lambda record: record[0] in {"214328887", "107830991"}).collect()):
        print(f"{user_id}: {counts[0] or 0} followers, {counts[1] or 0} followees")
    stats.foreachPartition(lambda records: write_partition(records, table_name))
    spark.stop()


if __name__ == "__main__":
    main()
