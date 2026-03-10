#!/usr/bin/env bash
# Push current branch to both repos (origin has two push URLs).
set -e
branch=$(git branch --show-current)
echo "Pushing $branch to both repos..."
git push origin "$branch"
echo "Done (pushed to sanjaribrokhimov/sarbonGO and itgenius24/Sarbon-backend)."
