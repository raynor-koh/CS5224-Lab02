"""Dataproc Serverless PySpark job for Twitter follower/followee counts.

Run with:
gcloud dataproc batches submit pyspark gs://BUCKET/jobs/process.py \
  --region=asia-southeast1 --deps-bucket=gs://BUCKET -- \
  gs://BUCKET/data/twitter_combined.txt PROJECT_ID userStats

The Dataproc workload service account needs Cloud Datastore User access to the
target Firestore database. Package google-cloud-firestore with the job runtime.
"""

import sys

from google.cloud import firestore
from pyspark.sql import SparkSession


def parse_edge(line):
    fields = line.split()
    if len(fields) != 2:
        return None
    return fields[0], fields[1]  # follower, followee


def write_partition(records, project_id, collection_name):
    """Write each Spark partition in Firestore batches (limit: 500 writes)."""
    client = firestore.Client(project=project_id)
    collection = client.collection(collection_name)
    batch = client.batch()
    writes = 0

    for user_id, counts in records:
        followers, followees = counts
        batch.set(collection.document(user_id), {
            "userId": user_id,
            "followers": int(followers or 0),
            "followees": int(followees or 0),
        })
        writes += 1
        # Stay below Firestore's 500-operation batch limit.
        if writes == 400:
            batch.commit()
            batch = client.batch()
            writes = 0

    if writes:
        batch.commit()


def main():
    if len(sys.argv) != 4:
        raise SystemExit("usage: process.py INPUT_GCS_URI GCP_PROJECT_ID FIRESTORE_COLLECTION")

    input_uri, project_id, collection_name = sys.argv[1:]
    spark = SparkSession.builder.appName("TwitterUserStats").getOrCreate()
    sc = spark.sparkContext

    edges = sc.textFile(input_uri).map(parse_edge).filter(lambda edge: edge is not None).distinct()
    followers = edges.map(lambda edge: (edge[1], 1)).reduceByKey(lambda left, right: left + right)
    followees = edges.map(lambda edge: (edge[0], 1)).reduceByKey(lambda left, right: left + right)
    stats = followers.fullOuterJoin(followees)
    stats.foreachPartition(lambda records: write_partition(records, project_id, collection_name))
    spark.stop()


if __name__ == "__main__":
    main()
