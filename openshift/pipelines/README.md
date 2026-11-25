What each new file is for:
File	Purpose
workspace-pvc.yaml	Temporary shared storage where pipeline tasks access the cloned source code
go-ci-task.yaml	Our own Task for Go code validation
deployment-role.yaml	Allows the pipeline to update the existing API Deployment
deployment-rolebinding.yaml	Gives that permission to the pipeline service account
pipeline.yaml	Defines the complete CI/CD workflow
pipelinerun.yaml	Starts an execution of the pipeline
README.md	Explains the pipeline and how to operate it

The git-clone, buildah, and openshift-client Tasks are already installed in the openshift-pipelines namespace. We will reuse them rather than creating duplicate versions. Your uploaded output confirms their names, parameters and workspace requirements.




# OpenShift CI/CD Pipeline — Tekton

## 1. Project details

- OpenShift project: `haritharavichandran-dev`
- GitHub repository: `https://github.com/Haritha2001/golang-openshift.git`
- Branch: `main`
- Pipeline: `task-api-cicd`
- Pipeline service account: `pipeline`
- Docker Hub image: `docker.io/haritharavichandran/golang-openshift`
- Pipeline workspace PVC: `pipeline-workspace`
- Docker Hub Secret: `dockerhub-auth`

## 2. Verify OpenShift Pipelines Tasks

```powershell
oc get tasks -n openshift-pipelines

oc get task git-clone -n openshift-pipelines
oc get task buildah -n openshift-pipelines
oc get task openshift-client -n openshift-pipelines
```

## 3. Create the pipeline workspace

```powershell
oc apply -f .\openshift\pipelines\workspace-pvc.yaml

oc get pvc pipeline-workspace
```

## 4. Create the Go validation Task

```powershell
oc apply -f .\openshift\pipelines\go-ci-task.yaml

oc get tasks
```

## 5. Configure pipeline deployment permissions

```powershell
oc apply -f .\openshift\pipelines\deployment-role.yaml

oc apply -f .\openshift\pipelines\deployment-rolebinding.yaml

oc get role pipeline-deployer

oc get rolebinding pipeline-deployer-binding
```

## 6. Configure Docker Hub authentication

Create a Docker Hub access token with permission to push images.

```powershell
$env:DOCKERHUB_USERNAME = Read-Host "Docker Hub username"
$env:DOCKERHUB_TOKEN = Read-Host "Docker Hub access token"

oc create secret docker-registry dockerhub-auth `
  --docker-server=https://index.docker.io/v1/ `
  --docker-username=$env:DOCKERHUB_USERNAME `
  --docker-password=$env:DOCKERHUB_TOKEN `
  --docker-email="unused@example.com"

Remove-Item Env:DOCKERHUB_USERNAME
Remove-Item Env:DOCKERHUB_TOKEN

oc get secret dockerhub-auth
```

If the Secret already exists and you need to update it, delete and recreate it:

```powershell
oc delete secret dockerhub-auth
```

Then repeat the creation commands above.

## 7. Create the Tekton Pipeline

```powershell
oc apply -f .\openshift\pipelines\pipeline.yaml

oc get pipelines
oc describe pipeline task-api-cicd
```

## 8. Start the Pipeline

```powershell
oc create -f .\openshift\pipelines\pipelinerun.yaml

oc get pipelineruns
```

## 9. Monitor the Pipeline

```powershell
oc get pipelineruns -w

oc get taskruns

oc get pods
```

To inspect a TaskRun:

```powershell
oc describe taskrun <taskrun-name>
```

To inspect the logs of a TaskRun pod:

```powershell
oc logs <pod-name> -c step-<step-name>
```

## 10. Verify the updated application

```powershell
oc rollout status deployment/task-api

oc get deployment task-api

oc get pods

oc get route
```

## 11. Run the pipeline after future code changes

Commit and push your application changes:

```powershell
git add .
git commit -m "Update Go application"
git push origin main
```

Start a new pipeline execution:

```powershell
oc create -f .\openshift\pipelines\pipelinerun.yaml
```

Monitor it:

```powershell
oc get pipelineruns -w
```

## 12. Useful troubleshooting commands

```powershell
oc get pipelines

oc get tasks

oc get pipelineruns

oc get taskruns

oc get pods

oc get events --sort-by=.metadata.creationTimestamp

oc describe pipelinerun <pipelinerun-name>

oc describe taskrun <taskrun-name>

oc logs <pod-name> --all-containers=true
```


## 13. Remove pipeline resources

Only use these commands if you intentionally want to remove the pipeline setup.

```powershell
oc delete -f .\openshift\pipelines\pipelinerun.yaml
oc delete -f .\openshift\pipelines\pipeline.yaml
oc delete -f .\openshift\pipelines\deployment-rolebinding.yaml
oc delete -f .\openshift\pipelines\deployment-role.yaml
oc delete -f .\openshift\pipelines\go-ci-task.yaml
oc delete -f .\openshift\pipelines\workspace-pvc.yaml
oc delete secret dockerhub-auth
```

**Warning:** Deleting `pipeline-workspace` removes the pipeline's stored source workspace. Do not delete your existing application or PostgreSQL resources as part of pipeline cleanup.