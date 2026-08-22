#!/bin/bash
PUID=${PUID:-911}
PGID=${PGID:-911}

groupmod -o -g "$PGID" abc
usermod -o -u "$PUID" abc

echo "
-------------------------------------
User uid:    $(id -u abc)
User gid:    $(id -g abc)
-------------------------------------
"

# define default directories
LOGS_DIR=${LOGS_DIR:-"/data/logs"}
CONFIG_DIR=${CONFIG_DIR:-"/data/config"}
VIDEOS_DIR=${VIDEOS_DIR:-"/data/videos"}
TEMP_DIR=${TEMP_DIR:-"/data/temp"}
SKIP_CHOWN=${SKIP_CHOWN:-false}

# set permissions
# SKIP_CHOWN can be set to true to skip this, useful for network storage (NFS/SMB)
# that does not support chown or POSIX permissions
if [ "${SKIP_CHOWN}" != "true" ]; then
  chown -R abc:abc "${LOGS_DIR}"
  chown -R abc:abc "${CONFIG_DIR}"
  chown -R abc:abc "${TEMP_DIR}"
  chown abc:abc "${VIDEOS_DIR}"
else
  echo "Skipping chown because SKIP_CHOWN=true"
fi

# fonts
mkdir -p /var/cache/fontconfig
chown abc:abc /var/cache/fontconfig
gosu abc fc-cache -f

# dotnet envs
export DOTNET_BUNDLE_EXTRACT_BASE_DIR=/tmp
export FONTCONFIG_CACHE=/var/cache/fontconfig

# start applications
/usr/bin/supervisord -c /opt/app/supervisord.conf