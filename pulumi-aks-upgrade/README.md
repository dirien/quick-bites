# How to upgrade your AKS cluster using only Pulumi

## TL;DR: The code

## Introduction

Recently, I got a request from one of our customers asking how to upgrade their AKS cluster using only [Pulumi](https://www.pulumi.com) and not the Azure CLI. The customer also mentioned that they did not find a solution to this problem. So I took up this challenge, hoping that I can find a solution to this problem.

## Prerequisites

Before we start, you need to have the following prerequisites in place:

* [Pulumi](https://www.pulumi.com/docs/get-started/install/)

* Optional: Pulumi Account (for state storage), see [Pulumi Cloud Account](https://www.pulumi.com/docs/pulumi-cloud/accounts/)

* [Node.js](https://nodejs.org/en/download/)

* [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)

* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)


In the folder `infrastructure` you will find the Pulumi code to create the AKS cluster. This will be the starting point for this blog post.

### Create the AKS cluster

> **Note:** You need to be logged in to Azure. You can see all the different ways to log into Azure in the Pulumi provider documentation for [Azure Native](https://www.pulumi.com/registry/packages/azure-native-v2/installation-configuration/).

To create the AKS cluster, you need to run the following commands:

```bash
cd infrastructure
pulumi up
```

The code is written in TypeScript, because why not! It may take a while to create the AKS cluster. So go and grab yourself a coffee.

## Upgrade your AKS cluster, the Azure CLI way

Before I start with the Pulumi solution, I want to show you how to upgrade your AKS cluster using the Azure CLI. No, I am not showing you how to upgrade your AKS cluster using the Azure Portal. I am sure you will figure that out yourself, I am not going to support ClickOps here!

Before we start our upgrade, let's check the available Kubernetes versions for our AKS cluster:

```bash
az aks get-upgrades --name my-cluster --resource-group my-resource-group0012049e --output table
```

The output should look like this:

```text
Name     ResourceGroup              MasterVersion    Upgrades
-------  -------------------------  ---------------  -----------------------
default  my-resource-group0012049e  1.24.9           1.24.10, 1.25.5, 1.25.6
```

Nice, we pick the latest version, which is `1.25.6` and upgrade our AKS cluster:

```bash
az aks upgrade --name my-cluster --resource-group my-resource-group0012049e --kubernetes-version 1.25.6
```

With the flag `--control-plane-only` you can upgrade only the control plane and not the nodes.

> **Note:** Keep in mind, that you can only upgrade one minor version at a time. So if you are on version `1.24.9` you can't upgrade to `1.26.x` directly. You need to upgrade to `1.25.x` first and then to `1.26.x`.

You should see the following output:

```text
ubernetes may be unavailable during cluster upgrades.
 Are you sure you want to perform this operation? (y/N): y
Since control-plane-only argument is not specified, this will upgrade the control plane AND all nodepools to version 1.25.6. Continue? (y/N): y
{
...
}
```

And we check also via the `kubectl` command, if the upgrade was successful:

```bash
kubectl get nodes

aks-agentpool-32360557-vmss000000   Ready    agent   12m     v1.25.6
aks-agentpool-32360557-vmss000001   Ready    agent   11m     v1.25.6
aks-agentpool-32360557-vmss000002   Ready    agent   9m53s   v1.25.6
aks-workload1-32360557-vmss000000   Ready    agent   15m     v1.25.6
aks-workload1-32360557-vmss000001   Ready    agent   14m     v1.25.6
aks-workload1-32360557-vmss000002   Ready    agent   12m     v1.25.6
```

And the control plane:

```bash
kubectl version --short

...
Server Version: v1.25.6
```

Looks good, right? Before we move on to the Pulumi solution, let me give you some background information on the choreography Azure is doing in the background to ensure minimal disruption to your running workloads.

## How does the upgrade work?

As mentioned before, Azure is doing certain steps in the background to ensure minimal disruption to your running.

* Adds a new, so-called buffer node to the cluster with the specified Kubernetes version.

* Cordons and drains one of the old nodes.

* When the node is drained, it's re-imaged with the new Kubernetes version and becomes a new buffer node.

* Rinse and repeat until all nodes are upgraded.

* In the end, the buffer node is removed to make sure you have the same number of nodes as before.


## Upgrade your AKS cluster, the Pulumi way!

With the knowledge we have now, we can start to write our Pulumi code to upgrade our AKS cluster. First of all, we reset our AKS cluster to the original state

> **Note:** You can't downgrade your AKS cluster. So if you want to downgrade your AKS cluster, you need to delete it and create a new one.

```bash
pulumi destroy
pulumi up
```

### The Problem

If you look at the Pulumi code, you will see that there are two variables defined for the Kubernetes version.

```typescript
const kubernetesVersion = "1.24.9"
const agentKubernetesVersion = "1.24.9"
```

So naturally, you would think that you can just change the value to the new version and run `pulumi up` again. But if you do that, you will see the following error:

```bash
 error: Code="NotAllAgentPoolOrchestratorVersionSpecifiedAndUnchanged" Message="Using managed cluster api, all Agent pools' OrchestratorVersion must be all specified or all unspecified. If all specified, they must be stay unchanged or the same with control plane. For agent pool specific change, please use per agent pool operations: https://aka.ms/agent-pool-rest-api"
```

To safely upgrade your AKS cluster, we create a second Pulumi program, which we can execute when we want to upgrade our AKS cluster.

To do that, we create a new folder `upgrade-aks` and initialize a new Pulumi project:

```bash
mkdir upgrade-aks
cd upgrade-aks
pulumi new azure-typescript -n upgrade-aks -d "upgrade aks cluster" -s dev
```

And the magic sauce is the usage of the `pulumi-azapi` provider. In particular, the `UpdateResource` resource. This resource is used to add or modify properties on an existing resource. The good thing is, we can delete this Pulumi project afterward, because when `UpdateResource` is deleted, no operation will be performed in Azure.

To get information about the AKS cluster, we will also use the Pulumi concept of `StackReference`. This allows us to access the information from the other Pulumi project.

If you peek into the `index.ts` file, in the `infrastructure` folder, you will see that we are exporting the following values

```typescript
export const resourceGroupName = resourceGroup.name;
export const managedClusterName = managedCluster.name;

export const managedClusterId = managedCluster.id;
export const resourceGroupId = resourceGroup.id;
```

The code for the `upgrade-aks` project looks like this:

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as cluster from "@pulumi/azure-native/containerservice/v20230301";

const stackReference = new pulumi.StackReference("<org/projectname/stack>");

import * as azapi from "@ediri/azapi";

const newKubernetesVersion = "1.24.10"
const newAgentKubernetesVersion = "1.24.10"

const clusterUpdate = new azapi.UpdateResource("cluster", {
    type: "Microsoft.ContainerService/managedClusters@2023-03-01",
    parentId: stackReference.requireOutput("resourceGroupId"),
    name: stackReference.requireOutput("managedClusterName"),
    body: pulumi.jsonStringify({
        properties: {
            id: stackReference.requireOutput("managedClusterId"),
            kubernetesVersion: newKubernetesVersion
        }
    })
})

const aks = cluster.getManagedClusterOutput({
    resourceGroupName: stackReference.requireOutput("resourceGroupName"),
    resourceName: stackReference.requireOutput("managedClusterName"),
})

aks.agentPoolProfiles?.apply(agentPoolProfiles => {
    for (const agentPoolProfile of agentPoolProfiles ?? []) {
        const agentPoolUpdate = new azapi.UpdateResource(`agentpool-${agentPoolProfile.name}`, {
            type: "Microsoft.ContainerService/managedClusters/agentPools@2023-03-01",
            parentId: stackReference.requireOutput("managedClusterId"),
            name: agentPoolProfile.name,
            body: pulumi.jsonStringify({
                properties: {
                    id: stackReference.requireOutput("managedClusterId"),
                    orchestratorVersion: newAgentKubernetesVersion
                }
            })
        }, {
            dependsOn: clusterUpdate
        })
    }
})
```

And we add the `pulumi-azapi` provider to the `package.json`

```json
{
  "name": "upgrade-aks",
  "main": "index.ts",
  "devDependencies": {
    "@types/node": "^16"
  },
  "dependencies": {
    "@pulumi/pulumi": "^3.0.0",
    "@pulumi/azure-native": "2.0.0-beta.1",
    "@ediri/azapi": "1.2.9"
  }
}
```

Now we can change the values of the variables in the `index.ts` file and run `pulumi up`.

```bash
const newKubernetesVersion = "1.24.10"
const newAgentKubernetesVersion = "1.24.10"
```

> **Note:** Keep in mind to have not a resource quota limit in your subscription. Otherwise, the update will fail, with an error message like this: `Operation could not be completed as it results in exceeding approved standardBSFamily Cores quota`. In this case, you need to request a quota increase.

This will first update the AKS cluster and then update the agent pools. This can take a while, so be patient.

```bash
Updating (dev)

View in Browser (Ctrl+O): https://app.pulumi.com/dirien/agent-pool/dev/updates/14

     Type                           Name                 Status              
 +   pulumi:pulumi:Stack            agent-pool-dev       created (0.85s)     
 +   ├─ azapi:index:UpdateResource  cluster              created (134s)      
 +   ├─ azapi:index:UpdateResource  agentpool-workload2  created (197s)      
 +   ├─ azapi:index:UpdateResource  agentpool-workload1  created (177s)      
 +   └─ azapi:index:UpdateResource  agentpool-agentpool  created (474s)      


Resources:
    + 5 created

Duration: 10m16s
```

### How to speed up the update process with the help of `surge`?

If you have a lot of agent pools, the update process can take a while. To speed up the process, you can tweak the max `surge` settings of your AKS cluster. Per default, the max `surge` setting is set to 1. This means that one extra node will be created and only one old node cordoned and drained.

So if you set the surge to `100%` will cause all old nodes to be cordoned off and drained at the same time. While this speeds up your update process, it can also cause some disruption to your workload. So be careful with this setting!

A rule of thumb is to set the surge for production clusters to `33%`.

## Wrap up

In this blog post, I showed you an easy way to upgrade your AKS cluster with the help of Pulumi and the help of an extra update Pulumi project. This allows you to upgrade your AKS cluster in a controlled way without using the Azure Portal or the Azure CLI.
