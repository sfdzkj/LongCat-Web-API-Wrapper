#!/bin/sh
set -e

mkdir -p /app/data
chown -R 10001:10001 /app/data 2>/dev/null || true
chmod -R u+rwX,g+rwX /app/data 2>/dev/null || true

exec /app/longcat-web-api
