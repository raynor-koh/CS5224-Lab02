from collections import Counter
from pathlib import Path
import csv

edges = set()
invalid = 0

for line in Path("data/twitter_combined.txt").open(encoding="utf-8"):
    fields = line.split()
    if len(fields) == 2:
        edges.add(tuple(fields))  # (follower, followee)
    else:
        invalid += 1

followers = Counter(followee for _, followee in edges)
followees = Counter(follower for follower, _ in edges)

print("distinct edges:", len(edges))
print("invalid lines:", invalid)
print("users:", len(set(followers) | set(followees)))

with open("data/expected_user_stats.csv", "w", newline="", encoding="utf-8") as file:
    writer = csv.DictWriter(
        file,
        fieldnames=["userId", "followers", "followees"],
    )
    writer.writeheader()

    for user_id in sorted(set(followers) | set(followees)):
        writer.writerow(
            {
                "userId": user_id,
                "followers": followers[user_id],
                "followees": followees[user_id],
            }
        )
