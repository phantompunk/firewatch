#!/bin/sh
set -e
# Ensure the data directory is writable by the app user regardless of how
# Docker initialised the named volume (root-owned volumes are common).
chown appuser:appuser /data
exec su-exec appuser "$@"
