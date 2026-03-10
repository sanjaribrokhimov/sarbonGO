#!/usr/bin/env bash
# Pull (fetch + merge) from both remotes: origin (sanjaribrokhimov) and itgenius (itgenius24).
set -e
branch=$(git branch --show-current)
echo "Fetching from origin and itgenius..."
git fetch origin
git fetch itgenius
echo "Merging origin/$branch into $branch..."
git merge "origin/$branch" --no-edit
echo "Done. (To also merge itgenius: git merge itgenius/$branch)"
