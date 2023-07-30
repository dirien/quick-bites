import * as pulumi from "@pulumi/pulumi";
import * as scaleway from "@ediri/scaleway";
import * as k8s from "@pulumi/kubernetes";

const clusterConfig = new pulumi.Config("cluster")

const kapsule = new scaleway.K8sCluster("pulumi-validating-admission-policy-cluster", {
    version: clusterConfig.require("version"),
    cni: "cilium",
    deleteAdditionalResources: true,
    featureGates: [
        "ValidatingAdmissionPolicy",
    ],
    autoUpgrade: {
        enable: clusterConfig.requireBoolean("auto_upgrade"),
        maintenanceWindowStartHour: 3,
        maintenanceWindowDay: "monday"
    },
});

const nodeConfig = new pulumi.Config("node")

const nodePool = new scaleway.K8sPool("pulumi-validating-admission-policy-node-pool", {
    nodeType: nodeConfig.require("node_type"),
    size: nodeConfig.requireNumber("node_count"),
    autoscaling: nodeConfig.requireBoolean("auto_scale"),
    autohealing: nodeConfig.requireBoolean("auto_heal"),
    clusterId: kapsule.id,
});

export const kapsuleName = kapsule.name;
export const kubeconfig = pulumi.secret(kapsule.kubeconfigs[0].configFile);

const provider = new k8s.Provider("k8s-provider", {
    kubeconfig: kubeconfig,
}, {dependsOn: [kapsule, nodePool]});

const teamLabel = new k8s.admissionregistration.v1alpha1.ValidatingAdmissionPolicy("pulumi-validating-admission-policy-team-label", {
    metadata: {
        name: "team-label",
    },
    spec: {
        failurePolicy: "Fail",
        matchConstraints: {
            resourceRules: [
                {
                    apiGroups: ["apps"],
                    apiVersions: ["v1"],
                    operations: ["CREATE", "UPDATE"],
                    resources: ["deployments"],
                },
                {
                    apiGroups: ["apps"],
                    apiVersions: ["v1"],
                    operations: ["CREATE", "UPDATE"],
                    resources: ["statefulsets"],
                }
            ]
        },
        matchConditions: [
            {
                name: "team-label",
                expression: `has(object.metadata.namespace) && !(object.metadata.namespace.startsWith("kube-"))`
            }
        ],
        validations: [
            {
                expression: `has(object.metadata.labels.team)`,
                message: "Team label is missing.",
                reason: "Invalid",
            },
            {
                expression: `has(object.spec.template.metadata.labels.team)`,
                message: "Team label is missing from pod template.",
                reason: "Invalid",
            }
        ]
    }
}, {provider: provider});

const teamLabelBinding = new k8s.admissionregistration.v1alpha1.ValidatingAdmissionPolicyBinding("pulumi-validating-admission-policy-binding-team-label", {
    metadata: {
        name: "team-label-binding",
    },
    spec: {
        policyName: teamLabel.metadata.name,
        validationActions: ["Deny"],
        matchResources: {}
    }
}, {provider: provider});


const prodReadyPolicy = new k8s.admissionregistration.v1alpha1.ValidatingAdmissionPolicy("pulumi-validating-admission-policy-prod-ready", {
    metadata: {
        name: "prod-ready-policy",
    },
    spec: {
        failurePolicy: "Fail",
        matchConstraints: {
            resourceRules: [
                {
                    apiGroups: ["apps"],
                    apiVersions: ["v1"],
                    operations: ["CREATE", "UPDATE"],
                    resources: ["deployments"],
                }]
        },
        validations: [
            {
                expression: `object.spec.template.spec.containers.all(
    c, has(c.resources) && has(c.resources.limits) && has(c.resources.limits.cpu)
)`,
                message: "No CPU resource limits specified for any container.",
                reason: "Invalid",
            },
            {
                expression: `object.spec.template.spec.containers.all(
    c, has(c.resources) && has(c.resources.limits) && has(c.resources.limits.memory)
)`,
                message: "No memory resource limits specified for any container.",
                reason: "Invalid",
            },
            {
                expression: `object.spec.template.spec.containers.all(
    c, !c.image.endsWith(':latest')
)`,
                message: "Image tag must not be latest.",
                reason: "Invalid",
            },
            {
                expression: `object.spec.template.spec.containers.all(
    c, c.image.startsWith('myregistry.azurecr.io')
)`,
                message: "Image must be pulled from myregistry.azurecr.io.",
            },
            {
                expression: `object.spec.template.spec.containers.all(
    c, has(c.securityContext)
)`,
                message: "Security context is missing.",
            },
            {
                expression: `object.spec.template.spec.containers.all(
c, has(c.readinessProbe) && has(c.livenessProbe)
)`,
                message: "No health checks configured.",
            },
            {
                expression: `object.spec.template.spec.containers.all(
    c, !('securityContext' in c) || !('privileged' in c.securityContext) || c.securityContext.privileged == false
)`,
                message: "Privileged containers are not allowed.",
            },
            {
                expression: `object.spec.template.spec.initContainers.all(
    c, !('securityContext' in c) || !('privileged' in c.securityContext) || c.securityContext.privileged == false
)`,
                message: "Privileged init containers are not allowed.",
            }]
    }
}, {provider: provider});

const prodReadyPolicyBinding = new k8s.admissionregistration.v1alpha1.ValidatingAdmissionPolicyBinding("pulumi-validating-admission-policy-prod-ready-binding", {
    metadata: {
        name: "prod-ready-policy-binding",
    },
    spec: {
        policyName: prodReadyPolicy.metadata.name,
        validationActions: ["Deny"],
        matchResources: {
            objectSelector: {
                matchLabels: {
                    "environment": "production",
                }
            }
        }
    }
}, {provider: provider});

const prodReadyCRD = new k8s.apiextensions.v1.CustomResourceDefinition("pulumi-validating-admission-policy-prod-ready-crd", {
    metadata: {
        name: "prodreadychecks.pulumi.com",
    },
    spec: {
        group: "pulumi.com",
        versions: [
            {
                name: "v1",
                served: true,
                storage: true,
                schema: {
                    openAPIV3Schema: {
                        type: "object",
                        properties: {
                            spec: {
                                type: "object",
                                properties: {
                                    registry: {
                                        type: "string",
                                    },
                                    version: {
                                        type: "string",
                                    },
                                    privileged: {
                                        type: "boolean",
                                    }
                                }
                            }
                        }
                    }
                }
            }
        ],
        scope: "Namespaced",
        names: {
            plural: "prodreadychecks",
            singular: "prodreadycheck",
            kind: "ProdReadyCheck",
            shortNames: ["prc"],
        },
    },
}, {provider: provider});

const prodReadyCR = new k8s.apiextensions.CustomResource("pulumi-validating-admission-policy-prod-ready-crd-validation", {
    apiVersion: "pulumi.com/v1",
    kind: "ProdReadyCheck",
    metadata: {
        name: "prodreadycheck-validation",
    },
    spec: {
        registry: "myregistry.azurecr.io/",
        version: "latest",
        privileged: false,
    }

}, {provider: provider, dependsOn: [prodReadyCRD]});

const prodReadyPolicyParam = new k8s.admissionregistration.v1alpha1.ValidatingAdmissionPolicy("pulumi-validating-admission-policy-prod-ready-param", {
    metadata: {
        name: "prod-ready-policy-param",
    },
    spec: {
        failurePolicy: "Fail",
        paramKind: {
            apiVersion: prodReadyCR.apiVersion,
            kind: prodReadyCR.kind,
        },
        matchConstraints: {
            resourceRules: [
                {
                    apiGroups: ["apps"],
                    apiVersions: ["v1"],
                    operations: ["CREATE", "UPDATE"],
                    resources: ["deployments"],
                }]
        },
        validations: [
            {
                expression: `object.spec.template.spec.containers.all(
    c, has(c.resources) && has(c.resources.limits) && has(c.resources.limits.cpu)
)`,
                message: "No CPU resource limits specified for any container.",
                reason: "Invalid",
            },
            {
                expression: `object.spec.template.spec.containers.all(
    c, has(c.resources) && has(c.resources.limits) && has(c.resources.limits.memory)
)`,
                message: "No memory resource limits specified for any container.",
                reason: "Invalid",
            },
            {
                expression: `object.spec.template.spec.containers.all(
    c, !c.image.endsWith(params.spec.version)
)`,
                messageExpression: "'Image tag must not be ' + params.spec.version",
                reason: "Invalid",
            },
            {
                expression: `object.spec.template.spec.containers.all(
    c, c.image.startsWith(params.spec.registry)
)`,
                reason: "Invalid",
                messageExpression: "'Registry only allowed from: ' + params.spec.registry",
            },
            {
                expression: `object.spec.template.spec.containers.all(
    c, has(c.securityContext)
)`,
                message: "Security context is missing.",
            },
            {
                expression: `object.spec.template.spec.containers.all(
c, has(c.readinessProbe) && has(c.livenessProbe)
)`,
                message: "No health checks configured.",
            },
            {
                expression: `object.spec.template.spec.containers.all(
    c, !('securityContext' in c) || !('privileged' in c.securityContext) || c.securityContext.privileged == params.spec.privileged
)`,
                message: "Privileged containers are not allowed.",
            },
            {
                expression: `object.spec.template.spec.initContainers.all(
    c, !('securityContext' in c) || !('privileged' in c.securityContext) || c.securityContext.privileged == params.spec.privileged
)`,
                message: "Privileged init containers are not allowed.",
            }]
    }
}, {provider: provider});

const prodReadyPolicyBindingParam = new k8s.admissionregistration.v1alpha1.ValidatingAdmissionPolicyBinding("pulumi-validating-admission-policy-prod-ready-binding-param", {
    metadata: {
        name: "prod-ready-policy-binding-param",
    },
    spec: {
        policyName: prodReadyPolicyParam.metadata.name,
        validationActions: ["Deny"],
        paramRef: {
            name: prodReadyCR.metadata.name,
            namespace: prodReadyCR.metadata.namespace,
        },
        matchResources: {
            objectSelector: {
                matchLabels: {
                    "environment": "production",
                }
            }
        }
    }
}, {provider: provider});
