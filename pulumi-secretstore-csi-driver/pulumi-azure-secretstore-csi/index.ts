import * as pulumi from "@pulumi/pulumi";
import * as resources from "@pulumi/azure-native/resources/v20220901";
import * as containerservice from "@pulumi/azure-native/containerservice/v20220902preview";
import * as keyvault from "@pulumi/azure-native/keyvault/v20220701";
import * as azuread from "@pulumi/azuread";
import * as k8s from "@pulumi/kubernetes";
import {config} from "@pulumi/azure-native"

const resourceGroup = new resources.ResourceGroup("csi-driver-demo", {
    resourceGroupName: "csi-driver-demo",
});

const aks = new containerservice.ManagedCluster("csi-driver-demo", {
    kubernetesVersion: "1.25.2",
    location: resourceGroup.location,
    resourceGroupName: resourceGroup.name,
    resourceName: "csi-driver-demo-aks",
    nodeResourceGroup: "csi-driver-demo-aks-nodes",
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
    }],
});

const app = new azuread.Application("csi-driver-demo-app", {
    displayName: "csi-driver-demo",

})

const enterpriseSP = new azuread.ServicePrincipal("csi-driver-demo-sp", {
    applicationId: app.applicationId,
})

const vault = new keyvault.Vault("csi-driver-demo-vault", {
    location: resourceGroup.location,
    resourceGroupName: resourceGroup.name,
    vaultName: "csi-driver-demo-vault",
    properties: {
        accessPolicies: [
            {
                objectId: enterpriseSP.objectId,
                permissions: {
                    secrets: ["get"],
                    keys: ["get"],
                    certificates: ["get"],
                },
                tenantId: pulumi.interpolate`${pulumi.output(config.tenantId).apply(tenantId => tenantId)}`,
            }
        ],
        tenantId: pulumi.interpolate`${pulumi.output(config.tenantId).apply(tenantId => tenantId)}`,
        sku: {
            name: "standard",
            family: "A",
        },
    }
})

const secret = new keyvault.Secret("csi-driver-demo-secret", {
    resourceGroupName: resourceGroup.name,
    vaultName: vault.name,
    secretName: "secret",
    properties: {
        value: "secret",
    }
})


new azuread.ApplicationFederatedIdentityCredential("exampleApplicationFederatedIdentityCredential", {
    applicationObjectId: app.objectId,
    displayName: "kubernetes-federated-credential",
    description: "Kubernetes service account federated credential",
    audiences: ["api://AzureADTokenExchange"],
    issuer: pulumi.interpolate`${pulumi.output(aks.oidcIssuerProfile).apply(issuer => issuer?.issuerURL)}`,
    subject: "system:serviceaccount:default:test",
});


const creds = pulumi.all([resourceGroup.name, aks.name]).apply(([resourceGroupName, resourceName]) => {
    return containerservice.listManagedClusterUserCredentials({
        resourceGroupName,
        resourceName,
    });
});


const kubeconfig = creds.kubeconfigs[0].value.apply(enc => Buffer.from(enc, "base64").toString())

const provider = new k8s.Provider("k8s-provider", {
    kubeconfig: kubeconfig,
    enableServerSideApply: true,
}, {dependsOn: [aks, app]})


new k8s.helm.v3.Release("csi-secrets-store-provider-azure", {
    chart: "csi-secrets-store-provider-azure",
    namespace: "kube-system",
    repositoryOpts: {
        repo: "https://azure.github.io/secrets-store-csi-driver-provider-azure/charts",
    }
}, {provider: provider, dependsOn: [vault]})

new k8s.core.v1.ServiceAccount("k8s-sa", {
    metadata: {
        name: "test",
        namespace: "default",
    }
}, {provider: provider})

const secretProviderClass = new k8s.apiextensions.CustomResource("k8s-cr", {
    apiVersion: "secrets-store.csi.x-k8s.io/v1",
    kind: "SecretProviderClass",
    metadata: {
        name: "azure-secret-provider",
        namespace: "default",
    },
    spec: {
        provider: "azure",
        parameters: {
            keyvaultName: vault.name,
            clientID: app.applicationId,
            tenantId: pulumi.interpolate`${pulumi.output(config.tenantId).apply(tenantId => tenantId)}`,
            objects: pulumi.interpolate`array:
   - |
     objectName: "${secret.name}"
     objectType: "secret"`,
        }
    }
}, {provider: provider})

new k8s.apps.v1.Deployment("k8s-demo-deployment-authorized", {
    apiVersion: "apps/v1",
    kind: "Deployment",
    metadata: {
        name: "hello-server-deployment-authorized",
        labels: {
            app: "hello-server-authorized",
        },
        annotations: {
            "pulumi.com/skipAwait": "true",
        }
    },
    spec: {
        replicas: 1,
        selector: {
            matchLabels: {
                app: "hello-server-authorized",
            },
        },
        template: {
            metadata: {
                labels: {
                    app: "hello-server-authorized",
                },
            },
            spec: {
                serviceAccountName: "test",
                volumes: [{
                    name: "secrets-store-inline",
                    csi: {
                        driver: "secrets-store.csi.k8s.io",
                        readOnly: true,
                        volumeAttributes: {
                            secretProviderClass: secretProviderClass.metadata.name,
                        },
                    },
                }],
                containers: [{
                    name: "hello-server-authorized",
                    image: "ghcr.io/dirien/hello-server/hello-server:latest",
                    ports: [{
                        containerPort: 8080,
                    }],
                    env: [{
                        name: "FILE",
                        value: pulumi.interpolate`/mnt/secrets-store/${secret.name}`,
                    }],
                    volumeMounts: [{
                        name: "secrets-store-inline",
                        mountPath: "/mnt/secrets-store",
                        readOnly: true,
                    }],
                }],
            },
        },
    },
}, {provider: provider});

new k8s.apps.v1.Deployment("k8s-demo-deployment-unauthorized", {
    apiVersion: "apps/v1",
    kind: "Deployment",
    metadata: {
        name: "hello-server-deployment-unauthorized",
        labels: {
            app: "hello-server-unauthorized",
        },
        annotations: {
            "pulumi.com/skipAwait": "true",
        }
    },
    spec: {
        replicas: 1,
        selector: {
            matchLabels: {
                app: "hello-server-unauthorized",
            },
        },
        template: {
            metadata: {
                labels: {
                    app: "hello-server-unauthorized",
                },
            },
            spec: {
                serviceAccountName: "default",
                volumes: [{
                    name: "secrets-store-inline",
                    csi: {
                        driver: "secrets-store.csi.k8s.io",
                        readOnly: true,
                        volumeAttributes: {
                            secretProviderClass: secretProviderClass.metadata.name,
                        },
                    },
                }],
                containers: [{
                    name: "hello-server-unauthorized",
                    image: "ghcr.io/dirien/hello-server/hello-server:latest",
                    ports: [{
                        containerPort: 8080,
                    }],
                    env: [{
                        name: "FILE",
                        value: pulumi.interpolate`/mnt/secrets-store/${secret.name}`,
                    }],
                    volumeMounts: [{
                        name: "secrets-store-inline",
                        mountPath: "/mnt/secrets-store",
                        readOnly: true,
                    }],
                }],
            },
        },
    },
}, {provider: provider});

export const kubeConfig = pulumi.secret(kubeconfig)



