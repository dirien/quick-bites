# Kubernetes Validating Admission Policy with Pulumi and Scaleway

## Introduction

In this blog article, we will discover how we can leverage Pulumi and the `kubernetes` provider to write and deploy a
Validating Admission Policy in Kubernetes during the creation of a Kubernetes cluster.

This use case is interesting for a variety of reasons. It enables immediate installation of essential policies right
after the initiation of the cluster. This not only ensures alignment with your set of company rules from the outset but
also guarantees some kind of peace of mind. You can rest assured that any tools or services deployed on the cluster
subsequently will abide by these policies. The application of policies takes precedence over any service deployment,
including potential policy tools like Kyverno or Gatekeeper. This is crucial because the sequence of subsequent tool
deployments can't always be relied upon, thus it's imperative that these policies are established before anything else
is deployed.

This feature is currently in alpha and only available in
Kubernetes [1.26](https://kubernetes.io/blog/2022/12/20/validating-admission-policies-alpha/) and above and requires
that the `ValidatingAdmissionPolicy` feature gate is enabled. Depending on the way you deploy your cluster, this might
be the case already. For example, if you use the `kubeadm` tool to bootstrap your cluster, you can enable the feature
gate by adding the following to your `kubeadm-config.yaml` file:

```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
featureGates:
  ValidatingAdmissionPolicy: true
```

But this is not topic of this article. We will focus on provision a managed Kubernetes cluster with the possibility to
enable the feature gate during the creation of the cluster. Recently `Scaleway` added the `ValidatingAdmissionPolicy` to
their list of supported features.

To see all the features supported by Scaleway, you can run the following command using the `scw` CLI:

```shell
scw k8s version get 1.26.0
Name    1.26.0
Label   Kubernetes 1.26.0
Region  fr-par

Available Kubelet Arguments:
map[containerLogMaxFiles:uint16 containerLogMaxSize:quantity cpuCFSQuota:bool cpuCFSQuotaPeriod:duration cpuManagerPolicy:enum:none|static enableDebuggingHandlers:bool imageGCHighThresholdPercent:uint32 imageGCLowThresholdPercent:uint32 maxPods:uint16]

Available CNIs:
[cilium calico kilo]

Available Ingresses:
[none]

Available Container Runtimes:
[containerd]

Available Feature Gates:
[HPAScaleToZero GRPCContainerProbe ReadWriteOncePod ValidatingAdmissionPolicy CSINodeExpandSecret]

Available Admission Plugins:
[PodNodeSelector AlwaysPullImages PodTolerationRestriction]
```

As you can see, the `ValidatingAdmissionPolicy` feature gate is available, we just have to keep in mind to enable it in
our Pulumi code.

It is also worth mentioning that the `ValidatingAdmissionPolicy` uses CEL (Common Expression Language) to declare the
rules of the policy.

### CEL?

> The Common Expression Language (CEL) implements common semantics for expression evaluation, enabling different
> applications to more easily interoperate.

In short, CEL is a language that allows you to write expressions that can be evaluated. It is capable of creating
complex policies that can be used in a variety of use cases with dealing with Webhooks at all.

In my former blog article, I wrote how to write a DIY policy engine using
webhooks https://blog.kubesimplify.com/diy-how-to-build-a-kubernetes-policy-engine

## Prerequisites

To follow along with this tutorial, you will need the following:

- A [Scaleway account](https://console.scaleway.com/)
- Pulumi CLI installed on your machine ([installation instructions](https://www.pulumi.com/docs/get-started/install/))
- [Free Pulumi account](https://app.pulumi.com/signup)

Optionally, you can also install the following tools and

- [node.js](https://nodejs.org/en/download/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [The Scaleway CLI](https://github.com/scaleway/scaleway-cli)

## Create a new Pulumi project

You may spotted it, that we will use `TypeScript` as programming language for this project. But feel free to use any of
the Pulumi supported languages. To create a new Pulumi project, run the following command:

```shell
mkdir pulumi-validating-admission-policy
cd pulumi-validating-admission-policy
pulumi new typescript --force
```

You will be prompted with a wizard to create a new Pulumi project. You can keep the default values for all questions
except the last one.

> Note: You may have to run `pulumi login` before you can create a new project, depending if you already have a Pulumi
> account or not.

```shell
 pulumi new typescript --force
This command will walk you through creating a new Pulumi project.

Enter a value or leave blank to accept the (default), and press <ENTER>.
Press ^C at any time to quit.

project name: (pulumi-validating-admission-policy) 
project description: (A minimal TypeScript Pulumi program) 
Created project 'pulumi-validating-admission-policy'

Please enter your desired stack name.
To create a stack in an organization, use the format <org-name>/<stack-name> (e.g. `acmecorp/dev`).
stack name: (dev) 
Created stack 'dev'

Installing dependencies...


added 193 packages, and audited 194 packages in 14s

65 packages are looking for funding
  run `npm fund` for details

found 0 vulnerabilities
Finished installing dependencies

Your new project is ready to go! ✨

To perform an initial deployment, run `pulumi up`
```

Now we can add the `scaleway` provider to our project. To do so, run the following command:

```shell
npm install @ediri/scaleway@2.25.1 --save-exact
```

And we need also the `kubernetes` provider:

```shell
npm install @pulumi/kubernetes@4.0.3 --save-exac
```

> I like to use the `--save-exact` flag to ensure that the exact version of the provider is used. Reduce the risk of
> potential breaking changes/bugs, if a new version of the provider is released. But feel free to omit this flag.

## Create a new Kubernetes cluster

After having the libraries ready, we can add some configuratrion to your `Pulumi.yaml` file. We will add the following
configuration:

```yaml
name: pulumi-validating-admission-policy
...
config:
  scaleway:region: "fr-par"
  scaleway:zone: "fr-par-1"
  cluster:version: "1.27"
  cluster:auto_upgrade: true
  node:node_type: "PLAY2-NANO"
  node:auto_scale: false
  node:node_count: 1
  node:auto_heal: true
```

This set some default values for our cluster, like the region, zone, version, node type, node count and so on. You can
keep the default values or change them to your needs. Every Pulumi `stack` can have its own configuration values and
override the default values.

After having the configuration in place, we can add the following code to our `index.ts` file:

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as scaleway from "@ediri/scaleway";

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

new scaleway.K8sPool("pulumi-validating-admission-policy-node-pool", {
    nodeType: nodeConfig.require("node_type"),
    size: nodeConfig.requireNumber("node_count"),
    autoscaling: nodeConfig.requireBoolean("auto_scale"),
    autohealing: nodeConfig.requireBoolean("auto_heal"),
    clusterId: kapsule.id,
});

export const kapsuleName = kapsule.name;
export const kubeconfig = pulumi.secret(kapsule.kubeconfigs[0].configFile);
```

Before we start to deploy our Validating Admission Policy, let's have a look at the code. The first resource we create
is a `K8sCluster`. This resource creates a new Kubernetes cluster on Scaleway. We use some of the configuration values
to set some of the properties of the cluster. The `featureGates` property is the one we are interested in. Here we
activate the `ValidatingAdmissionPolicy` plugin.

The second resource we create is a `K8sPool`. This resource creates a new node pool for our cluster. Next to the
definition of the node type, aka the instance type, we also set the `size` property to our `node_count` configuration
value.

The last two lines of the code are used to export the name of the cluster and the kubeconfig.

## Write some Validating Admission Policies

Now that we have our cluster defines, we can head over to create some example Validating Admission Policies.

First we create a Pulumi Kubernetes provider. This provider gets the kubeconfig from our cluster and uses it to connect
to the cluster.

```typescript
import * as k8s from "@pulumi/kubernetes";

// omitted code for brevity

const provider = new k8s.Provider("k8s-provider", {
    kubeconfig: kubeconfig,
}, {dependsOn: [kapsule, nodePool]});
```

As next step, we can finally start to write our first Validating Admission Policy. We will do for this example a simple
check, if the team label is set on `Deployment` and `StatefulSet` resources. If not, we will reject the resource
creation and return an error message.

```typescript
const teamLabel = new k8s.admissionregistration.v1alpha1.ValidatingAdmissionPolicy("pulumi-validating-admission-policy-0", {
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
```

There is a lot of code in this example, so let's have a more in-depth look at it:

- We set the `failurePolicy` to `Fail`. This means that the resource creation will fail, if the validation fails. The
  other option is `Ignore`, which will ignore the validation and create the resource.
- The `matchConstraints` property defines the resources, which should be validated. In our case, we want to validate
  `Deployment` and `StatefulSet` resources.
- The `matchConditions` property defines the conditions, which should be met, to run the validation. In our case, we
  want to validate only resources in namespaces, which are not `kube-*`. This helps us to even fine-grained control over
  the resources, which should be validated.
- Last but not least, the `validations` property defines the actual validation. In our case, we check if the `team`
  label is set on the resource and the pod template.

Now we need to connect the Validating Admission Policy to a specific scope or context. We can do this by creating
a `ValidatingAdmissionPolicyBinding` resource.

> **Note:** You can bind one Validating Admission Policy to multiple scopes or contexts.

```typescript
const teamLabelBinding = new k8s.admissionregistration.v1alpha1.ValidatingAdmissionPolicyBinding("pulumi-validating-admission-policy-1", {
    metadata: {
        name: "team-label-binding",
    },
    spec: {
        policyName: teamLabel.metadata.name,
        validationActions: ["Deny"],
        matchResources: {}
    }
}, {provider: provider});
```

Here we set the `policyName` to the name of the Validating Admission Policy we created before. One interesting property
is the `validationActions` property. Here we can define, how validations of a policy should be enforced. We have
the following options:

- `Audit`: This will only audit the validation and not enforce it and will be added to the audit event.
- `Deny`: This will deny the request and return an error message.
- `Warn`: This will warn the user, but will not deny the request.

The last property we need to set is the `matchResources` property. This property defines the resources, in our case the
value and this means all resources, which should be validated by the Validating Admission Policy.

## Deploy the cluster and the Validating Admission Policies

Now that we have our cluster and our Validating Admission Policies defined, we can deploy them. To do so, run following
command:

```shell
pulumi up
Previewing update (dev)

     Type                                                                                  Name                                                   Plan       
 +   pulumi:pulumi:Stack                                                                   pulumi-validating-admission-policy-dev                 create     
 +   ├─ scaleway:index:K8sCluster                                                          pulumi-validating-admission-policy-cluster             create     
 +   ├─ scaleway:index:K8sPool                                                             pulumi-validating-admission-policy-node-pool           create     
 +   ├─ pulumi:providers:kubernetes                                                        k8s-provider                                           create     
 +   ├─ kubernetes:admissionregistration.k8s.io/v1alpha1:ValidatingAdmissionPolicy         pulumi-validating-admission-policy-team-label          create     
 +   └─ kubernetes:admissionregistration.k8s.io/v1alpha1:ValidatingAdmissionPolicyBinding  pulumi-validating-admission-policy-binding-team-label  create     


Outputs:
    kapsuleName: "pulumi-validating-admission-policy-cluster-d786e2b"
    kubeconfig : output<string>

Resources:
    + 6 to create

Do you want to perform this update?  [Use arrows to move, type to filter]
  yes
> no
  details
```

And confirm the deployment with `yes`. After a few minutes, the deployment should be finished and you should see similar
output:

```shell
Do you want to perform this update? yes
Updating (dev)
...

Resources:
    + 6 created

Duration: 5m39s
```

## Test the Validating Admission Policy

Now that we have our cluster and our Validating Admission Policies deployed, we can test them. To do so, we will create
a `Deployment` resource without the `team` label. First we need to get the kubeconfig from the cluster. We can do this
by running following command:

```shell
pulumi stack output kubeconfig --show-secrets -s dev > kubeconfig
```

Create a new file called `deployment.yaml` and add following content:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  labels:
    environment: production
spec:
  selector:
    matchLabels:
      app: guestbook
      tier: frontend
  replicas: 2
  template:
    metadata:
      labels:
        app: guestbook
        tier: frontend
    spec:
      containers:
        - name: php-redis
          image: gcr.io/google-samples/gb-frontend:v4
          resources:
            requests:
              cpu: 100m
              memory: 100Mi
            limits:
              cpu: "3"
              memory: 4Gi
```

Now we can create the `Deployment` resource by running following command:

```shell
kubectl apply -f deployment.yaml --kubeconfig kubeconfig
```

This should fail with following error message:

```shell            
The deployments "frontend" is invalid: : ValidatingAdmissionPolicy 'team-label' with binding 'team-label-binding' denied request: Team label is missing.
```

## Create a more complex Validating Admission Policy

In the previous example, we created a simple Validating Admission Policy, which checks if the `team` label is set.
Useful but still pretty basic.

Let us build a more complex Validating Admission Policy. We will check in the next policy several different rules for
the `Deployment` resource:

```typescript
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
```

Let us go through this "monster" step by step and we will focus here only on the `validations` property as the rest is
similar to the previous example:

- `object.spec.template.spec.containers.all(c, has(c.resources) && has(c.resources.limits) && has(c.resources.limits.cpu))`:
  This expression checks if all containers have a CPU resource limit set.

- `object.spec.template.spec.containers.all(c, has(c.resources) && has(c.resources.limits) && has(c.resources.limits.memory))`:
  This expression checks if all containers have a memory resource limit set.

- `object.spec.template.spec.containers.all(c, !c.image.endsWith(':latest'))`: This expression checks if the image tag
  is not `latest`. We enforce this to make sure that we do not use the `latest` tag in production.

- `object.spec.template.spec.containers.all(c, c.image.startsWith('myregistry.azurecr.io'))`: This expression checks if
  the image is pulled from `myregistry.azurecr.io`. This can be useful if you want to enforce that only images from a
  specific registry are used.

- `object.spec.template.spec.containers.all(c, has(c.securityContext))`: This expression checks if all containers have a
  security context set.

- `object.spec.template.spec.containers.all(c, has(c.readinessProbe) && has(c.livenessProbe))`: This expression checks
  if all containers have a readiness and liveness probe configured.

- `object.spec.template.spec.containers.all(c, !('securityContext' in c) || !('privileged' in c.securityContext) || c.securityContext.privileged == false)`:
  This expression checks that no container is privileged.

- `object.spec.template.spec.initContainers.all(c, !('securityContext' in c) || !('privileged' in c.securityContext) || c.securityContext.privileged == false)`:
  This expression checks that no init container is privileged.

## Parameter resources

Now we come to an more advanced topic. In the previous example, we hard coded the policy configuration within the
definition. With the property `paramKind` we can define a custom Kubernetes resource which contains the policy
configuration.

First we need to create a Custom Resource Definition (CRD) for our parameter resource:

```typescript
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
```

The CRD defines the schema of the parameter resource. In our case, we define a schema with three properties:

- `registry`: The registry from which the image must be pulled.
- `version`: The image version to block.
- `privileged`: If the privileged flag is allowed.

And then we create a CR of this type.

Now we can use this CR in our policy, for this example I will create a new policy and binding

```typescript
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
```

The policy is the same as in the previous example, but now we use the parameter resource in the policy and in the
binding.

> **Note:** To get better error messages, we use the `messageExpression` property instead of the `message` property.

Now we can test the new policy by creating a deployment with a container that violates the policy and we should see a
message similar to this:

```shell
The deployments "frontend" is invalid: : ValidatingAdmissionPolicy 'prod-ready-policy-param' with binding 'prod-ready-policy-binding-param' denied request: Image tag must not be latest
```

## Housekeeping

To clean up all resources created by this example, run following command:

```shell
pulumi destroy
```

## Conclusion

We have seen how to use the `ValidatingAdmissionPolicy` resource to create a policy that validates the resources using
CEL expressions. We have also seen how to use parameter resources to make the policy more flexible and reusable.

It is very nice to have a policy functionality inbuilt in Kubernetes without the need to install a separate policy
engine. Especially when you want to ensure that the policy right from the start of the cluster available.

And: You don't need to handle Webhooks, which is a big plus for me. You can easily render your cluster usless when your
webhook is not available or not working correctly.

## Links

- https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/#getting-started-with-validating-admission-policy
- https://minikube.sigs.k8s.io/docs/handbook/config/#:~:text=in%20constants.go-,Enabling%20feature%20gates,is%20the%20status%20of%20it
- https://www.pulumi.com/registry/packages/kubernetes/api-docs/admissionregistration/v1alpha1/
