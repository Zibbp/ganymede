#!/bin/sh
# Exit supervisord if any program reaches FATAL state
# This allows Docker or other orchestration to restart the container
# after a fatal error occurs.
set -eu

SUPERV_PIDFILE=${SUPERV_PIDFILE:-/var/run/supervisord.pid}
SUPERV_CONF=${SUPERV_CONF:-/opt/app/supervisord.conf}

while :; do
  printf "READY\n"

  IFS= read -r HEADER || exit 0
  LEN=$(printf "%s" "$HEADER" | awk -F'len:' '{print $2+0}')
  if [ "${LEN:-0}" -gt 0 ]; then dd bs=1 count="$LEN" >/dev/null 2>&1; fi

  printf "RESULT 2\nOK"

  echo "[exit_on_failure] Fatal detected — attempting shutdown" >&2

  # Attempt to gracefully shutdown supervisord in several ways
  if command -v supervisorctl >/dev/null 2>&1; then
    supervisorctl -c "$SUPERV_CONF" shutdown >/dev/null 2>&1 || true
  fi

  # SIGTERM/SIGKILL supervisord if still running
  if [ -f "$SUPERV_PIDFILE" ]; then
    PID="$(cat "$SUPERV_PIDFILE" 2>/dev/null || echo "")"
    if [ -n "${PID}" ] && [ -d "/proc/$PID" ]; then
      echo "[exit_on_failure] SIGTERM supervisord pid=$PID" >&2
      kill -TERM "$PID" 2>/dev/null || true
      # wait briefly, then SIGKILL if still alive
      for i in 1 2 3; do
        [ ! -d "/proc/$PID" ] && break
        sleep 0.2
      done
      if [ -d "/proc/$PID" ]; then
        echo "[exit_on_failure] SIGKILL supervisord pid=$PID" >&2
        kill -KILL "$PID" 2>/dev/null || true
      fi
    fi
  fi

  # Fallback: kill parent process if still running
  PPID="$(awk '/PPid:/{print $2}' /proc/$$/status 2>/dev/null || echo "")"
  if [ -n "$PPID" ] && [ -d "/proc/$PPID" ]; then
    echo "[exit_on_failure] fallback SIGTERM ppid=$PPID" >&2
    kill -TERM "$PPID" 2>/dev/null || true
  fi

  sleep 0.2
  exit 0
done
