# Next-Gen Self-Hosted Stack (2025)

This project is designed to be deployed using the latest "Next-Gen" self-hosted tools, focusing on simplicity, ownership, and Developer Experience (DX).

## 1. Deployment & Infrastructure (IaC)
**Tool: [Coolify](https://coolify.io/)**

Forget complex Kubernetes configs or expensive AWS bills. Coolify is an open-source, self-hostable Heroku/Vercel alternative.

- **Why?**
  - **Push-to-Deploy**: Connect your Git repo, push code, and it builds.
  - **Self-Hosted**: Runs on your own VPS (Hetzner, DigitalOcean, etc.).
  - **Docker Native**: Manages `docker-compose.yml` automatically.
  - **Preview Environments**: Auto-deploy Pull Requests.

**Setup:**
1. Install Coolify on a VPS: `curl -fsSL https://cdn.coollabs.io/coolify/install.sh | bash`
2. Connect this repository.
3. Coolify detects the `deploy/docker-compose.yml` or `Dockerfile`.

## 2. Secret Management
**Tool: [Infisical](https://infisical.com/)**

Stop storing `.env` files in Slack or sharing them insecurely. Infisical is the open-source secret management platform.

- **Why?**
  - **End-to-End Encrypted**: Secure by default.
  - **GitOps**: Sync secrets to your infrastructure (Coolify, Kubernetes, Docker).
  - **Developer Experience**: CLI tool (`infisical run -- go run main.go`) injects secrets locally.
  - **Self-Hostable**: Run it alongside your app in Coolify.

**Integration:**
- **Local**: Use `infisical init` to sync secrets.
- **Prod**: Infisical Operator syncs secrets directly to Coolify environment variables.

## 3. Database
**Tool: [SurrealDB](https://surrealdb.com/)**

- **Why?**
  - **Multi-Model**: Relational, Graph, Document in one.
  - **Real-Time**: WebSocket subscriptions built-in (no need for Pusher).
  - **Auth Built-in**: Handle users directly in the DB.

## 4. Observability
**Tool: [OpenObserve](https://openobserve.ai/)** (or SigNoz)

- **Why?**
  - **Rust-based**: High performance, low resource usage (unlike ELK).
  - **Full Stack**: Logs, Metrics, Traces in one UI.
  - **Drop-in Replacement**: Compatible with Elastic/Prometheus APIs.

## Recommended Deployment Flow

1. **Secrets**: Developer adds secret to Infisical UI.
2. **Code**: Developer pushes to `main` branch.
3. **CI/CD**: 
   - Infisical injects secrets into the build process.
   - Coolify pulls the new code, builds the Docker image (using `apps/backend/Dockerfile`).
   - Coolify rolls out the update with zero downtime.

