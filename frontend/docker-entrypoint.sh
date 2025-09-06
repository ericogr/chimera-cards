#!/bin/sh
set -e

ENV_FILE=/usr/share/nginx/html/env-config.js

echo "Generating $ENV_FILE from environment variables (REACT_APP_*)"
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

# Warn in container logs if required runtime keys are missing (names only)
REQUIRED_KEYS="REACT_APP_GOOGLE_CLIENT_ID REACT_APP_API_BASE_URL"
MISSING_KEYS=""
for K in $REQUIRED_KEYS; do
  if [ -z "$(printenv $K || true)" ]; then
    MISSING_KEYS="$MISSING_KEYS $K"
  fi
done
if [ -n "$MISSING_KEYS" ]; then
  echo "WARNING: frontend runtime variables missing:$MISSING_KEYS" >&2
fi

exec "$@"
