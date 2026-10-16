# API Docs

Swagger API docs can be found at `http://IP:PORT/swagger/index.html`

## Authentication

Programmatic clients should use a scoped API key created by an administrator in the web UI. Send the complete key in the `Authorization` header:

```http
Authorization: Bearer gym_{prefix}_{secret}
```

The complete key is shown only once when it is created. In Swagger UI, select **Authorize** and enter the complete value, including the `Bearer` prefix.

The Ganymede web UI uses a separate HTTP-only session cookie. Signing in through the web UI or `POST /api/v1/auth/login` establishes the session, and the browser sends it automatically with same-origin requests. Browser sessions are not external API credentials and cannot be entered in Swagger's **Authorize** dialog. Browser-only operations, including API-key management and per-user playback state, require this interactive session and do not accept API keys.
