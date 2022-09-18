# Continuous Cluster Audit Scanning With The Trivy Operator Using Custom Policies

## Introduction

This article is the second part on custom policies with `Trivy`. If you missed the first article, here is the link to the article.

%[https://blog.ediri.io/how-to-write-custom-policies-for-trivy/]

In this article, we're going to write a custom policy for the `Trivy Operator` rather than for the `Trivy` itself.

Let me start with the good news: Everything you learned in the previous article is still valid. There are some small differences to reflect the fact that the `Trivy Operator` runs on Kubernetes. We get to that later.

## Trivy k8s CLI

Before we create the custom policy for the operator, let's have a deeper look on how we could use the `Trivy CLI` to check for configuration issues in our Kubernetes cluster.

The `trivy k8s` subcommand allows us to scan a Kubernetes cluster for vulnerabilities, secrets and misconfigurations. We can run the command locally or integrate it into our CI/CD pipeline. There is already a huge list of [builtin policies](https://aquasecurity.github.io/trivy-operator/v0.2.0/configuration-auditing/built-in-policies/) to use.

To use our custom policies, we need to point to our custom rego policy folder as such:

```bash
trivy k8s --config-policy ./quick-bites/trivy-custom-policy/policies --policy-namespaces user --report all -n <namespace to scan> all
```
> Take care to set the `kubeconfig` globaly or pass the k8s `context` and `kubeconfig` to the command (`--context` and `--kubeconfig` flag)

On my local `docker-desktop` cluster and got the following output:

![image.png](https://cdn.hashnode.com/res/hashnode/image/upload/v1663505438795/wz433YVZI.png align="center")

Great for on-demand scans but if we want continuous cluster audit scanning we need to use the `Trivy Operator`.

## The `Trivy Operator`

In a nutshell: the `Trivy Operator` automatically updates security report resources in response to workload and other changes on a Kubernetes cluster! This means for example, initiating a configuration audit when a new Pod is started.

![image.png](https://cdn.hashnode.com/res/hashnode/image/upload/v1663505619178/Lo97ozTUF.png align="center")

Before we can jump into deploying our custom policy. we need to create a demo Kubernetes cluster and deploy the `Trivy Operator` on it.

I am going to use `Civo` as my cloud provider and use `Pulumi` as my infrastructure as code tool.

Of course, you could use also `KIND`, `Rancher Desktop` or `Docker Desktop`.

### Prerequisites

- Have a [Civo](https://civo.com) account and the API key by hand. The `Civo` CLI is not required for this tutorial.
- Have `Pulumi` CLI installed and configured. You can find the detailed installation instructions [here](https://www.pulumi.com/docs/get-started/install/).

### Create a Kubernetes Cluster

`Pulumi` offers wide range of support for different languages to choose from to write your IaC code. I decided to use `YAML` for this tutorial.

Create a new directory, initialize a new `Pulumi` project and install the `Civo` provider with following commands:

```bash
export CIVO_TOKEN=xxxx
pulumi new yaml
```

You should have now a file called `Pulumi.yaml` in your directory. Open it and add the following lines:

```yaml
name: trivy-operator-custom-policy
runtime: yaml
description: Trivy Operator with custom policy on Civo

variables:
  civo-region: FRA1
  civo-k8s-node-size: g4s.kube.medium
  civo-k8s-node-count: 2
  trivy-namespaces: trivy-system
  trivy-operator-version: 0.2.0

resources:
  civo-firewall:
    type: civo:Firewall
    properties:
      name: MyCivoFirewall
      region: ${civo-region}

  civo-k3s-cluster:
    type: civo:KubernetesCluster
    properties:
      name: MyCivoCluster
      region: ${civo-region}
      firewallId: ${civo-firewall.id}
      cni: cilium
      pools:
        nodeCount: ${civo-k8s-node-count}
        size: ${civo-k8s-node-size}

  k8s-provider:
    type: pulumi:providers:kubernetes
    properties:
      kubeconfig: ${civo-k3s-cluster.kubeconfig}
      enableServerSideApply: true

  trivy-namespace:
    type: kubernetes:core/v1:Namespace
    properties:
      metadata:
        name: ${trivy-namespaces}
    options:
      provider: ${k8s-provider}

  trivy-operator:
    type: kubernetes:helm.sh/v3:Release
    properties:
      namespace: ${trivy-namespace.metadata.name}
      chart: trivy-operator
      version: ${trivy-operator-version}
      repositoryOpts:
        repo: https://aquasecurity.github.io/helm-charts/
      values:
        trivy:
          ignoreUnfixed: true
          imageRef: ghcr.io/aquasecurity/trivy:0.31.3
    options:
      provider: ${k8s-provider}
      
outputs:
  kubeconfig:
    Fn::Secret:
      ${civo-k3s-cluster.kubeconfig}
```

Run the preview command to see what resources will be created:

```bash
pulumi preview
```

You should see something like this:

```bash
Previewing update (dev)

View Live: https://app.pulumi.com/dirien/trivy-operator-custom-policy/dev/previews/25d8b831-a743-4090-9105-ae25be9625db

     Type                              Name                              Plan       
 +   pulumi:pulumi:Stack               trivy-operator-custom-policy-dev  create     
 +   ├─ civo:index:Firewall            civo-firewall                     create     
 +   ├─ civo:index:KubernetesCluster   civo-k3s-cluster                  create     
 +   ├─ pulumi:providers:kubernetes    k8s-provider                      create     
 +   ├─ kubernetes:core/v1:Namespace   trivy-namespace                   create     
 +   └─ kubernetes:helm.sh/v3:Release  trivy-operator                    create     
 
Outputs:
    kubeconfig: [secret]

Resources:
    + 6 to create
```

If everything looks good, we can go ahead and create the resources:

```bash
pulumi up -y -f
```

> Info: I added the flags `-y -f` to the command. The flag `-y` will automatically approve the changes and `-f` will force the update.

This may take a few moments until all resources are created and your cluster is ready to use.

To get the `kubeconfig` of the cluster, we can run the following command:

```bash
pulumi stack output kubeconfig --show-secrets > kubeconfig.yaml
```

As we are done with the cluster setup, we can go ahead transform our existing policy so the `Trivy Operator` can load it and use it for auditing and config violations.

Following steps are necessary:

- Create a `ConfigMap` named `trivy-operator-policies-config`. The `Trivy Operator` will look for this `ConfigMap` and use the policies defined in it.
- Define two data entries in the `trivy-operator-policies-config` `ConfigMap`. The `policy.<your_policy_name>.kinds` and `policy.<your_policy_name>.rego` entries.
  - The `policy.<your_policy_name>.kinds` entry should contain a list of Kubernetes resources that should be checked by the policy.
  - The `policy.<your_policy_name>.rego` entry should contain the Rego policy.
- The package name in the Rego policy should be `trivyoperator.policy.k8s.custom` package to avoid naming collision with built-in policies that are pre-installed.

The finished `ConfigMap` should look like this:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: trivy-operator-policies-config
  namespace: trivy-system
data:
  policy.ED001.kinds: "*"
  policy.ED001.rego: |
    package builtin.trivyoperator.policy.k8s.custom
        
    import future.keywords.in
    import data.lib.kubernetes
    import data.lib.result
    
    default allowedRegistries = ["quay.io","ghcr.io","gcr.io"]
    
    __rego_metadata__ := {
      "id": "ED001",
      "title": "Allowed container registry checks",
      "severity": "CRITICAL",
      "description": "The usage of non approved container registries is not permitted",
    }
    
    __rego_input__ := {
      "combine": false,
      "selector": [{"type": "kubernetes"}],
    }
    
    allowedRegistry(image) {
      registry := allowedRegistries[_]
      startswith(image, registry)
    }
    
    deny[res] {
      container := kubernetes.containers[_]
      not allowedRegistry(container.image)
      msg :=  kubernetes.format(sprintf("Container '%s' with image '%s' of %s '%s' comes from not approved container registry %s", [container.name, container.image, kubernetes.kind, kubernetes.name, allowedRegistries]))
      res := result.new(msg, container)
    }
```

As we continue to use `Pulumi`, I am going to deploy the `ConfigMap` with `Pulumi` as well. Add following code snippet after the`trivy-operator`resource definition in the `Pulumi.yaml` file:

```yaml
  trivy-policy-cm:
    type: kubernetes:core/v1:ConfigMap
    options:
      provider: ${k8s-provider}
      parent: ${trivy-operator}
    properties:
      metadata:
        name: trivy-operator-policies-config
        namespace: ${trivy-namespace.metadata.name}
      data:
        policy.ED001.kinds: "*"
        policy.ED001.rego: |
          package builtin.trivyoperator.policy.k8s.custom

          import future.keywords.in
          import data.lib.kubernetes
          import data.lib.result

          default allowedRegistries = ["quay.io","ghcr.io","gcr.io"]

          __rego_metadata__ := {
            "id": "ED001",
            "title": "Allowed container registry checks",
            "severity": "CRITICAL",
            "description": "The usage of non approved container registries is not permitted",
          }

          __rego_input__ := {
            "combine": false,
            "selector": [{"type": "kubernetes"}],
          }

          allowedRegistry(image) {
            registry := allowedRegistries[_]
            startswith(image, registry)
          }

          deny[res] {
            container := kubernetes.containers[_]
            not allowedRegistry(container.image)
            msg :=  kubernetes.format(sprintf("Container '%s' with image '%s' of %s '%s' comes from not approved container registry %s", [container.name, container.image, kubernetes.kind, kubernetes.name, allowedRegistries]))
            res := result.new(msg, container)
          }
```

Now, we can go ahead and update the stack:

```bash
pulumi up -y -f
```

And you should see in the output that the `ConfigMap` was created:

```bash
Resources:
    + 1 created
    6 unchanged
```

We can now go ahead and create a deployment that will violate our policy:

```bash
kubectl create deployment nginx --image=nginx
```
> The `Trivy Operator` will exclude the `kube-system` namespace and its own namespace (`trivy-system`) from the scan. You can configure this behaviour with setting the `excludeNamespaces` and `targetNamespaces` values.

When we retrieve the corresponding configuration audit report, we can see that our custom policy was triggered:

```bash
kubectl get configauditreport replicaset-nginx-6799fc88d8 -o wide
NAME                          SCANNER   AGE     CRITICAL   HIGH   MEDIUM   LOW
replicaset-nginx-6799fc88d8   Trivy     2m14s   1          0      3        10
```
If we describe that report we will see that it's failing because of our custom policy:

```yaml
apiVersion: aquasecurity.github.io/v1alpha1
kind: ConfigAuditReport
metadata:
  creationTimestamp: "2022-09-18T12:24:59Z"
  generation: 1
  labels:
    plugin-config-hash: 745c586b6c
    resource-spec-hash: 74d79948df
    trivy-operator.resource.kind: ReplicaSet
    trivy-operator.resource.name: nginx-6799fc88d8
    trivy-operator.resource.namespace: default
  name: replicaset-nginx-6799fc88d8
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: false
    controller: true
    kind: ReplicaSet
    name: nginx-6799fc88d8
    uid: 5d92b606-dce8-4240-a7cb-39526663a4a7
  resourceVersion: "3034"
  uid: d8d320de-1d9a-4ac8-ab34-cc2f8c2af390
report:
  checks:
...
    - category: Kubernetes Security Check
      checkID: ED001
      description: The usage of non approved container registries is not permitted
      messages:
        - Container 'nginx' with image 'nginx' of ReplicaSet 'nginx-6799fc88d8' comes
          from not approved container registry ["quay.io", "ghcr.io", "gcr.io"]
      severity: CRITICAL
      success: false
      title: Allowed container registry checks
```

Before we head over to the wrap up of this tutorial, let us clean our test cluster with the following `Pulumi` command:

```bash
pulumi destroy -y -f
```

This will delete the stack and all the resources that were created before!

## Wrap up

We have seen how we can use the `Trivy Operator` to enable continuous cluster audit scanning in our Kubernetes cluster with our own custom policies.
This is a great way to ensure that our cluster is always in a secure state and that we are not violating any policies that we have defined inside our project team or company.

It would be great if the good folks from Aqua Security would add some kind of `Admissions Controller` support to the `Trivy Operator`. This would allow us to block the deployment of any resources that violate our policies.

This would be a great addition to the `Trivy Operator` and would make it an even more perfect tool for operation teams.
