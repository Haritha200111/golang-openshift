# OpenShift learning and implementation tracker

Status legend: [x] completed, [~] in progress, [ ] not started. Only mark a task complete after observing it in the cluster.

## Environment discovery (previously completed)
- [x] Confirm active OpenShift project and CLI context.
- [x] Inspect existing Pods, Deployments, Services, Routes, BuildConfigs, Builds and ImageStreams.
- [x] Inspect StorageClasses; identify default gp3 EBS CSI class and `WaitForFirstConsumer`.
- [x] Inspect compute quotas, storage quota and LimitRange defaults.
- [x] Check project permissions for Deployments, StatefulSets, PVCs and Routes.
- [x] Identify that listing cluster-scoped PVs is forbidden by current RBAC.
- [x] Confirm there are no existing PVCs in the project at inspection time.
- [x] Confirm existing sample Go app used Kubernetes Deployment rather than DeploymentConfig.

## Local application preparation
- [x] Add pgx dependency to `go.mod` (target: v5.7.2 / Go 1.23).
- [ ] Run `go mod tidy` with local toolchain and inspect final `go.mod` / `go.sum`.
- [ ] Run `go test ./...` and `go build ./...`.
- [ ] Verify Dockerfile builds a Linux/AMD64 image.
- [ ] Push image to a registry accessible to OpenShift.

## OpenShift implementation
- [ ] Create database Secret without committing credentials.
- [ ] Create ConfigMap.
- [ ] Create PostgreSQL PVC and understand StorageClass provisioning.
- [ ] Deploy PostgreSQL StatefulSet and Service.
- [ ] Verify PostgreSQL readiness and persistent data.
- [ ] Deploy Go API ServiceAccount, Deployment and Service.
- [ ] Inspect pod SCC and effective security context.
- [ ] Verify readiness, liveness and startup probes.
- [ ] Create Route with edge TLS termination.
- [ ] Test all API endpoints through the Route.
- [ ] Apply NetworkPolicies and verify permitted/blocked traffic.
- [ ] Test database pod recreation without deleting its PVC.

## OpenShift concepts to study
- [ ] Projects, namespaces, quotas, LimitRanges and RBAC.
- [ ] BuildConfig, Build, S2I, ImageStreams and triggers.
- [ ] Route vs Ingress; router and TLS termination types.
- [ ] SCC, restricted-v2, arbitrary UID and non-root containers.
- [ ] Operators, OLM, CRDs, CSVs, Subscriptions and InstallPlans.
- [ ] CSI, StorageClasses, PVC/PV lifecycle and reclaim policies.
- [ ] OpenShift monitoring, metrics, alerts, events and logs.
- [ ] GitOps reconciliation using OpenShift GitOps / Argo CD.

## EOD status update source
Only report tasks marked [x] as completed. Keep deployment and cluster validation items as pending until observed.
