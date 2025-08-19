#!/bin/env bash

SCRIPT_FOLDER="$HOME/.custom_scripts/spotify/utilities"
SCRIPT_GO="main.go"
SCRIPT_EXE="main"

if ! command -v go >/dev/null; then
  echo "Install go before running this script!"
  exit 1
fi

pushd $SCRIPT_FOLDER

if [ ! -e $SCRIPT_EXE ]; then
  go build $SCRIPT_GO
  chmod +x $SCRIPT_EXE
fi

$SCRIPT_EXE like_current_track
notify-send "$( $SCRIPT_EXE is_current_track_liked)"

popd
