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

# set permissions
chown -R abc:abc /logs
chown -R abc:abc /data
chown -R abc:abc /tmp
chown abc:abc /vods

# fonts
mkdir -p /var/cache/fontconfig
chown abc:abc /var/cache/fontconfig
su-exec abc fc-cache -f

# dotnet envs
export DOTNET_BUNDLE_EXTRACT_BASE_DIR=/tmp
export FONTCONFIG_CACHE=/var/cache/fontconfig

su-exec abc /opt/app/ganymede-api