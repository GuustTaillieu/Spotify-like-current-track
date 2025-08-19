#!/bin/env bash

SCRIPT_FOLDER="$HOME/.custom_scripts/spotify/utilities"
SCRIPT_GO="main.go"

if ! command -v go >/dev/null; then
  echo "Install go before running this script!"
  exit 1
fi

pushd $SCRIPT_FOLDER

go run $SCRIPT_GO like_current_track
notify-send "$( go run $SCRIPT_GO is_current_track_liked)"

popd
