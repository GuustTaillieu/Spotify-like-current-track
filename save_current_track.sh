#!/bin/env bash

SCRIPT="$HOME/.custom_scripts/spotify/save_current_track/main.go"

if ! command -v go >/dev/null; then
  echo "Install go before running this script!"
  exit 1
fi

if ! go run $SCRIPT; then
  exit 1
fi
