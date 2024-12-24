<br />
<div align="center">
  <a>
    <img src=".github/ganymede-logo.png" alt="Logo" width="80" height="80">
  </a>

  <h2 align="center">Ganymede</h2>

  <p align="center">
    Ganymede is a Twitch VOD and Live Stream archiving platform with a real-time chat experience. Every archive includes a rendered chat for viewing outside of Ganymede. Files are saved in a friendly format allowing for use without Ganymede.
  </p>
</div>

---

## Screenshot

![ganymede-readme_landing](https://github.com/user-attachments/assets/b1c024f5-f5ad-4611-84db-42d599364a74)

https://github.com/user-attachments/assets/184451f1-e3ce-4329-8516-a9842648c01b

## About

Ganymede allows archiving of past streams (VODs) and live streams with a real-time chat playback along with a archival-friendly rendered chat. All files are saved in a friendly way that doesn't require Ganymede to view them (see [file structure](https://github.com/Zibbp/ganymede/wiki/File-Structure)). Ganymede is the successor of [Ceres](https://github.com/Zibbp/Ceres).

## Features

- Realtime Chat Playback
- SSO / OAuth authentication ([wiki](https://github.com/Zibbp/ganymede/wiki/SSO---OpenID-Connect))
- Light/dark mode toggle.
- 'Watched channels'
  - Allows watching channels for archiving past broadcasts and live streams. Includes advanced filtering options.
- Twitch VOD/Livestream support.
- Full VOD, Channel, and User management.
- Custom post-download video FFmpeg parameters.
- Custom chat render parameters.
- Webhook notifications.
- Simple file structure for long-term archival that will outlast Ganymede.
- Recoverable queue system.
- Playback / progress saving.
- Playlists.

## Documentation

For in-depth documentation on features visit the [wiki](https://github.com/Zibbp/ganymede/wiki).

## API

Visit the [docs](https://github.com/Zibbp/ganymede/tree/master/docs) folder for the API docs.

## Installation

### Requirements

- Linux environment with Docker.
- _Optional_ network mounted storage.
- 50gb+ free storage, see [storage requirements](https://github.com/Zibbp/ganymede/wiki/Storage-Requirements).
- A Twitch Application
  - [Create an applicaton](https://dev.twitch.tv/console/apps/create).

### Installation

Ganymede consists of two docker containers:

1. Server
2. Postgres Database

Feel free to use an existing Postgres database container if you don't want to spin new ones up.

1. Download a copy of the `docker-compose.yml` file.
2. Edit the `docker-compose.yml` file modifying the environment variables, see [environment variables](https://github.com/Zibbp/ganymede#environment-variables) for more information.
3. Run `docker compose up -d`.
4. Visit the address and port you specified for the frontend and login with username: `admin` password: `ganymede`.
5. Change the admin password _or_ create a new user, grant admin permissions on that user, and delete the admin user.

### Rootless

The API container can be run as a non root user. To do so add `PUID` and `PGID` environment variables, setting the value to your user. Read [linuxserver's docs](https://docs.linuxserver.io/general/understanding-puid-and-pgid) about this for more information.

Note: On startup the container will `chown` the config, temp, and logs directory. It will not recursively `chown` the `/data/videos` directory. Ensure the mounted `/data/videos` directory is readable by the set user.

### Environment Variables

The `docker-compose.yml` file has comments for each environment variable. Below is a list of all environment variables and their descriptions.

##### Server

| ENV Name                        | Description                                                                                                                   |
| ------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `DEBUG`                         | Enable debug logging `true` or `false`.                                                                                       |
| `VIDEOS_DIR`                    | Path inside the container to the videos directory. Default: `/data/videos`.                                                   |
| `TEMP_DIR`                      | Path inside the container where temporary files are stored during archiving. Default: `/data/temp`.                           |
| `LOGS_DIR`                      | Path inside the container where log files are stored. Default: `/data/logs`.                                                  |
| `CONFIG_DIR`                    | Path inside the container where the config is stored. Default: `/data/config`.                                                |
| `PATH_MIGRATION_ENABLED`        | Enable path migration at startup. Default: `true`.                                                                            |
| `TZ`                            | Timezone.                                                                                                                     |
| `DB_HOST`                       | Host of the database.                                                                                                         |
| `DB_PORT`                       | Port of the database.                                                                                                         |
| `DB_USER`                       | Username for the database.                                                                                                    |
| `DB_PASS`                       | Password for the database.                                                                                                    |
| `DB_NAME`                       | Name of the database.                                                                                                         |
| `DB_SSL`                        | Whether to use SSL. Default: `disable`. See [DB SSL](https://github.com/Zibbp/ganymede/wiki/DB-SSL) for more information.     |
| `DB_SSL_ROOT_CERT`              | _Optional_ Path to DB SSL root certificate. See [DB SSL](https://github.com/Zibbp/ganymede/wiki/DB-SSL) for more information. |
| `TWITCH_CLIENT_ID`              | Twitch application client ID.                                                                                                 |
| `TWITCH_CLIENT_SECRET`          | Twitch application client secret.                                                                                             |
| `OAUTH_ENABLED`                 | _Optional_ Wheter OAuth is enabled `true` or `false`. Must have the other OAuth variables set if this is enabled.             |
| `OAUTH_PROVIDER_URL`            | _Optional_ OAuth provider URL. See https://github.com/Zibbp/ganymede/wiki/SSO---OpenID-Connect                                |
| `OAUTH_CLIENT_ID`               | _Optional_ OAuth client ID.                                                                                                   |
| `OAUTH_CLIENT_SECRET`           | _Optional_ OAuth client secret.                                                                                               |
| `OAUTH_REDIRECT_URL`            | _Optional_ OAuth redirect URL, points to the API. Example: `http://localhost:4000/api/v1/auth/oauth/callback`.                |
| `MAX_CHAT_DOWNLOAD_EXECUTIONS`  | Maximum number of chat downloads that can be running at once. Live streams bypass this limit.                                 |
| `MAX_CHAT_RENDER_EXECUTIONS`    | Maximum number of chat renders that can be running at once.                                                                   |
| `MAX_VIDEO_DOWNLOAD_EXECUTIONS` | Maximum number of video downloads that can be running at once. Live streams bypass this limit.                                |
| `MAX_VIDEO_CONVERT_EXECUTIONS`  | Maximum number of video conversions that can be running at once.                                                              |
| `SHOW_SSO_LOGIN_BUTTON`         | Frontend: `true/false` Show a "login via sso" button on the login page.                                                       |
| `FORCE_SSO_AUTH`                | Frontend: `true/false` Force users to login via SSO by bypassing the login page.                                              |
| `REQUIRE_LOGIN`                 | Frontend: `true/false` Require users to be logged in to view videos.                                                          |

##### DB

**Ensure these are the same in the API environment variables.**

| ENV Name            | Description           |
| ------------------- | --------------------- |
| `POSTGRES_PASSWORD` | Database password     |
| `POSTGRES_USER`     | Database username.    |
| `POSTGRES_DB`       | Name of the database. |

### Volumes

##### API

| Volume         | Description                                                                                                                                                                                       | Example                      |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------- |
| `/data/videos` | Mount for video storage. This **must** match the `VIDEOS_DIR` environment variable.                                                                                                               | `/mnt/nas/vods:/data/videos` |
| `/data/logs`   | Mount to store task logs. This **must** match the `LOGS_DIR` environment variable.                                                                                                                | `./logs:/data/logs`          |
| `/data/temp`   | Mount to store temporary files during the archive process. This is mounted to the host so files are recoverable in the event of a crash. This **must** match the `TEMP_DIR` environment variable. | `./temp:/data/temp`          |
| `/data/config` | Mount to store the config. This **must** match the `CONFIG_DIR` environment variable.                                                                                                             | `./config:/data/config`      |

## Acknowledgements

- [TwitchDownloader](https://github.com/lay295/TwitchDownloader)
- [Streamlink](https://streamlink.github.io/)
- [Chat-Downloader](https://github.com/xenova/chat-downloader)

## License

[GNU General Public License v3.0](https://github.com/Zibbp/ganymede/blob/master/LICENSE)
