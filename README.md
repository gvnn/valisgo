# valisgo

🧳 A suitcase full of wonders.

The goal of this project is to provide a free and open-source alternative to Nexus and JFrog.

## Supported Protocols

Here is a comparison of protocols currently supported by Valisgo compared to Nexus and JFrog Artifactory, to track what remains to be implemented:

| Protocol / Format | Valisgo |    Nexus     | JFrog |
| ----------------- | :-----: | :----------: | :---: |
| Generic (File)    |   ✅    |      ✅      |  ✅   |
| Go Modules        |   ✅    |      ✅      |  ✅   |
| npm               |   ✅    |      ✅      |  ✅   |
| PyPI              |   ✅    |      ✅      |  ✅   |
| Docker / OCI      |   ❌    |      ✅      |  ✅   |
| Maven / Gradle    |   ❌    |      ✅      |  ✅   |
| NuGet             |   ❌    |      ✅      |  ✅   |
| Helm              |   ❌    |      ✅      |  ✅   |
| RubyGems          |   ❌    |      ✅      |  ✅   |
| APT (Debian)      |   ❌    |      ✅      |  ✅   |
| YUM / RPM         |   ❌    |      ✅      |  ✅   |
| Conan (C/C++)     |   ❌    |      ✅      |  ✅   |
| Cargo (Rust)      |   ❌    | _via plugin_ |  ✅   |
| Composer (PHP)    |   ❌    |      ✅      |  ✅   |
| Swift             |   ❌    |      ❌      |  ✅   |
| CocoaPods         |   ❌    |      ✅      |  ✅   |

## Prerequisites

- **Go**: `1.26.x`
- **[Atlas](https://atlasgo.io/)**: For managing database migrations
- **Docker**: For running PostgreSQL locally

## Quick Start

1. **Install dependencies and tooling** (installs `air` for live-reloading):

    ```bash
    make setup
    ```

2. **Apply database migrations**:

    ```bash
    make migrate-apply
    ```

3. **Seed the database** with initial data:

    ```bash
    make seed
    ```

4. **Start the development server** (with hot-reload):
    ```bash
    make dev
    ```

## CLI Utility

`valisgo` includes a command-line interface (`valisgo-cli`) for interacting with the management API.

To run it directly using Go:

```bash
go run ./cmd/cli --help
```

To build a standalone binary:

```bash
make build-cli
./bin/valisgo-cli --help
```

### Database Migrations (Atlas)

Migrations are generated and applied via Atlas.

- **Generate new migrations** (after changing schema):
    ```bash
    MIGRATION_NAME="add_users_table" make migrate-diff
    ```
    ```bash
    make migrate-apply        # Applies to Postgres
    ```

## Authentication (OIDC + Local Proxy)

`valisgo` does not manage users or passwords itself. Instead it delegates identity to any standard **OIDC provider** (Dex, Keycloak, Auth0, Okta, Google, etc.) — _bring your own OIDC_. Both the server and the CLI validate and refresh tokens against the issuer you configure.

Local development ships with [Dex](https://dexidp.io/) via `docker-compose`, pre-configured with two clients (`valisgo-cli` and `valisgo-web`) and a test user (`admin@valisgo.local` / `password`). Point the issuer flags at your own provider to use anything else.

### How the proxy manages authentication

Package managers (`go`, `npm`, `pip`, …) speak plain HTTP and have no idea how to perform an OIDC browser login or refresh a token. The `valisgo-cli proxy` command bridges that gap: it runs a **local transparent reverse proxy** that injects your credentials into every request on the fly, so your tools never touch a token directly.

```
┌───────────┐     plain HTTP      ┌────────────────────┐   HTTP + Bearer <token>   ┌──────────────────┐
│  go / npm │ ──────────────────► │  valisgo-cli proxy │ ────────────────────────► │ Valisgo registry │
│  / pip    │   localhost:9000    │  (localhost)       │   Authorization: Bearer   │   (upstream)     │
└───────────┘                     └────────────────────┘                           └──────────────────┘
                                          │
                                          ▼
                                  system keyring (refresh token)
                                  + OIDC issuer (token refresh)
```

On each request the proxy:

1. Pulls a valid token from its `oauth2.TokenSource`, transparently refreshing it against the OIDC issuer when expired.
2. Sets `Authorization: Bearer <token>`, preferring the OIDC `id_token`, falling back to the `access_token`.
3. Forwards the request upstream (adding `X-Forwarded-*` headers).

If no valid token can be obtained, the proxy fails fast with `401` and tells you to run `valisgo-cli login` — it never forwards an unauthenticated request.

The server side (`internal/auth`) verifies every incoming request via OIDC middleware, accepting the token from either the `Authorization: Bearer` header (CLI/proxy) or an `access_token` cookie (web UI).

### CLI login flow

```bash
# 1. Authenticate — opens your browser against the OIDC provider
valisgo-cli login

# 2. Start the local proxy (defaults to 0.0.0.0:9000, upstream = --address)
valisgo-cli proxy

# 3. Point your package manager at the proxy, e.g.
GOPROXY=http://localhost:9000 go get ...

# 4. Log out (removes the refresh token from the keyring)
valisgo-cli logout
```

`login` runs the OAuth2 authorization-code flow: it spins up a short-lived local callback server (`http://0.0.0.0:8585/callback`), opens the browser to the provider, validates a CSRF `state` parameter, exchanges the code for tokens, and stores the **refresh token** in your OS keyring (via `go-keyring`). Only the refresh token is persisted; short-lived access/ID tokens are minted on demand by the proxy.

### Configuration

Auth flags are global on the CLI (all have env-var equivalents):

| Flag          | Env var          | Default                               | Purpose                                                            |
| ------------- | ---------------- | ------------------------------------- | ------------------------------------------------------------------ |
| `--issuer`    | `OIDC_ISSUER`    | `http://dex:5556/dex`                 | OIDC issuer URL                                                    |
| `--client-id` | `OIDC_CLIENT_ID` | `valisgo-cli`                         | OAuth2 client ID                                                   |
| `--scopes`    | `OIDC_SCOPES`    | `openid,profile,email,offline_access` | Requested scopes (needs `offline_access` for refresh tokens)       |
| `--token`     | `VALISGO_TOKEN`  | —                                     | Raw access/ID token; bypasses OIDC login + keyring (useful for CI) |

Proxy-specific flags:

| Flag             | Env var                      | Default            | Purpose                  |
| ---------------- | ---------------------------- | ------------------ | ------------------------ |
| `--port`         | `VALISGO_PROXY_PORT`         | `9000`             | Local proxy port         |
| `--bind-address` | `VALISGO_PROXY_BIND_ADDRESS` | `0.0.0.0`          | Local proxy bind address |
| `--upstream`     | —                            | global `--address` | Registry to forward to   |

The server reads its own OIDC config from `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URL`, and `OIDC_SCOPES`.

## Authorization (Casbin)

`valisgo` uses [Casbin](https://casbin.org/) (via `gorm-adapter`) to handle Role-Based Access Control (RBAC).
