import * as resources from "@pulumi/azure-native/resources";
import * as cluster from "@pulumi/azure-native/containerservice/v20230301";


const resourceGroup = new resources.ResourceGroup("my-resource-group");

const kubernetesVersion = "1.24.9"
const agentKubernetesVersion = "1.24.9"

const managedCluster = new cluster.ManagedCluster("my-cluster", {
    kubernetesVersion: kubernetesVersion,
    location: resourceGroup.location,
    resourceGroupName: resourceGroup.name,
    resourceName: "my-cluster",
    nodeResourceGroup: "my-cluster-nodes",
    dnsPrefix: resourceGroup.name,
    identity: {
        type: "SystemAssigned",
    },
    networkProfile: {
        networkPlugin: "azure",
        networkPolicy: "calico",
    },
    oidcIssuerProfile: {
        enabled: true,
    },
    agentPoolProfiles: [{
        name: "agentpool",
        count: 3,
        vmSize: "Standard_B2ms",
        osType: "Linux",
        osDiskSizeGB: 30,
        type: "VirtualMachineScaleSets",
        mode: "System",
        orchestratorVersion: agentKubernetesVersion,
    },
        {
            name: "workload1",
            count: 1,
            vmSize: "Standard_B2ms",
            osType: "Linux",
            osDiskSizeGB: 30,
            type: "VirtualMachineScaleSets",
            mode: "User",
            orchestratorVersion: agentKubernetesVersion,
        }],
});

const workloadPool = new cluster.AgentPool("workload-pool", {
    resourceGroupName: resourceGroup.name,
    resourceName: managedCluster.name,
    agentPoolName: "workload2",
    count: 1,
    vmSize: "Standard_B2ms",
    osType: "Linux",
    osDiskSizeGB: 30,
    type: "VirtualMachineScaleSets",
    mode: "User",
    orchestratorVersion: agentKubernetesVersion,
}, {
    dependsOn: [managedCluster],
});

export const resourceGroupName = resourceGroup.name;
export const managedClusterName = managedCluster.name;

export const managedClusterId = managedCluster.id;
export const resourceGroupId = resourceGroup.id;
