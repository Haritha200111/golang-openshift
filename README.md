# Go Task API on OpenShift

A learning project that deploys a Go REST API and PostgreSQL to OpenShift using explicit, version-controlled manifests.

> This is a production-style learning baseline, not a fully production-ready database platform. It uses a single PostgreSQL replica and does not include backups, HA/failover, migrations, external secret management, or image vulnerability scanning.

## API

- `GET /` — service information
- `GET /healthz` — process liveness
- `GET /readyz` — checks PostgreSQL connectivity
- `GET /api/tasks` — list tasks
- `POST /api/tasks` — create a task, JSON: `{"title":"Learn OpenShift Routes"}`
- `PATCH /api/tasks/{id}` — update completion, JSON: `{"completed":true}`

## 1. Local validation (PowerShell)

```powershell
$env:GOTOOLCHAIN = "local"
go mod tidy
go test ./...
go build ./...
```

For local database testing, provide `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, and `DB_PASSWORD`. The API waits up to about one minute for PostgreSQL on startup.

## 2. Build and push the container image

Choose a registry/repository your OpenShift cluster can pull from. Example using Docker Hub:

```powershell
$env:IMAGE="docker.io/YOUR_DOCKERHUB_USER/golang-openshift:1.0.0"
docker build --platform linux/amd64 -t $env:IMAGE .
docker login
docker push $env:IMAGE
```

If Docker Desktop is unavailable, build with Podman or use an OpenShift build strategy. Do not continue until the image is available to the cluster. For a private registry, configure an image pull secret and link it to the appropriate ServiceAccount.

## 3. Select your project

```powershell
oc whoami
oc project
oc get project
```

Use your existing project or create one if you have permission:

```powershell
oc new-project YOUR_PROJECT
```

Do not create a new project if you are expected to use the assigned sandbox project.

## 4. Create the database Secret

Run this in PowerShell. Replace the sample password with a strong unique value; don't commit it.

```powershell
oc create secret generic tasks-db `
  --from-literal=username=tasks_user `
  --from-literal=password='REPLACE_WITH_A_STRONG_PASSWORD'
```

If you need to change credentials later, update the Secret and restart the database/API as appropriate. PostgreSQL's already-initialized data directory may retain the original database password; changing a Kubernetes Secret alone does not rotate the database user's password.

## 5. Deploy PostgreSQL

```powershell
oc apply -f openshift/config/configmap.yaml
oc apply -f openshift/database/pvc.yaml
oc apply -f openshift/database/service.yaml
oc apply -f openshift/database/statefulset.yaml

oc get pvc
oc get statefulset,pods,svc
oc describe pvc postgres-data
oc logs statefulset/postgres
```

Wait for the PVC to bind and the PostgreSQL pod to become Ready:

```powershell
oc wait --for=jsonpath='{.status.phase}'=Bound pvc/postgres-data --timeout=180s
oc rollout status statefulset/postgres --timeout=300s
```

The StorageClass may use `WaitForFirstConsumer`, so the PVC can remain Pending until the pod is scheduled. Check `oc describe pvc postgres-data` and pod events if it does not bind.

The example uses `quay.io/sclorg/postgresql-15-c9s:latest`, an OpenShift-compatible community image. For repeatable production use, pin a tested image digest or approved registry image after validating availability and support policy.

## 6. Build and push the Go API image

```powershell
$env:IMAGE="docker.io/YOUR_DOCKERHUB_USER/golang-openshift:1.0.0"
docker build --platform linux/amd64 -t $env:IMAGE .
docker push $env:IMAGE
```

Edit `openshift/application/deployment.yaml` and replace `quay.io/YOUR_ORG/golang-openshift:1.0.0` with the exact image you pushed.

## 7. Deploy the API and expose it

```powershell
oc apply -f openshift/security/service-account.yaml
oc apply -f openshift/application/deployment.yaml
oc apply -f openshift/application/service.yaml
oc apply -f openshift/application/route.yaml
oc apply -f openshift/networking/network-policy.yaml

oc rollout status deployment/task-api --timeout=180s
oc get pods -l app=task-api
oc get svc task-api
oc get route task-api
oc describe route task-api
```

If the NetworkPolicy causes the route to fail, inspect the router namespace and labels in your cluster. Some sandbox network plugins or policies may require a platform-specific ingress allowance. Do not blindly remove security controls; inspect events and test the router-to-pod path.

## 8. Test the API

Get the route hostname:

```powershell
$HOSTNAME = oc get route task-api -o jsonpath='{.spec.host}'
$BASE = "https://$HOSTNAME"
```

Then:

```powershell
Invoke-RestMethod "$BASE/"
Invoke-RestMethod "$BASE/healthz"
Invoke-RestMethod "$BASE/readyz"

$task = Invoke-RestMethod -Method Post `
  -Uri "$BASE/api/tasks" `
  -ContentType "application/json" `
  -Body '{"title":"Learn OpenShift Routes"}'

Invoke-RestMethod "$BASE/api/tasks"

Invoke-RestMethod -Method Patch `
  -Uri "$BASE/api/tasks/$($task.id)" `
  -ContentType "application/json" `
  -Body '{"completed":true}'
```

## 9. OpenShift troubleshooting commands

```powershell
oc get all
oc get events --sort-by=.lastTimestamp
oc describe pod -l app=task-api
oc logs deployment/task-api
oc logs statefulset/postgres
oc get endpoints task-api postgres
oc get networkpolicy
oc get pvc
oc describe pvc postgres-data
oc get pods -o wide
```

## 10. Persistence test

1. Create a task through the API.
2. Record its ID.
3. Delete the PostgreSQL pod (not the PVC): `oc delete pod postgres-0`.
4. Wait for the StatefulSet to recreate it: `oc rollout status statefulset/postgres`.
5. Query `/api/tasks` again and verify the task remains.

Never delete `postgres-data` for this test. Deleting a PVC can delete the underlying volume depending on the StorageClass reclaim policy.

## Important production gaps to address before real production

- Database backups and tested restores; storage persistence is not a backup.
- PostgreSQL HA, replication, failover and disaster recovery.
- Schema migrations as a controlled release step instead of creating tables at every API startup.
- Managed secrets / external secret integration and credential rotation.
- Image scanning, SBOM, signed images and immutable digests.
- TLS policy, authentication/authorization and API rate limiting.
- Pod disruption budgets, autoscaling strategy and capacity planning.
- Centralized logs, metrics, alerts and runbooks.
- Separate dev/test/prod configuration and approvals.
