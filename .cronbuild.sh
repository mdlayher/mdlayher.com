#!/usr/bin/env bash
set -x
PATH=$PATH:/usr/local/bin:/usr/local/go/bin

cd "$(dirname "$0")"

git pull

# Regenerate static content and blog.
go generate ./...
hugo

if [[ -z $(git status -s) ]]; then
    # Nothing to do.
    exit 0
fi

# Content was updated: push a new commit.
git commit -am "mdlayher.com: automatic push of updated content"
git push
