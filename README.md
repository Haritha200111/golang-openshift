# OpenShift Go Demo API

A small in-memory REST API for practising Git, OpenShift Source-to-Image (S2I), builds, deployments, Services and Routes.

## Requirements

- Go 1.23 or newer
- Git
- An OpenShift project / Developer Sandbox

## Run locally

```bash
go version
go run .
```

The server listens on port `8080` by default. You can override it with `PORT`.

## API endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/` | Welcome message |
| GET | `/healthz` | Health check |
| GET | `/api/tasks` | List tasks |
| POST | `/api/tasks` | Create a task |
| PATCH | `/api/tasks/{id}` | Update completion status |

### Try the API

List tasks:

```bash
curl http://localhost:8080/api/tasks
```

Create a task:

```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Deploy my first Go app to OpenShift"}'
```

Mark task 1 complete:

```bash
curl -X PATCH http://localhost:8080/api/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"completed":true}'
```

Check health:

```bash
curl http://localhost:8080/healthz
```

## Push to GitHub

Create an empty GitHub repository, then run these commands from this project directory:

```bash
git init
git add .
git commit -m "Initial commit"
git branch -M main
git remote add origin https://github.com/YOUR-USERNAME/openshift-go-demo.git
git push -u origin main
```

Replace the remote URL with your repository URL.

## Deploy using the OpenShift Developer Console

1. Open your OpenShift project.
2. Go to **Developer → +Add → Import from Git**.
3. Enter your GitHub repository URL and choose the branch.
4. Select the Go builder image / Go runtime if prompted.
5. Choose a builder image whose Go toolchain supports the `go 1.23` directive in `go.mod`.
6. Keep the context directory as `/`.
7. Create the application and watch the build logs.
8. Once deployed, expose the Service using a Route if the console has not created one.
9. Open the Route URL and try `/healthz` and `/api/tasks`.

## Notes

- This demo intentionally has no Dockerfile: it is suitable for learning an S2I source build.
- Tasks are stored in memory. They disappear when the pod restarts and are not shared between replicas.
- The app listens on `0.0.0.0` via `:8080`, which allows the OpenShift Service to reach it.
- If the selected builder image is older than Go 1.23, choose a newer compatible builder image rather than changing the module version arbitrarily.
