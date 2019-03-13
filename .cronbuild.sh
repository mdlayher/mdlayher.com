#!/usr/bin/env bash
set -e
set -x
PATH=$PATH:/usr/local/go/bin

cd "$(dirname "$0")"

git pull

# Regenerate data and see if anything changed.
go generate ./...

if [[ -z $(git status -s) ]]; then
    # Nothing to do.
    exit 0
fi

# Content was updated: push a new commit.
git commit -am "mdlayher.com: automatic push of updated content"
git push
