# OpenShift Go + PostgreSQL — Complete Deployment Commands

This document covers the complete deployment workflow, from logging into OpenShift to accessing the Go REST API, inserting records, and verifying that the records are stored in PostgreSQL.

## 1. Project structure

Expected project structure:

```text
golang-openshift/
├── main.go
├── go.mod
├── go.sum
├── Dockerfile
├── .dockerignore
└── openshift/
    ├── config/
    │   └── configmap.yaml
    ├── database/
    │   ├── secret.yaml.example
    │   ├── pvc.yaml
    │   ├── service.yaml
    │   └── statefulset.yaml
    ├── application/
    │   ├── deployment.yaml
    │   ├── service.yaml
    │   └── route.yaml
    ├── security/
    │   └── service-account.yaml
    └── networking/
        └── network-policy.yaml
```

## 2. Log in to OpenShift

Open the OpenShift Web Console.

1. Click your username.
2. Select **Copy login command**.
3. Log in using the displayed command in the Web Terminal or your local terminal.

Example:

```bash
oc login https://YOUR-OPENSHIFT-API-SERVER:6443 \
  --token=YOUR_TOKEN
```

Verify the connection:

```bash
oc whoami
oc whoami --show-server
oc project
oc get projects
```

Select your project:

```bash
oc project haritharavichandran-dev
```

Check the current project:

```bash
oc project
```

## 3. Check cluster permissions and resources

Check permissions:

```bash
oc auth can-i create deployments
oc auth can-i create statefulsets
oc auth can-i create services
oc auth can-i create routes
oc auth can-i create persistentvolumeclaims
oc auth can-i create secrets
```

Check available storage classes:

```bash
oc get storageclass
```

Check the default storage class:

```bash
oc get storageclass gp3 -o yaml
```

Check resource quotas:

```bash
oc get resourcequota
oc describe resourcequota
```

Check limit ranges:

```bash
oc get limitrange
oc describe limitrange
```

Check existing project resources:

```bash
oc get all
oc get pvc
oc get secrets
oc get configmaps
```

## 4. Create the PostgreSQL credentials

Do not store real credentials in Git.

Create the database Secret:

```bash
oc create secret generic tasks-db \
  --from-literal=username=appuser \
  --from-literal=password='ChangeThisToAStrongPassword'
```

Verify:

```bash
oc get secret tasks-db
```

Inspect Secret keys without printing their values:

```bash
oc describe secret tasks-db
```

The PostgreSQL StatefulSet and Go Deployment should both reference this Secret.

## 5. Deploy PostgreSQL persistent storage

Apply the PVC:

```bash
oc apply -f openshift/database/pvc.yaml
```

Check the PVC:

```bash
oc get pvc
oc describe pvc postgres-data
```

Wait for the claim to bind:

```bash
oc wait --for=jsonpath='{.status.phase}'=Bound \
  pvc/postgres-data --timeout=180s
```

Note: The storage class in the provided manifest is `gp3`. Confirm that it exists in your cluster. With `WaitForFirstConsumer`, the PVC may remain Pending until a pod using it is scheduled.

## 6. Deploy the PostgreSQL Service and StatefulSet

Create the PostgreSQL Service:

```bash
oc apply -f openshift/database/service.yaml
```

Deploy the PostgreSQL StatefulSet:

```bash
oc apply -f openshift/database/statefulset.yaml
```

Check resources:

```bash
oc get statefulset
oc get pods
oc get services
oc get pvc
```

Wait for PostgreSQL:

```bash
oc rollout status statefulset/postgres --timeout=300s
```

Check PostgreSQL pod logs:

```bash
oc logs postgres-0
```

Check the pod details:

```bash
oc describe pod postgres-0
```

Verify the PostgreSQL Service:

```bash
oc get service postgres
oc get endpoints postgres
```

Expected internal service address:

```text
postgres:5432
```

The Go application will use this service name to connect to PostgreSQL.

## 7. Deploy the ConfigMap

Apply the application configuration:

```bash
oc apply -f openshift/config/configmap.yaml
```

Verify:

```bash
oc get configmap
oc describe configmap
```

The ConfigMap should provide:

```text
PORT=8080
DB_HOST=postgres
DB_PORT=5432
DB_NAME=tasks
```

The username and password must come from the Secret, not the ConfigMap.

## 8. Create the ServiceAccount

Apply:

```bash
oc apply -f openshift/security/service-account.yaml
```

Verify:

```bash
oc get serviceaccount task-api
oc describe serviceaccount task-api
```

The application does not need Kubernetes API access for its normal HTTP and PostgreSQL operations.

## 9. Build the Go application image

Before deploying, build the application image and push it to a registry accessible by OpenShift.

Run these commands on your local machine where Docker is available.

From the project root:

```bash
go mod tidy
go test ./...
go build ./...
```

Build the image:

```bash
docker build -t YOUR_DOCKERHUB_USERNAME/golang-openshift:1.0.0 .
```

Log in to Docker Hub:

```bash
docker login
```

Push the image:

```bash
docker push YOUR_DOCKERHUB_USERNAME/golang-openshift:1.0.0
```

Verify that the image exists in your registry.

If Docker is unavailable locally, use an available CI build or OpenShift build process instead. The Deployment still needs to reference an image that the cluster can pull.

## 10. Update the Deployment image

Open:

```text
openshift/application/deployment.yaml
```

Replace:

```yaml
image: quay.io/YOUR_ORG/golang-openshift:1.0.0
```

With your actual image:

```yaml
image: docker.io/YOUR_DOCKERHUB_USERNAME/golang-openshift:1.0.0
```

Ensure the manifest contains:

- Two application replicas.
- The ConfigMap environment variables.
- The database Secret references.
- The `task-api` ServiceAccount.
- Non-root security settings.
- Startup, readiness, and liveness probes.

## 11. Deploy the Go application

Apply the Deployment:

```bash
oc apply -f openshift/application/deployment.yaml
```

Apply the Service:

```bash
oc apply -f openshift/application/service.yaml
```

Check the Deployment:

```bash
oc get deployment task-api
oc describe deployment task-api
```

Check pods:

```bash
oc get pods -l app=task-api
```

Wait for rollout:

```bash
oc rollout status deployment/task-api --timeout=300s
```

Check application logs:

```bash
oc logs deployment/task-api
```

If there are multiple replicas:

```bash
oc logs -l app=task-api --all-containers=true
```

Check the Service:

```bash
oc get service task-api
oc get endpoints task-api
```

## 12. Deploy the NetworkPolicy

Apply:

```bash
oc apply -f openshift/networking/network-policy.yaml
```

Verify:

```bash
oc get networkpolicy
oc describe networkpolicy
```

The intended communication flow is:

```text
External Client
      |
      v
OpenShift Router
      |
      v
Go Application Service
      |
      v
Go Application Pods
      |
      v
PostgreSQL Service
      |
      v
PostgreSQL Pod
      |
      v
Persistent Volume
```

Check that the policy selectors and router namespace labels match your cluster. If the application cannot be reached after applying the policies, inspect the policy rules and cluster networking implementation.

## 13. Expose the application using an OpenShift Route

Apply the Route:

```bash
oc apply -f openshift/application/route.yaml
```

Get the Route:

```bash
oc get routes
```

Get only the hostname:

```bash
oc get route task-api \
  -o jsonpath='{.spec.host}'
```

Save the hostname:

```bash
APP_HOST=$(oc get route task-api -o jsonpath='{.spec.host}')
echo "$APP_HOST"
```

The application URL:

```text
https://YOUR-ROUTE-HOSTNAME
```

Check the Route configuration:

```bash
oc describe route task-api
```

## 14. Test the application endpoints

Set the application URL:

```bash
APP_URL="https://YOUR-ROUTE-HOSTNAME"
```

### 14.1 Home endpoint

```bash
curl -i "$APP_URL/"
```

### 14.2 Liveness endpoint

```bash
curl -i "$APP_URL/healthz"
```

Expected: HTTP 200 if the application is running.

### 14.3 Readiness endpoint

```bash
curl -i "$APP_URL/readyz"
```

Expected: HTTP 200 if the database connection is healthy.

### 14.4 Retrieve all tasks

```bash
curl -i "$APP_URL/api/tasks"
```

Initially, this should return an empty list if no tasks have been inserted.

## 15. Insert a record into PostgreSQL through the API

Send a POST request:

```bash
curl -i -X POST "$APP_URL/api/tasks" \
  -H "Content-Type: application/json" \
  -d '{"title":"Learn OpenShift PostgreSQL deployment"}'
```

This request goes through:

1. OpenShift Route.
2. Go application Service.
3. Go application Pod.
4. Go REST API.
5. PostgreSQL Service.
6. PostgreSQL database.

The Go application should insert the record into the `tasks` table.

The response should contain the newly created task and its generated ID.

## 16. Retrieve the inserted record

Run:

```bash
curl -i "$APP_URL/api/tasks"
```

You should now see the task created in the previous step.

For example:

```json
[
  {
    "id": 1,
    "title": "Learn OpenShift PostgreSQL deployment",
    "completed": false
  }
]
```

The actual ID and timestamp depend on the database.

## 17. Verify the record directly inside PostgreSQL

Open a shell inside the PostgreSQL pod:

```bash
oc rsh postgres-0
```

Connect to the database:

```bash
psql -U appuser -d tasks
```

If the database image does not expose `psql` in the default PATH, locate the client binary or use an approved PostgreSQL client image.

Run:

```sql
\dt
```

Check the table:

```sql
SELECT * FROM tasks;
```

Check the record count:

```sql
SELECT COUNT(*) FROM tasks;
```

Exit PostgreSQL:

```sql
\q
```

Exit the pod:

```bash
exit
```

If the task appears in the SQL query, the data is stored in PostgreSQL rather than only in the Go application's memory.

## 18. Update a task

First retrieve the task ID:

```bash
curl "$APP_URL/api/tasks"
```

Use the returned ID:

```bash
curl -i -X PATCH "$APP_URL/api/tasks/1" \
  -H "Content-Type: application/json" \
  -d '{"completed":true}'
```

Verify through the API:

```bash
curl "$APP_URL/api/tasks"
```

Verify directly in PostgreSQL:

```bash
oc rsh postgres-0
psql -U appuser -d tasks
```

```sql
SELECT id, title, completed FROM tasks;
```

## 19. Verify persistence after application restart

List the application pods:

```bash
oc get pods -l app=task-api
```

Delete one application pod:

```bash
oc delete pod POD_NAME
```

Wait for a replacement:

```bash
oc get pods -l app=task-api -w
```

Test the API again:

```bash
curl "$APP_URL/api/tasks"
```

The previously inserted task should still exist because PostgreSQL stores the data, not the application pod.

## 20. Verify persistence after PostgreSQL pod restart

This is a separate test. Do this only after confirming that the PVC is Bound and the database has successfully initialized.

Delete the PostgreSQL pod:

```bash
oc delete pod postgres-0
```

Wait for the StatefulSet to recreate it:

```bash
oc rollout status statefulset/postgres --timeout=300s
```

Check:

```bash
oc get pods
oc get pvc
```

Query the API again:

```bash
curl "$APP_URL/api/tasks"
```

The existing records should remain available because the StatefulSet mounts the persistent volume claim.

Do not delete the PVC during this test.

## 21. Useful monitoring commands

Check all resources:

```bash
oc get all
```

Check pods:

```bash
oc get pods -o wide
```

Check Deployments:

```bash
oc get deployments
```

Check StatefulSets:

```bash
oc get statefulsets
```

Check Services:

```bash
oc get services
```

Check Routes:

```bash
oc get routes
```

Check PVCs:

```bash
oc get pvc
```

Check Events:

```bash
oc get events --sort-by=.lastTimestamp
```

Check Go application logs:

```bash
oc logs deployment/task-api --tail=100
```

Check PostgreSQL logs:

```bash
oc logs postgres-0 --tail=100
```

Follow application logs:

```bash
oc logs -f deployment/task-api
```

## 22. Troubleshooting

### Pod is ImagePullBackOff

```bash
oc describe pod POD_NAME
```

Check:

- Image repository and tag.
- Registry permissions.
- Image pull Secret configuration.
- Whether the image exists.

### Pod is CrashLoopBackOff

```bash
oc logs POD_NAME --previous
oc describe pod POD_NAME
```

Check application startup errors and database connection settings.

### PostgreSQL pod is Pending

```bash
oc describe pod postgres-0
oc get pvc
oc describe pvc postgres-data
oc get events --sort-by=.lastTimestamp
```

Check storage class, quota, access mode, and available capacity.

### Application cannot connect to PostgreSQL

```bash
oc get service postgres
oc get endpoints postgres
oc logs deployment/task-api
oc logs postgres-0
```

Verify:

```text
DB_HOST=postgres
DB_PORT=5432
DB_NAME=tasks
```

Also confirm the username and password Secret keys match the application manifest.

### Route is not accessible

```bash
oc get route task-api
oc describe route task-api
oc get service task-api
oc get endpoints task-api
```

Check that the Route points to the correct Service and port.

### Readiness probe is failing

```bash
oc describe pod POD_NAME
oc logs POD_NAME
```

Check whether PostgreSQL is available and `/readyz` returns HTTP 200.

## 23. Final verification checklist

- [ ] OpenShift login successful.
- [ ] Correct project selected.
- [ ] Permissions and quotas checked.
- [ ] Database Secret created.
- [ ] PostgreSQL PVC created and Bound.
- [ ] PostgreSQL StatefulSet running.
- [ ] PostgreSQL Service has endpoints.
- [ ] ConfigMap created.
- [ ] Go application image built and pushed.
- [ ] Go application Deployment running.
- [ ] Application Service has endpoints.
- [ ] NetworkPolicy applied and connectivity verified.
- [ ] Route created.
- [ ] Home endpoint accessible.
- [ ] Health endpoint returns success.
- [ ] Readiness endpoint returns success.
- [ ] POST API successfully inserts a task.
- [ ] GET API returns the inserted task.
- [ ] PostgreSQL SQL query confirms the task exists.
- [ ] Data remains after application pod restart.
- [ ] Data remains after PostgreSQL pod restart.

## 24. Important production considerations

This exercise demonstrates a production-style OpenShift deployment baseline. It is not yet a fully production-ready database architecture.

Before production use, consider:

- PostgreSQL high availability and disaster recovery.
- Automated backups and restore testing.
- Database migration Jobs instead of schema creation by every application replica.
- Secret rotation and an external secret manager.
- Image vulnerability scanning and image signing.
- Resource sizing and autoscaling.
- Pod disruption budgets.
- Monitoring, alerting, and centralized logs.
- Database connection limits and pool sizing.
- Authentication and authorization for application APIs.
- Tested NetworkPolicies and restricted SCC compatibility.

**Expected final result:** A Go REST API runs on OpenShift, connects to PostgreSQL through an internal Service, accepts API requests through an OpenShift Route, and persists task records on a PostgreSQL PersistentVolumeClaim.



Red Hat's official Developer Sandbox documentation. It explicitly states that:
You can bring them back by setting the replica count again.
Pods in the Developer Sandbox are automatically stopped/scaled to zero after 12 hours of use.
 

PS C:\Users\HarithaRavichandran> oc scale statefulset postgres --replicas=1
statefulset.apps/postgres scaled
PS C:\Users\HarithaRavichandran> oc get pods -w
NAME                                         READY   STATUS    RESTARTS   AGE
postgres-0                                   0/1     Running   0          10s
workspace8b01e2bce60641fb-5c84c98d48-gs684   3/3     Running   0          21m
postgres-0                                   1/1     Running   0          15s
PS C:\Users\HarithaRavichandran> oc scale deployment task-api --replicas=2
deployment.apps/task-api scaled
PS C:\Users\HarithaRavichandran>
