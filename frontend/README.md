# aipm (frontend)

## Install the dependencies

```bash
pnpm install
# or: yarn/npm/bun install
```

### Start the app in development mode (HMR, error reporting, etc.)

```bash
pnpm dev
```

The development server is configured to run at `http://localhost:8000`.

When finishing app-affecting work, leave both the backend and Quasar development
servers running so the current UI can be tested immediately. Run the backend
from `backend/` with `go run ./cmd/server` and run Quasar from `frontend/` with
`pnpm dev`.

### Build the app for production

```bash
quasar build
```

### Customize the configuration

See [Configuring quasar.config.js](https://v2.quasar.dev/quasar-cli-vite/quasar-config-js).
