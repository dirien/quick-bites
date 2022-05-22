# Quick Bites of FluxCD: Health assessment

During KubeCon EU, I talked with [Stefan Prodan](https://twitter.com/stefanprodan) about a way to delay the deployment
of workload. I told him, that I use `dependsOn` but this did not check if the deployed resources is ready.

He suggested, that I have a look into Health checks. Health checks are available in the `Kustomization` resource.

## How to implement health checks

As stated before, a Kustomization can contain health checks, actually a whole series of them. This will be used to
determine the rollout status of the deployed workloads. In addition, you can check the ready status of custom resources
too.

To enabled health checking just set `spec.wait` and `spec.timeout`. This will be valid for all the reconciled resources.

```yaml
TODO
```

If you want to check the only certain resources, you need to list them under `spec.healthChecks`.

> Remember: when `spec.wait` is set, the `spec.healthChecks` field will be ignored.

Following types can be referenced by health check entries:

* Kubernetes builtin kinds: Deployment, DaemonSet, StatefulSet, PersistentVolumeClaim, Pod, PodDisruptionBudget, Job,
  CronJob, Service, Secret, ConfigMap, CustomResourceDefinition
* GitOps Toolkit kinds: HelmRelease, HelmRepository, GitRepository, etc
* Custom resources that are compatible
  with [kstatus](https://github.com/kubernetes-sigs/cli-utils/tree/master/pkg/kstatus)

```yaml
TODO
```

After applying the `Kustomization` resource, the controller tries to verify if the rollout completed successfully.

If the deployment went successfully through, the condition on `Kustomization` resource is marked as `true`. If the
deployment failed, or timeout, then the `Kustomization` ready condition will be `false`.

In case the deployment becomes healthy on the next execution cycle, then the `Kustomization` will be marked as ready.

If the `Kustomization` contains `HelmRelease` objects you can define a health check that waits for the `HelmReleases` to
be reconciled.

```yaml
TODO
```

## Combine health checks with `dependsOn`

We know that when applying a Kustomization, that we can define workloads that must be deployed before the Kustomization
will be applied. Best example is the `cert-manager`, as we may need to create a certificate in our actual deployment.

With the field `spec.dependsOn` we can bring this deployments into an order. The Kustomization with dependencies will be
applied only after the dependencies are ready.

```yaml
TODO
```

Now combine this with health assessment, and we have a perfect way to ensusre that the current Kusomization will be
appliend when all the dependencies are healthy.

## Demo

For this little demo, we will create a kind cluster and install the Flux via the Helm chart:

```bash
kind create cluster --name flux-health
```

Takes a couple of minutes, to be ready:

```bash
Creating cluster "flux-health" ...
 ✓ Ensuring node image (kindest/node:v1.24.0) 🖼 
 ✓ Preparing nodes 📦  
 ✓ Writing configuration 📜 
 ✓ Starting control-plane 🕹️ 
 ✓ Installing CNI 🔌 
 ✓ Installing StorageClass 💾 
Set kubectl context to "kind-flux-health"
You can now use your cluster with:

kubectl cluster-info --context kind-flux-health

Not sure what to do next? 😅  Check out https://kind.sigs.k8s.io/docs/user/quick-start/
```

Now we apply the Helm chart:

```bash
helm repo add fluxcd-community https://fluxcd-community.github.io/helm-charts
helm repo update
helm upgrade -i flux2 fluxcd-community/flux2 --create-namespace --namespace flux-system
```

I created a `bootstrap` folder, to create the `GitRepository` and `Kustomization` objects.

```bash
kubectl apply -f bootstrap/
kustomization.kustomize.toolkit.fluxcd.io/quick-bites created
gitrepository.source.toolkit.fluxcd.io/quick-bites created
```

<pic>



