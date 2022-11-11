<br />
<div align="center">
  <a>
    <img src=".github/ganymede-logo.png" alt="Logo" width="80" height="80">
  </a>

  <h2 align="center">Ganymede</h2>

  <p align="center">
    Twitch VOD and Stream archiving platform with a rendered chat. Files are saved in a friendly format allowing for use without Ganymede.
  </p>
</div>

---

## Demo

![landing-demo](.github/landing-demo.jpg)

https://user-images.githubusercontent.com/21207065/180067579-674af497-090f-4e07-9c81-0314c6361a87.mp4

## About

Ganymede allows archiving of past streams (VODs) and livestreams both with a rendered chat. All files are saved in a friendly way that doesn't require Ganymede to view them (see [file structure](https://github.com/Zibbp/ganymede/wiki/File-Structure)). Ganymede is the successor of [Ceres](https://github.com/Zibbp/Ceres).

## Features

- SSO / OAuth authentication
- Light/dark mode toggle.
- Twitch VOD/Livestream support.
- Queue holds.
- Queue task restarts.
- Full VOD, Channel, and User management.
- Custom post-download video FFmpeg parameters.
- Custom chat render parameters.
- Webhook notifications.


## Documentation

For in-depth documentation on features visit the [wiki](https://github.com/Zibbp/ganymede/wiki).

## API

Visit the [docs](https://github.com/Zibbp/ganymede/tree/master/docs) folder for the API docs.

## Installation

### Requirements

* Linux environment with Docker.
* *Optional* network mounted storage.
* 50gb+ free storage, see [storage requirements](https://github.com/Zibbp/ganymede/wiki/Storage-Requirements).
* A Twitch Application
  * [Create an applicaton](https://dev.twitch.tv/console/apps/create).
  
### Installation

Ganymede consists of four docker containers:

1. API
2. Frontend
3. Postgres Database
4. Nginx

Feel free to use an existing Postgres database container and Nginx container if you don't want to spin new ones up.

1. Download a copy of the `docker-compose.yml` file and `nginx.conf`.
2. Edit the `docker-compose.yml` file modifying the environment variables, see [environment variables](https://github.com/Zibbp/ganymede#environment-variables).
3. Run `docker compose up -d`.
4. Visit the address and port you specified for the frontend and login with username: `admin` password: `ganymede`.
5. Change the admin password *or* create a new user, grant admin permissions on that user, and delete the admin user.

### Environment Variables

##### API

| ENV Name               | Description                                                                                                                                                     |
|------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `DB_HOST`              | Host of the database.                                                                                                                                           |
| `DB_PORT`              | Port of the database.                                                                                                                                           |
| `DB_USER`              | Username for the database.                                                                                                                                      |
| `DB_PASS`              | Password for the database.                                                                                                                                      |
  | `DB_NAME`              | Name of the database.                                                                                                                                           |
| `DB_SSL`               | Whether to use SSL. Default: `disable`. See [DB SSL](https://github.com/Zibbp/ganymede/wiki/DB-SSL) for more information.                                       |
| `DB_SSL_ROOT_CERT`     | *Optional* Path to DB SSL root certificate. See [DB SSL](https://github.com/Zibbp/ganymede/wiki/DB-SSL) for more information.                                   
| `JWT_SECRET`           | Secret for JWT tokens.                                                                                                                                          |
| `JWT_REFRESH_SECRET`   | Secret for JWT refresh tokens.                                                                                                                                  |
| `TWITCH_CLIENT_ID`     | Twitch application client ID.                                                                                                                                   |
| `TWITCH_CLIENT_SECRET` | Twitch application client secret.                                                                                                                               |
| `FRONTEND_HOST`        | Host of the frontend, used for CORS. Example: `http://192.168.1.2:4801`                                                                                         |
| `COOKIE_DOMAIN`        | *Optional* Base domain for cookies. Used when reverse proxying. See [reverse proxy](https://github.com/Zibbp/ganymede/wiki/Reverse-Proxy) for more information. 
| `OAUTH_PROVIDER_URL`   | *Optional* OAuth provider URL. See https://github.com/Zibbp/ganymede/wiki/SSO---OpenID-Connect                                                                                                                             |
| `OAUTH_CLIENT_ID`      | *Optional* OAuth client ID.                                                                                                                                     |
| `OAUTH_CLIENT_SECRET`  | *Optional* OAuth client secret.                                                                                                                                 |
| `OAUTH_REDIRECT_URL`   | *Optional* OAuth redirect URL, points to the API. Example: `http://localhost:4000/api/v1/auth/oauth/callback`.                                                  |                                                                    |

##### Frontend

| ENV Name              | Description                                                     |
|-----------------------|-----------------------------------------------------------------|
| `NUXT_PUBLIC_API_URL` | Host for the API. Example: `http://192.168.1.2:4800`.           |
| `NUXT_PUBLIC_CDN_URL` | Host for the Nginx serivce. Example: `http://197.148.1.2:4802`. |

##### DB

**Ensure these are the same in the API environment variables.**

| ENV Name            | Description           |
|---------------------|-----------------------|
| `POSTGRES_PASSWORD` | Database password     |
| `POSTGRES_USER`     | Database username.    |
| `POSTGRES_DB`       | Name of the database. |

### Volumes

##### API

| Volume  | Description                                                                     | Example                 |
|---------|---------------------------------------------------------------------------------|-------------------------|
| `/vods` | Mount for VOD storage. This example I have my NAS mounted to `/mnt/vault/vods`. | `/mnt/vault/vods:/vods` |
| `/logs` | Queue log folder.                                                               | `./logs:/logs`          |
| `/data` | Config folder.                                                                  | `./data:/data`          |

**Optional**

`./tmp:/tmp` Binding the `tmp` folder prevents lost data if the container crashes as temporary downloads are stored in `tmp` which gets flushed when the container stops.

##### Nginx

| Volume                     | Description                                    | Example                                        |
|----------------------------|------------------------------------------------|------------------------------------------------|
| `/mnt/vods`                | VOD storage, same as the API container volume. | `/mnt/vault/vods:/mnt/vods`                    |
| `/etc/nginx/nginx.conf:ro` | Path to the Nginx conf file.                   | `/path/to/nginx.conf:/etc/nginx/nginx.conf:ro` |


## Acknowledgements

 - [TwitchDownloader](https://github.com/lay295/TwitchDownloader)
 - [Streamlink](https://streamlink.github.io/)
 - [Chat-Downloader](https://github.com/xenova/chat-downloader)
 
 ## License

[GNU General Public License v3.0](https://github.com/Zibbp/ganymede/blob/master/LICENSE)

## Authors

- [@Zibbp](https://www.github.com/Zibbp)
