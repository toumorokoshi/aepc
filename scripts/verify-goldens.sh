#!/usr/bin/env bash
set -e
./scripts/regenerate-all.sh
if git diff --quiet --exit-code; then
    echo "No differences found."
else
    echo "Differences found.";
    exit 1;
fi