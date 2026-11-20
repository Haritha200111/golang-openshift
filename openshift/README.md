# OpenShift deployment

This folder contains the Kubernetes/OpenShift resources for the Go Task API.

## Before deploying
- Confirm the Go source compiles: `go test ./...` and `go build ./...`.
- Build and push the image to a registry accessible to your OpenShift cluster.
- Create the database Secret using the commands in the root README. Do not commit real credentials.
- Check that your sandbox permits PVC creation, Routes, and NetworkPolicies.

## Apply order
1. `oc apply -f openshift/config/configmap.yaml`
2. Create the `tasks-db` Secret (commands in root README)
3. `oc apply -f openshift/database/`
4. Wait for PostgreSQL to become Ready.
5. Update the image in `openshift/application/deployment.yaml` to your pushed image.
6. `oc apply -f openshift/security/`
7. `oc apply -f openshift/application/`
8. `oc apply -f openshift/networking/`

Use `oc get`, `oc describe`, `oc logs`, and `oc events` to inspect every resource. Do not delete the database PVC unless you intentionally want to destroy the database data.
