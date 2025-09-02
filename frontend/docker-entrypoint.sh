#!/bin/sh
set -e

ENV_FILE=/usr/share/nginx/html/env-config.js

{
  echo "// This file is generated at container start. Do not edit."
  echo "window._env_ = {"

  env | awk -F= '/^REACT_APP_/ {print $1}' | while read -r NAME; do
    VAL=$(printenv "$NAME" | sed \
      -e 's/\\/\\\\/g' \
      -e 's/"/\\"/g' \
      -e ':a;N;$!ba;s/\n/\\n/g')
    [ -n "$VAL" ] && printf '  "%s": "%s"\n' "$NAME" "$VAL"
  done | sed '$!s/$/,/'

  echo "};"
} > "$ENV_FILE"

exec "$@"
