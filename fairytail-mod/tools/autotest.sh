#!/usr/bin/env bash
# Headless in-game test: connects a software-rendered client to a dedicated server on localhost:25565
# (start one first with `./gradlew runServer`, op "Dev", online-mode=false), then casts through the real
# item -> server -> packet -> client path and saves screenshots into run/screenshots.
#
#   tools/autotest.sh <key|all> [shots] [camera: first|back|front] [repeat] [nofx: true|false]
set -euo pipefail
cd "$(dirname "$0")/.."
KEY="${1:-aquarius}"; SHOTS="${2:-8,16,28,40,48,54,58,62,64,68,76,90}"; CAM="${3:-first}"; REP="${4:-1}"; NOFX="${5:-false}"
rm -rf run/screenshots
LIBGL_ALWAYS_SOFTWARE=1 MESA_GL_VERSION_OVERRIDE=4.6 MESA_GLSL_VERSION_OVERRIDE=460 \
  xvfb-run -a -s "-screen 0 1280x720x24" \
  ./gradlew runClient -PftTest="$KEY" -PftShots="$SHOTS" -PftCamera="$CAM" -PftRepeat="$REP" -PftNoFx="$NOFX" -PftConnect=localhost:25565 2>&1 \
  | tee run/autotest.log | grep --line-buffered -E "\[autotest\]|Exception in thread|Crash report|FAILURE"
