# Developer Onboarding Checklist

Quickstart checklist for new contributors. Run through top to bottom — each step builds on the previous.

For the detailed walkthrough, see [DEVELOPER_ONBOARDING.md](docs/DEVELOPER_ONBOARDING.md).

## Automated Setup

Run the bootstrap script to handle steps 1–5 automatically:

```bash
./scripts/bootstrap.sh
```

To verify your environment without making changes:

```bash
./scripts/bootstrap.sh --check
```

---

## Manual Checklist

### Step 1: Install Tools (~5 min)

| Tool    | Version          | Verify                         |
| ------- | ---------------- | ------------------------------ |
| Go      | 1.24+            | `go version`                   |
| Node.js | 20+ LTS          | `node --version`               |
| Docker  | 24+ (Compose v2) | `docker compose version`       |
| Make    | any              | `make --version`               |
| libvips | 8.15+ (optional) | `pkg-config --modversion vips` |

- [ ] All tools installed and version-verified

### Step 2: Clone & Configure (~3 min)

```bash
git clone git@github.com:subculture-collective/subcults.git
cd subcults
cp configs/dev.env.example configs/dev.env
```

Edit `configs/dev.env` and fill in required values:

| Variable             | Required      | How to Get                                             |
| -------------------- | ------------- | ------------------------------------------------------ |
| `DATABASE_URL`       | Yes           | Use Docker Compose default or Neon dashboard           |
| `JWT_SECRET_CURRENT` | Yes           | `openssl rand -base64 32`                              |
| `MAPTILER_API_KEY`   | For maps      | [maptiler.com](https://cloud.maptiler.com/) free tier  |
| `LIVEKIT_*`          | For streaming | [livekit.io](https://livekit.io/) dashboard            |
| `STRIPE_*`           | For payments  | [Stripe test mode](https://dashboard.stripe.com/test/) |
| `R2_*`               | For media     | Cloudflare R2 dashboard                                |

- [ ] `configs/dev.env` created and secrets filled

### Step 3: Install Dependencies (~2 min)

```bash
go mod download
cd web && npm ci && cd ..
```

- [ ] Go modules downloaded
- [ ] Frontend `node_modules` installed

### Step 4: Start Infrastructure (~1 min)

```bash
make compose-up     # Starts Postgres 16 + PostGIS on localhost:5439
```

Wait for health check:

```bash
docker compose exec postgres pg_isready -U subcults
```

- [ ] Postgres running and healthy

### Step 5: Run Migrations (~1 min)

```bash
export DATABASE_URL="postgres://subcults:subcults@localhost:5439/subcults?sslmode=disable"
make migrate-up
```

- [ ] All migrations applied (30+ tables)

### Step 6: Start Development Servers (~1 min)

```bash
make dev            # Starts API (8080) + frontend (5173)
```

Or individually:

```bash
make dev-api        # Go API on port 8080
make dev-frontend   # Vite on port 5173
```

- [ ] API running at `http://localhost:8080`
- [ ] Frontend running at `http://localhost:5173`

### Step 7: Verify (~2 min)

```bash
# Health check
curl -s http://localhost:8080/health | jq .

# Run tests
make test
```

- [ ] Health endpoint returns `200 OK`
- [ ] Tests pass

---

## Key Resources

| Resource                | Path                                                                   |
| ----------------------- | ---------------------------------------------------------------------- |
| Full onboarding guide   | [docs/DEVELOPER_ONBOARDING.md](docs/DEVELOPER_ONBOARDING.md)           |
| Backend development     | [docs/BACKEND_DEVELOPMENT_GUIDE.md](docs/BACKEND_DEVELOPMENT_GUIDE.md) |
| Testing guide           | [docs/TESTING_GUIDE.md](docs/TESTING_GUIDE.md)                         |
| Architecture            | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)                           |
| Contributing            | [CONTRIBUTING.md](CONTRIBUTING.md)                                     |
| Style guide             | [STYLE_GUIDE.md](STYLE_GUIDE.md)                                       |
| Configuration reference | [docs/CONFIGURATION.md](docs/CONFIGURATION.md)                         |
| Docker guide            | [docs/docker.md](docs/docker.md)                                       |

## Troubleshooting

| Problem              | Fix                                                                           |
| -------------------- | ----------------------------------------------------------------------------- |
| Port 5439 in use     | `docker compose down` then retry, or change port in `docker-compose.yml`      |
| Migration fails      | Verify `DATABASE_URL` is set and Postgres is running                          |
| Docker out of memory | Increase Docker Desktop memory to 4GB+                                        |
| `libvips` not found  | Install via `brew install vips` (macOS) or `apt install libvips-dev` (Ubuntu) |
| Frontend build fails | Delete `web/node_modules` and `npm ci` again                                  |
| Go build fails       | Run `go mod tidy` then `go mod download`                                      |

**Total estimated setup time: ~15 minutes** (with external service accounts pre-created).
