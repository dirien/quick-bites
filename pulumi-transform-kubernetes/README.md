# Kubernetes and Pulumi: Converting k8s YAML to a Pulumi supported language

## Introduction

%[https://twitter.com/surajincloud/status/1608006302390681600?s=20&t=LSqTzbLpgREOsGfVIZ49nw] 

This tweet and the corresponding [blog post](https://surajincloud.com/kubernetes-and-terraform-converting-yaml-to-hcl-for-better-automation-1299f8c4657b) from my friend `Suraj Narwade` inspired me to write this blog. Surak wrote about how to convert `Kubernetes manifests` to `HCL` for `Terraform`. So I thought: Why not show how to do this in `Pulumi`!

## Why convert Kubernetes YAML to Pulumi?

`Pulumi` is a multi-language `Infrastructure as Code` tool using imperative languages to create a declarative infrastructure description.

You have a wide range of programming languages available, and you can use the one you and your team are the most comfortable with. Currently, (12/2022) `Pulumi` supports the following languages:

* Node.js (JavaScript / TypeScript)

* Python

* Go

* Java

* .NET (C#, VB, F#)

* YAML


Converting your Kubernetes YAML to a Pulumi language will give you the ability to use the full power of a programming language. No more YAML indentation and syntax errors, like mixing arrays and objects. And you may already use Pulumi for your infrastructrure deployments. So having all aligned in Pulumi programs, helps to create a standardised organisation and you can reuse your existing toolchain and processes.

## How to convert Kubernetes YAML to Pulumi languages?

For this, there is an open-source tool called `kube2pulumi` created by the good folks at  
`Pulumi`. There is even a [web app](https://www.pulumi.com/kube2pulumi/) available to convert your YAML to Pulumi.

%[https://www.pulumi.com/kube2pulumi/] 

In this article, I will show how to use the CLI only!

%[https://github.com/pulumi/kube2pulumi] 

### Install `kube2pulumi`

I use `Homebrew` to install `kube2pulumi`, but you can also download the binary from the [release page](https://github.com/pulumi/kube2pulumi/releases/tag/v0.0.12) for your platform.

```bash
brew install pulumi/tap/kube2pulumi
```

### Run `kube2pulumi`

Now that we have `kube2pulumi` installed, we can run it. Let's try some examples I took from the Kubernetes documentation.

#### StatefulSet

I took the following example from the official [Kubernetes documentation](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#components) page:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  ports:
    - port: 80
      name: web
  clusterIP: None
  selector:
    app: nginx
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  selector:
    matchLabels:
      app: nginx # has to match .spec.template.metadata.labels
  serviceName: "nginx"
  replicas: 3 # by default is 1
  minReadySeconds: 10 # by default is 0
  template:
    metadata:
      labels:
        app: nginx # has to match .spec.selector.matchLabels
    spec:
      terminationGracePeriodSeconds: 10
      containers:
        - name: nginx
          image: registry.k8s.io/nginx-slim:0.8
          ports:
            - containerPort: 80
              name: web
          volumeMounts:
            - name: www
              mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
    - metadata:
        name: www
      spec:
        accessModes: [ "ReadWriteOnce" ]
        storageClassName: "my-storage-class"
        resources:
          requests:
            storage: 1Gi
```

Now calling `kube2pulumi` with `go` as an argument to convert the YAML to GoLang:

```bash
kube2pulumi go -f statefulset.yaml
```

And I get instantly the following `Pulumi` program code, ready to apply:

```go
package main

import (
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		_, err := corev1.NewService(ctx, "nginxService", &corev1.ServiceArgs{
			ApiVersion: pulumi.String("v1"),
			Kind:       pulumi.String("Service"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("nginx"),
				Labels: pulumi.StringMap{
					"app": pulumi.String("nginx"),
				},
			},
			Spec: &corev1.ServiceSpecArgs{
				Ports: corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Port: pulumi.Int(80),
						Name: pulumi.String("web"),
					},
				},
				ClusterIP: pulumi.String("None"),
				Selector: pulumi.StringMap{
					"app": pulumi.String("nginx"),
				},
			},
		})
		if err != nil {
			return err
		}
		_, err = appsv1.NewStatefulSet(ctx, "wwwStatefulSet", &appsv1.StatefulSetArgs{
			ApiVersion: pulumi.String("apps/v1"),
			Kind:       pulumi.String("StatefulSet"),
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("web"),
			},
			Spec: &appsv1.StatefulSetSpecArgs{
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app": pulumi.String("nginx"),
					},
				},
				ServiceName:     pulumi.String("nginx"),
				Replicas:        pulumi.Int(3),
				MinReadySeconds: pulumi.Int(10),
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app": pulumi.String("nginx"),
						},
					},
					Spec: &corev1.PodSpecArgs{
						TerminationGracePeriodSeconds: pulumi.Int(10),
						Containers: corev1.ContainerArray{
							&corev1.ContainerArgs{
								Name:  pulumi.String("nginx"),
								Image: pulumi.String("registry.k8s.io/nginx-slim:0.8"),
								Ports: corev1.ContainerPortArray{
									&corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(80),
										Name:          pulumi.String("web"),
									},
								},
								VolumeMounts: corev1.VolumeMountArray{
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("www"),
										MountPath: pulumi.String("/usr/share/nginx/html"),
									},
								},
							},
						},
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaimTypeArgs{
					&corev1.PersistentVolumeClaimTypeArgs{
						Metadata: &metav1.ObjectMetaArgs{
							Name: pulumi.String("www"),
						},
						Spec: &corev1.PersistentVolumeClaimSpecArgs{
							AccessModes: pulumi.StringArray{
								pulumi.String("ReadWriteOnce"),
							},
							StorageClassName: pulumi.String("my-storage-class"),
							Resources: &corev1.ResourceRequirementsArgs{
								Requests: pulumi.StringMap{
									"storage": pulumi.String("1Gi"),
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}
		return nil
	})
}
```

#### Deployment

Let's try another example, this time with a Deployment! The example is also from the [official documentation](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#creating-a-deployment).

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
```

Again, we call the `kube2pulumi` tool to generate the `Pulumi` program. This time we will use `python` as the target language.

```bash
kube2pulumi python -f deployment.yaml
```

The output is the following:

```python
import pulumi
import pulumi_kubernetes as kubernetes

nginx_deployment_deployment = kubernetes.apps.v1.Deployment("nginx_deploymentDeployment",
                                                            api_version="apps/v1",
                                                            kind="Deployment",
                                                            metadata=kubernetes.meta.v1.ObjectMetaArgs(
                                                                name="nginx-deployment",
                                                                labels={
                                                                    "app": "nginx",
                                                                },
                                                            ),
                                                            spec=kubernetes.apps.v1.DeploymentSpecArgs(
                                                                replicas=3,
                                                                selector=kubernetes.meta.v1.LabelSelectorArgs(
                                                                    match_labels={
                                                                        "app": "nginx",
                                                                    },
                                                                ),
                                                                template=kubernetes.core.v1.PodTemplateSpecArgs(
                                                                    metadata=kubernetes.meta.v1.ObjectMetaArgs(
                                                                        labels={
                                                                            "app": "nginx",
                                                                        },
                                                                    ),
                                                                    spec=kubernetes.core.v1.PodSpecArgs(
                                                                        containers=[kubernetes.core.v1.ContainerArgs(
                                                                            name="nginx",
                                                                            image="nginx:1.14.2",
                                                                            ports=[kubernetes.core.v1.ContainerPortArgs(
                                                                                container_port=80,
                                                                            )],
                                                                        )],
                                                                    ),
                                                                ),
                                                            ))
```

### Convert multiple Kubernetes YAML files at once!

`kube2pulumi` can also convert multiple Kubernetes YAML files at once. We use the same Kubernetes examples as before, but this time we do not pass a single YAML file but the directory containing the YAML files.

Side note: We will use `typescript` as the target language.

```bash
kube2pulumi typescript -o myapp.ts
```

The output is the following:

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as kubernetes from "@pulumi/kubernetes";

const nginx_deploymentDeployment = new kubernetes.apps.v1.Deployment("nginx_deploymentDeployment", {
    apiVersion: "apps/v1",
    kind: "Deployment",
    metadata: {
        name: "nginx-deployment",
        labels: {
            app: "nginx",
        },
    },
    spec: {
        replicas: 3,
        selector: {
            matchLabels: {
                app: "nginx",
            },
        },
        template: {
            metadata: {
                labels: {
                    app: "nginx",
                },
            },
            spec: {
                containers: [{
                    name: "nginx",
                    image: "nginx:1.14.2",
                    ports: [{
                        containerPort: 80,
                    }],
                }],
            },
        },
    },
});
const nginxService = new kubernetes.core.v1.Service("nginxService", {
    apiVersion: "v1",
    kind: "Service",
    metadata: {
        name: "nginx",
        labels: {
            app: "nginx",
        },
    },
    spec: {
        ports: [{
            port: 80,
            name: "web",
        }],
        clusterIP: "None",
        selector: {
            app: "nginx",
        },
    },
});
const wwwStatefulSet = new kubernetes.apps.v1.StatefulSet("wwwStatefulSet", {
    apiVersion: "apps/v1",
    kind: "StatefulSet",
    metadata: {
        name: "web",
    },
    spec: {
        selector: {
            matchLabels: {
                app: "nginx",
            },
        },
        serviceName: "nginx",
        replicas: 3,
        minReadySeconds: 10,
        template: {
            metadata: {
                labels: {
                    app: "nginx",
                },
            },
            spec: {
                terminationGracePeriodSeconds: 10,
                containers: [{
                    name: "nginx",
                    image: "registry.k8s.io/nginx-slim:0.8",
                    ports: [{
                        containerPort: 80,
                        name: "web",
                    }],
                    volumeMounts: [{
                        name: "www",
                        mountPath: "/usr/share/nginx/html",
                    }],
                }],
            },
        },
        volumeClaimTemplates: [{
            metadata: {
                name: "www",
            },
            spec: {
                accessModes: ["ReadWriteOnce"],
                storageClassName: "my-storage-class",
                resources: {
                    requests: {
                        storage: "1Gi",
                    },
                },
            },
        }],
    },
});
```

## Conclusion

As you can see, `kube2pulumi` is a great way to convert your existing Kubernetes YAML files to `Pulumi` programs and in the programming language of your choice! `kube2pulumi` will bootstrap a new `Pulumi` program for us! Ready to go!
