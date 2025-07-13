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

Visit the [docs](https://github.com/Zibbp/ganymede/tree/main/docs) folder for the API docs.

## Translations

See the [messages](https://github.com/Zibbp/ganymede/tree/main/frontend/messages) directory for available translations. If you would like to add a new translation, please create a pull request with the new translation file. The file should be named `<language>.json` where `<language>` is the language code (e.g. `de.json` for German). Additionally the language needs to be added to the navbar in the `frontend/app/layout/Navbar.tsx` file in the `languages` array. Use the `frontend/translation-coverage.js` script to see what has been translated or with the `-u` option to populate missing keys in translation files. 

## Installation

### Requirements

- Linux environment with Docker.
- _Optional_ network mounted storage.
- 50gb+ free storage, see [storage requirements](https://github.com/Zibbp/ganymede/wiki/Storage-Requirements).
- A Twitch Application
  - [Create an application](https://dev.twitch.tv/console/apps/create).

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

### Config

A configuration file is generate on initial start of Ganymede. By default the configuration is at `/data/config/config.json`. See the [config.go](https://github.com/Zibbp/ganymede/blob/main/internal/config/config.go) file for a full list of configuration settings. Most of the settings can be configured in the Web UI by navigating to Admin > Settings.

### Environment Variables

The `docker-compose.yml` file has comments for each environment variable. Below is a list of all environment variables and their descriptions. See the [env.go](https://github.com/Zibbp/ganymede/blob/main/internal/config/env.go) file for a full list of all environment variables and their default values.

##### Server

| ENV Name                                | Description                                                                                                                     |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `DEBUG`                                 | Enable debug logging `true` or `false`.                                                                                         |
| `VIDEOS_DIR`                            | Path inside the container to the videos directory. Default: `/data/videos`.                                                     |
| `TEMP_DIR`                              | Path inside the container where temporary files are stored during archiving. Default: `/data/temp`.                             |
| `LOGS_DIR`                              | Path inside the container where log files are stored. Default: `/data/logs`.                                                    |
| `CONFIG_DIR`                            | Path inside the container where the config is stored. Default: `/data/config`.                                                  |
| `PATH_MIGRATION_ENABLED`                | Enable path migration at startup. Default: `true`.                                                                              |
| `TZ`                                    | Timezone.                                                                                                                       |
| `DB_HOST`                               | Host of the database.                                                                                                           |
| `DB_PORT`                               | Port of the database.                                                                                                           |
| `DB_USER`                               | Username for the database.                                                                                                      |
| `DB_PASS`                               | Password for the database.                                                                                                      |
| `DB_NAME`                               | Name of the database.                                                                                                           |
| `DB_SSL`                                | Whether to use SSL. Default: `disable`. See [DB SSL](https://github.com/Zibbp/ganymede/wiki/DB-SSL) for more information.       |
| `DB_SSL_ROOT_CERT`                      | _Optional_ Path to DB SSL root certificate. See [DB SSL](https://github.com/Zibbp/ganymede/wiki/DB-SSL) for more information.   |
| `TWITCH_CLIENT_ID`                      | Twitch application client ID.                                                                                                   |
| `TWITCH_CLIENT_SECRET`                  | Twitch application client secret.                                                                                               |
| `OAUTH_ENABLED`                         | _Optional_ Wether OAuth is enabled `true` or `false`. Must have the other OAuth variables set if this is enabled.               |
| `OAUTH_PROVIDER_URL`                    | _Optional_ OAuth provider URL. See https://github.com/Zibbp/ganymede/wiki/SSO---OpenID-Connect                                  |
| `OAUTH_CLIENT_ID`                       | _Optional_ OAuth client ID.                                                                                                     |
| `OAUTH_CLIENT_SECRET`                   | _Optional_ OAuth client secret.                                                                                                 |
| `OAUTH_REDIRECT_URL`                    | _Optional_ OAuth redirect URL, points to the API. Example: `http://localhost:4000/api/v1/auth/oauth/callback`.                  |
| `MAX_CHAT_DOWNLOAD_EXECUTIONS`          | Maximum number of chat downloads that can be running at once. Live streams bypass this limit.                                   |
| `MAX_CHAT_RENDER_EXECUTIONS`            | Maximum number of chat renders that can be running at once.                                                                     |
| `MAX_VIDEO_DOWNLOAD_EXECUTIONS`         | Maximum number of video downloads that can be running at once. Live streams bypass this limit.                                  |
| `MAX_VIDEO_CONVERT_EXECUTIONS`          | Maximum number of video conversions that can be running at once.                                                                |
| `MAX_VIDEO_SPRITE_THUMBNAIL_EXECUTIONS` | Maximum number of video sprite thumbnail generation jobs that can be running at once. This is not very CPU intensive.           |
| `SHOW_SSO_LOGIN_BUTTON`                 | Frontend: `true/false` Show a "login via sso" button on the login page (defaults to false).                                     |
| `FORCE_SSO_AUTH`                        | Frontend: `true/false` Force users to login via SSO by bypassing the login page (defaults to false).                            |
| `REQUIRE_LOGIN`                         | Frontend: `true/false` Require users to be logged in to view videos (defaults to false).                                        |
| `SHOW_LOCALE_BUTTON`                    | Frontend: `true/false` Show the locale/language button on the navbar (defaults to true).                                        |
| `DEFAULT_LOCALE`                        | Frontend: Sets the default locale/language. Must be the short code of the language. Example: `en` for English, `de` for German. |
| `FORCE_LOGIN`                           | Frontend: `true/false` Force require users to login to view any page (defaults to false).                                       |

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


## Development

A [devcontainer](https://containers.dev/) is included for development. This container includes all the necessary tools to develop Ganymede. Once setup, the [Makefile](/Makefile) can be used to run the development environment.

- `make dev_server` - Starts the server.
- `make dev_worker` - Starts the worker.
- `make dev_web` - Starts the web server.

View the [Makefile](/Makefile) for more commands.

## Acknowledgements

- [TwitchDownloader](https://github.com/lay295/TwitchDownloader)
- [Streamlink](https://streamlink.github.io/)
- [Chat-Downloader](https://github.com/xenova/chat-downloader)

## License

[GNU General Public License v3.0](https://github.com/Zibbp/ganymede/blob/main/LICENSE)
