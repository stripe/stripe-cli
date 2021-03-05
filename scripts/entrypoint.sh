#!/usr/bin/env bash

if [ -n "$DOCKER_USERNAME" ] && [ -n "$DOCKER_PASSWORD" ]; then
    echo "Login to the docker..."
    docker login -u $DOCKER_USERNAME -p $DOCKER_PASSWORD $DOCKER_REGISTRY
fi

# Let us write to a secondary GitHub repo for homebrew
if [ -n "$GORELEASER_GITHUB_TOKEN" ] ; then
  export GITHUB_TOKEN=$GORELEASER_GITHUB_TOKEN
fi

goreleaser $@
