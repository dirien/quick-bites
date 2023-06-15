import * as pulumi from "@pulumi/pulumi";
import * as cluster from "@pulumi/azure-native/containerservice/v20230301";

const stackReference = new pulumi.StackReference("changeme");

import * as azapi from "@ediri/azapi";

const newKubernetesVersion = "1.26.0"
const newAgentKubernetesVersion = "1.26.0"

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
