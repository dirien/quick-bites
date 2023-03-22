# Minecraft Server: Secrets, Observability, Kubernetes and more with Pulumi and Scaleway

## Introduction

In this tutorial, I want to use some new Beta services from Scaleway to deploy a Minecraft server on Kubernetes.
To deploy the infrastructure I will be using Pulumi, a modern infrastructure as code tool. Because, nobody wants to
manage infrastructure by hand, right?

So what are the new Scaleway services we will be using? For the whole observability part we're going to
use [Cockpit](https://www.scaleway.com/en/docs/observability/cockpit/). Cockpit is
the new monitoring and logging tool from Scaleway. To handle the secrets we will use `external-secrets` Kubernetes
operator and as backed we will use the
new [Secret Manager](https://www.scaleway.com/en/docs/identity-and-access-management/secret-manager/) from Scaleway.

And yes, we will use the new Scaleway Managed Kubernetes Service
called [Kapsule](https://www.scaleway.com/en/docs/containers/kubernetes/) as runtime for all of this.

So without further ado, let's get started!

## Prerequisites

If you want to follow along, you need to have the following installed:

- [Pulumi](https://www.pulumi.com/docs/get-started/install/)
- A Scaleway account with a valid access key and secret key
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) if you want to interact with the Kubernetes cluster

## Setup your Pulumi project

To get things started, we need to create a new Pulumi project. Create a new directory and run `pulumi new` inside of it,
as I am going to use [Go](https://go.dev/) for this tutorial, I will select `go` to use a predefined template for this
blog post.

> You can find more about Pulumi templates [here](https://www.pulumi.com/templates/).

```bash
mkdir pulumi-scaleway-kapsule && cd pulumi-scaleway-kapsule
pulumi new go --force
```

I let all the default values as they are, no need to change anything here.

```bash
This command will walk you through creating a new Pulumi project.

Enter a value or leave blank to accept the (default), and press <ENTER>.
Press ^C at any time to quit.

project name: (pulumi-scaleway-kapsule) 
project description: (A minimal Go Pulumi program) 
Created project 'pulumi-scaleway-kapsule'

Please enter your desired stack name.
To create a stack in an organization, use the format <org-name>/<stack-name> (e.g. `acmecorp/dev`).
stack name: (dev) 
Created stack 'dev'

Installing dependencies...

Finished installing dependencies

Your new project is ready to go! ✨

To perform an initial deployment, run `pulumi up`
```

To use the Scaleway provider, we can add it using the `go get` command:

```
go get github.com/dirien/pulumi-scaleway/sdk/v2
```

And as we deploy several `Helm` charts, we need to add the Kubernetes provider too:

```bash
go get github.com/pulumi/pulumi-kubernetes/sdk/v3
```

And that's it from a Go dependency perspective. Your `go.mod` should look like this:

```go
module pulumi-scaleway-kapsule

go 1.20

require (
github.com/dirien/pulumi-scaleway/sdk/v2 v2.13.1
github.com/pulumi/pulumi-kubernetes/sdk/v3 v3.24.2
github.com/pulumi/pulumi/sdk/v3 v3.58.0
)
```

## Setup your Pulumi stack

Now with the project setup done, we can start to create our infrastructure. Head over to the `main.go` file and add the
following code:

```go
package main

import (
	"fmt"
	"github.com/dirien/pulumi-scaleway/sdk/v2/go/scaleway"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		return nil
	})
}
```

This is the basic structure of a Pulumi program. We have a `main` function which will be called by Pulumi and inside of
this function we can create our infrastructure.

We start by creating a new Scaleway `project`. Scaleway has a concept of projects, which are basically a groupings of
different resources. This is useful if you want to separate different environments like `dev`, `staging` and `prod` and
makes it easier to manage with `IAM`.

```go
package main

// omitting imports

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		project, err := scaleway.NewAccountProject(ctx, "scaleway-project", &scaleway.AccountProjectArgs{
			Name: pulumi.String("pulumi-scaleway-kapsule"),
		})
		if err != nil {
			return err
		}
		return nil
	})
}
```

Now we have a Scaleway `project` created, we can create our `Cockpit` instance. We create also a `Cockpit` token, which
allow you to authenticate against the `Cockpit` API. We can select the token permissions on creation too.

Following permissions are available:

* Push: allows you to send your metrics and logs to your Cockpit.
* Query: allows you to fetch your metrics and logs from your Cockpit.
* Rules: allow you to configure alerting and recording rules.
* Alerts: allow you to set up the alert manager.

Cockpit uses under the hood Cortex (Metrics and Alertmanager) and Loki. You get for all of them dedicated API URLs:

* Cortex: https://metrics.prd.obs.fr-par.scw.cloud/api/v1/push
* Loki: https://logs.prd.obs.fr-par.scw.cloud/loki/api/v1/push
* Alertmanager: https://alertmanager.prd.obs.fr-par.scw.cloud

The finale resource, we create in the `Cockpit` context is a local Grafana user. We use this user to authenticate
against the managed Grafana instance from Scaleway. There are two roles you can select:

* Editor: allows you to edit dashboards and create new ones.
* Viewer: allows you to only view dashboards.

```go
package main

// omitting imports

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		project, err := scaleway.NewAccountProject(ctx, "scaleway-project", &scaleway.AccountProjectArgs{
			Name: pulumi.String("pulumi-scaleway-kapsule"),
		})
		if err != nil {
			return err
		}
		cockpit, err := scaleway.NewCockpit(ctx, "scaleway-cockpit", &scaleway.CockpitArgs{
			ProjectId: project.ID(),
		})
		if err != nil {
			return err
		}
		cockpitToken, err := scaleway.NewCockpitToken(ctx, "scaleway-cockpit-token", &scaleway.CockpitTokenArgs{
			Name:      pulumi.String("cockpit-token"),
			ProjectId: cockpit.ProjectId,
			Scopes: scaleway.CockpitTokenScopesArgs{
				QueryLogs:    pulumi.Bool(true),
				WriteLogs:    pulumi.Bool(true),
				QueryMetrics: pulumi.Bool(true),
				WriteMetrics: pulumi.Bool(true),
			},
		})
		if err != nil {
			return err
		}

		user, err := scaleway.NewCockpitGrafanaUser(ctx, "scaleway-cockpit-grafana-user", &scaleway.CockpitGrafanaUserArgs{
			ProjectId: cockpit.ProjectId,
			Role:      pulumi.String("editor"),
			Login:     pulumi.String("pulumi"),
		})
		if err != nil {
			return err
		}
		ctx.Export("grafana-password", pulumi.ToSecret(user.Password))
		return nil
	})
}
```

With the Scaleway `Cockpit` service defined, we can now create the Scaleway Secret Manager. We will use this service to
store our Minecraft RCON password. To retrieve the password in our Kubernetes cluster, we will use later
the `external-secrets` Operator. We will also create a dedicated IAM user and a dedicated IAM policy for fetching the
secret.

Keep in mind to change the `please-change-me` password later on the Console, I just need this to get the example working
as the Helm chart to deploy the Minecraft server requires a Kubernetes secret with the RCON password.

The IAM API key will be used by the `external-secrets` Operator to authenticate against the Scaleway Secret Manager API.

```go
package main

// omitting imports

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// omitting previous code

		secret, err := scaleway.NewSecret(ctx, "scaleway-secret", &scaleway.SecretArgs{
			Name:      pulumi.String("scaleway-secret"),
			ProjectId: project.ID(),
		})
		if err != nil {
			return err
		}

		_, err = scaleway.NewSecretVersion(ctx, "scaleway-secret-version", &scaleway.SecretVersionArgs{
			SecretId: secret.ID(),
			Data:     pulumi.String("please-change-me"),
		})
		if err != nil {
			return err
		}

		iamApplication, err := scaleway.NewIamApplication(ctx, "scaleway-iam-application", &scaleway.IamApplicationArgs{
			Name: pulumi.String("pulumi-application"),
		})
		if err != nil {
			return err
		}
		_, err = scaleway.NewIamPolicy(ctx, "scaleway-iam-policy", &scaleway.IamPolicyArgs{
			Name:          pulumi.String("pulumi-scaleway-iam-policy"),
			ApplicationId: iamApplication.ID(),
			Rules: scaleway.IamPolicyRuleArray{
				&scaleway.IamPolicyRuleArgs{
					ProjectIds: pulumi.StringArray{
						project.ID(),
					},
					PermissionSetNames: pulumi.StringArray{
						pulumi.String("SecretManagerFullAccess"),
					},
				},
			},
		})
		if err != nil {
			return err
		}
		key, err := scaleway.NewIamApiKey(ctx, "scaleway-iam-api-key", &scaleway.IamApiKeyArgs{
			ApplicationId: iamApplication.ID(),
		})
		if err != nil {
			return err
		}
		return nil
	})
}
```

> **Note:** The IAM Policy permission is set to `SecretManagerFullAccess`

Finally, we can create our Kubernetes cluster. We will use the Scaleway Kapsule service to create a managed Kubernetes
cluster and add a node pool to it. There are a lot of options available to configure the cluster and the node pool. I
tried to select the most important ones.

Feel free to change them to your needs.

```go
package main

// omitting imports

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// omitting previous code

		k8sCluster, err := scaleway.NewK8sCluster(ctx, "k8s-cluster", &scaleway.K8sClusterArgs{
			Name:    pulumi.String("pulumi-scaleway-kapsule"),
			Version: pulumi.String("1.26"),
			Cni:     pulumi.String("cilium"),
			AutoUpgrade: scaleway.K8sClusterAutoUpgradeArgs{
				Enable:                     pulumi.Bool(true),
				MaintenanceWindowDay:       pulumi.String("sunday"),
				MaintenanceWindowStartHour: pulumi.Int(3),
			},
			AdmissionPlugins: pulumi.StringArray{
				pulumi.String("AlwaysPullImages"),
			},
			DeleteAdditionalResources: pulumi.Bool(true),
			ProjectId:                 project.ID(),
		})
		if err != nil {
			return err
		}
		pool, err := scaleway.NewK8sPool(ctx, "k8s-pool", &scaleway.K8sPoolArgs{
			Name:        pulumi.String("pulumi-scaleway-kapsule-pool"),
			ClusterId:   k8sCluster.ID(),
			NodeType:    pulumi.String("PLAY2-MICRO"),
			Autoscaling: pulumi.BoolPtr(true),
			MinSize:     pulumi.Int(1),
			MaxSize:     pulumi.Int(3),
			Size:        pulumi.Int(1),
			Autohealing: pulumi.BoolPtr(true),
		})
		if err != nil {
			return err
		}
		return nil
	})
}
```

## Deploying the Observability Stack

Now with the infrastructure in place, we can deploy the observability stack onto our recently created Kubernetes
cluster. For this we will use the `pulumi-kubernetes` provider as it offers us a handy way to deploy Helm charts.
The `helm.Release` resource is our key component here.

Our observability stack will consist of the following components:

- `kube-prometheus-stack`, but without deploying Grafana. We will use the Scaleway Cockpit Grafana instance instead.
- `promtail`, to collect logs from our Kubernetes cluster.

Both stacks are configured to use the `remote write` feature to forward metrics and logs to the Scaleway Cockpit
service.
And as both endpoints are protected, we have to use the IAM API key we created earlier to authenticate against it.

```go
package main

// omitting imports

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// omitting previous code

		kubernetesProvider, err := kubernetes.NewProvider(ctx, "k8s-provider", &kubernetes.ProviderArgs{
			Kubeconfig:            k8sCluster.Kubeconfigs.Index(pulumi.Int(0)).ConfigFile(),
			EnableServerSideApply: pulumi.Bool(true),
		}, pulumi.DependsOn([]pulumi.Resource{k8sCluster, pool}))
		if err != nil {
			return err
		}

		kubePrometheusStack, err := helm.NewRelease(ctx, "kube-prometheus-stack", &helm.ReleaseArgs{
			Name:  pulumi.String("kube-prometheus-stack"),
			Chart: pulumi.String("kube-prometheus-stack"),
			RepositoryOpts: helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://prometheus-community.github.io/helm-charts"),
			},
			Namespace:       pulumi.String("monitoring"),
			Version:         pulumi.String("45.7.1"),
			CreateNamespace: pulumi.BoolPtr(true),
			Values: pulumi.Map{
				"grafana": pulumi.Map{
					"enabled": pulumi.Bool(false),
				},
				"kube-state-metrics": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
				"prometheus-node-exporter": pulumi.Map{
					"enabled": pulumi.Bool(true),
					"prometheus": pulumi.Map{
						"monitor": pulumi.Map{
							"enabled": pulumi.Bool(true),
						},
					},
				},
				"prometheus": pulumi.Map{
					"prometheusSpec": pulumi.Map{
						"remoteWrite": pulumi.Array{
							pulumi.Map{
								"url":         pulumi.String("https://metrics.prd.obs.fr-par.scw.cloud/api/v1/push"),
								"bearerToken": cockpitToken.SecretKey,
							},
						},
						"ruleSelectorNilUsesHelmValues":           pulumi.Bool(false),
						"serviceMonitorSelectorNilUsesHelmValues": pulumi.Bool(false),
					},
				},
			},
		}, pulumi.Provider(kubernetesProvider))
		if err != nil {
			return err
		}

		_, err = helm.NewRelease(ctx, "promtail", &helm.ReleaseArgs{
			Name:  pulumi.String("promtail"),
			Chart: pulumi.String("promtail"),
			RepositoryOpts: helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://grafana.github.io/helm-charts"),
			},
			Namespace:       pulumi.String("monitoring"),
			Version:         pulumi.String("6.9.3"),
			CreateNamespace: pulumi.BoolPtr(true),
			Values: pulumi.Map{
				"config": pulumi.Map{
					"clients": pulumi.Array{
						pulumi.Map{
							"url":          pulumi.String("https://logs.prd.obs.fr-par.scw.cloud/loki/api/v1/push"),
							"bearer_token": cockpitToken.SecretKey,
						},
					},
				},
				"serviceMonitor": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
			},
		}, pulumi.Provider(kubernetesProvider), pulumi.DependsOn([]pulumi.Resource{kubePrometheusStack}))
		if err != nil {
			return err
		}
		return nil
	})
}
```

## Deploying the external-secrets operator

Deploying the `external-secrets` operator is a piece of cake. We just need to deploy the Helm chart and leave it much to
the defaults. I just activated the `prometheus` service monitor to get some metrics about the operator itself.

```go
package main

// omitting imports

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// omitting previous code

		externalSecrets, err := helm.NewRelease(ctx, "external-secrets", &helm.ReleaseArgs{
			Name:  pulumi.String("external-secrets"),
			Chart: pulumi.String("external-secrets"),
			RepositoryOpts: helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://charts.external-secrets.io"),
			},
			Namespace:       pulumi.String("external-secrets"),
			Version:         pulumi.String("0.8.1"),
			CreateNamespace: pulumi.BoolPtr(true),
			Values: pulumi.Map{
				"installCRDs": pulumi.Bool(true),
				"serviceMonitor": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
				"webhook": pulumi.Map{
					"serviceMonitor": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
				},
				"certController": pulumi.Map{
					"serviceMonitor": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
				},
			},
		}, pulumi.Provider(kubernetesProvider), pulumi.DependsOn([]pulumi.Resource{secret, kubePrometheusStack}))
		if err != nil {
			return err
		}
		return nil
	})
}
```

## Deploying the Minecraft server

Now we are going to deploy all components required to run a Minecraft server. This consists of the following parts:

- The Namespace `minecraft`, where all components will be deployed.
- The Helm chart for the Minecraft server. Important is here that we will use a sidecar container to run
  our `minecraft-exporter`.
- The `SecretStore` CR, which will be used by the `external-secrets` operator to fetch the secrets from the Scaleway
  Secrets Manager.
- The `ExternalSecret` CR, which will be used by the `external-secrets` operator to create the Kubernetes Secret.
- The `ServiceMonitor` CR, which will be used by Prometheus to scrape the metrics from the `minecraft-exporter`.

```go
package main

// omitting imports

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// omitting previous code
		mcNamespace, err := v1.NewNamespace(ctx, "minecraft", &v1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("minecraft"),
			},
		}, pulumi.Provider(kubernetesProvider))
		if err != nil {
			return err
		}

		mc, err := helm.NewRelease(ctx, "minecraft", &helm.ReleaseArgs{
			Chart:   pulumi.String("minecraft"),
			Version: pulumi.String("4.6.0"),
			RepositoryOpts: &helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://itzg.github.io/minecraft-server-charts"),
			},
			Namespace: mcNamespace.Metadata.Name(),
			Values: pulumi.Map{
				"minecraftServer": pulumi.Map{
					"eula":        pulumi.Bool(true),
					"motd":        pulumi.String("Scaleway and Pulumi: Minecraft Server"),
					"serviceType": pulumi.String("LoadBalancer"),
					"rcon": pulumi.Map{
						"enabled":        pulumi.Bool(true),
						"existingSecret": pulumi.String("minecraft-rcon"),
					},
					"extraPorts": pulumi.Array{
						pulumi.Map{
							"name":          pulumi.String("prom"),
							"containerPort": pulumi.Int(9150),
							"protocol":      pulumi.String("TCP"),
							"service": pulumi.Map{
								"enabled": pulumi.Bool(true),
								"port":    pulumi.Int(9150),
							},
							"ingress": pulumi.Map{
								"enabled": pulumi.Bool(false),
							},
						},
					},
				},
				"persistence": pulumi.Map{
					"dataDir": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
				},
				"sidecarContainers": pulumi.Array{
					pulumi.Map{
						"name":  pulumi.String("minecraft-exporter"),
						"image": pulumi.String("ghcr.io/dirien/minecraft-exporter:0.18.0"),
						"volumeMounts": pulumi.Array{
							pulumi.Map{
								"name":      pulumi.String("datadir"),
								"mountPath": pulumi.String("/data"),
							},
						},
						"env": pulumi.Array{
							pulumi.Map{
								"name":  pulumi.String("MC_WORLD"),
								"value": pulumi.String("/data/world"),
							},
							pulumi.Map{
								"name":  pulumi.String("MC_RCON_ADDRESS"),
								"value": pulumi.String("localhost:25575"),
							},
							pulumi.Map{
								"name": pulumi.String("MC_RCON_PASSWORD"),
								"valueFrom": pulumi.Map{
									"secretKeyRef": pulumi.Map{
										"name": pulumi.String("minecraft-rcon"),
										"key":  pulumi.String("rcon-password"),
									},
								},
							},
						},
					},
				},
			},
		}, pulumi.Provider(kubernetesProvider), pulumi.DependsOn([]pulumi.Resource{mcNamespace, kubePrometheusStack}))
		if err != nil {
			return err
		}

		mcSecretStore, err := apiextensions.NewCustomResource(ctx, "secret-store", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("external-secrets.io/v1beta1"),
			Kind:       pulumi.String("SecretStore"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("secret-store"),
				Namespace: mcNamespace.Metadata.Name(),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"provider": pulumi.Map{
						"scaleway": pulumi.Map{
							"region":    pulumi.String("fr-par"),
							"projectId": project.ID(),
							"accessKey": pulumi.Map{
								"value": key.AccessKey,
							},
							"secretKey": pulumi.Map{
								"value": key.SecretKey,
							},
						},
					},
				},
			},
		}, pulumi.Provider(kubernetesProvider), pulumi.DependsOn([]pulumi.Resource{mcNamespace, externalSecrets}))
		if err != nil {
			return err
		}

		apiextensions.NewCustomResource(ctx, "minecraft-servicemonitor", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("monitoring.coreos.com/v1"),
			Kind:       pulumi.String("ServiceMonitor"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("minecraft"),
				Namespace: mcNamespace.Metadata.Name(),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"selector": pulumi.Map{
						"matchLabels": pulumi.Map{
							"app": pulumi.All(mc.Name, mc.Namespace).ApplyT(func(args []interface{}) string {
								return fmt.Sprintf("%s-%s", *args[0].(*string), *args[1].(*string))
							}),
						},
					},
					"endpoints": pulumi.Array{
						pulumi.Map{
							"targetPort": pulumi.Int(9150),
						},
					},
					"namespaceSelector": pulumi.Map{
						"matchNames": pulumi.Array{
							mcNamespace.Metadata.Name(),
						},
					},
					"jobLabel": pulumi.String("minecraft-exporter"),
				},
			},
		}, pulumi.Provider(kubernetesProvider), pulumi.DependsOn([]pulumi.Resource{mcNamespace, mc, kubePrometheusStack}))

		apiextensions.NewCustomResource(ctx, "external-secrets", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("external-secrets.io/v1beta1"),
			Kind:       pulumi.String("ExternalSecret"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("minecraft-rcon"),
				Namespace: mcNamespace.Metadata.Name(),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"secretStoreRef": pulumi.Map{
						"name": mcSecretStore.Metadata.Name(),
						"kind": mcSecretStore.Kind,
					},
					"target": pulumi.Map{
						"name": pulumi.String("minecraft-rcon"),
					},
					"refreshInterval": pulumi.String("20s"),
					"data": pulumi.Array{
						pulumi.Map{
							"secretKey": pulumi.String("rcon-password"),
							"remoteRef": pulumi.Map{
								"key":     pulumi.Sprintf("name:%s", secret.Name),
								"version": pulumi.String("latest_enabled"),
							},
						},
					},
				},
			},
		}, pulumi.Provider(kubernetesProvider), pulumi.DependsOn([]pulumi.Resource{mcNamespace, externalSecrets}))
		return nil
	})
}
```

With everything in place, we can now run `pulumi up` to deploy our application. This can take a few minutes, so go grab
a coffee or something.

```
➜ pulumi up -y -f        
Updating (dev)

View in Browser (Ctrl+O): https://app.pulumi.com/dirien/pulumi-scaleway-kapsule/dev/updates/1

     Type                                                      Name                           Status              
 +   pulumi:pulumi:Stack                                       pulumi-scaleway-kapsule-dev    created (51s)       
 +   ├─ scaleway:index:AccountProject                          scaleway-project               created (0.65s)     
 +   ├─ scaleway:index:IamApplication                          scaleway-iam-application       created (0.73s)     
 +   ├─ scaleway:index:Secret                                  scaleway-secret                created (0.92s)     
 +   ├─ scaleway:index:K8sCluster                              k8s-cluster                    created (6s)        
 +   ├─ scaleway:index:Cockpit                                 scaleway-cockpit               created (47s)       
 +   ├─ scaleway:index:IamApiKey                               scaleway-iam-api-key           created (1s)        
 +   ├─ scaleway:index:IamPolicy                               scaleway-iam-policy            created (1s)        
 +   ├─ scaleway:index:SecretVersion                           scaleway-secret-version        created (1s)        
 +   ├─ scaleway:index:K8sPool                                 k8s-pool                       created (335s)      
 +   ├─ scaleway:index:CockpitGrafanaUser                      scaleway-cockpit-grafana-user  created (1s)        
 +   ├─ scaleway:index:CockpitToken                            scaleway-cockpit-token         created (0.83s)     
 +   ├─ pulumi:providers:kubernetes                            k8s-provider                   created (0.50s)     
 +   ├─ kubernetes:core/v1:Namespace                           minecraft                      created (2s)        
 +   ├─ kubernetes:helm.sh/v3:Release                          kube-prometheus-stack          created (76s)       
 +   ├─ kubernetes:helm.sh/v3:Release                          promtail                       created (3s)        
 +   ├─ kubernetes:helm.sh/v3:Release                          external-secrets               created (93s)       
 +   ├─ kubernetes:external-secrets.io/v1beta1:SecretStore     secret-store                   created (2s)        
 +   ├─ kubernetes:external-secrets.io/v1beta1:ExternalSecret  external-secrets               created (0.63s)     
 +   ├─ kubernetes:helm.sh/v3:Release                          minecraft                      created (2s)        
 +   └─ kubernetes:monitoring.coreos.com/v1:ServiceMonitor     minecraft-servicemonitor       created (0.67s)     


Outputs:
    grafana-password: [secret]
    kubeconfig      : [secret]

Resources:
    + 21 created

Duration: 9m4s
```

With the deployment finished, we can now access our Minecraft server. To do so, we need to get the IP address of the
LoadBalancer that was created for us. First we need to get our `kubeconfig` file. To do so, we can run the following
command:

```
pulumi stack output kubeconfig --show-secrets -s dev > kubeconfig.yaml
```

After that, we can use the `kubectl` command to get the IP address of the LoadBalancer:

```
kubectl get services --all-namespaces | grep LoadBalancer | awk '{print $5}'
```

## Testing

After having the IP address, we can connect now to our Minecraft server. To do so, we can use the Minecraft client!

Awesome, that worked fine! Now we come to fun part! Let us check the metrics of our Minecraft game server in the
Scaleway Grafana. We need to get the password of the Grafana user. To do so, we can run the following command:

```
pulumi stack output grafana-password --show-secrets -s dev
```

Note this password. The login we set to `pulumi` in our `NewCockpitGrafanaUser` resource. Head over to the Scaleway
console for Cockpit and click on the `"Open your dashboards"` button. This will open Grafana for you. Enter the username
and password we just got and press on "Log in". You should now be in the Grafana dashboard.

Here is an example dashboard I made for my Minecraft server:

```json
{
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": {
          "type": "datasource",
          "uid": "grafana"
        },
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "target": {
          "limit": 100,
          "matchAny": false,
          "tags": [],
          "type": "dashboard"
        },
        "type": "dashboard"
      }
    ]
  },
  "editable": true,
  "fiscalYearStartMonth": 0,
  "graphTooltip": 0,
  "id": 19,
  "links": [],
  "liveNow": false,
  "panels": [
    {
      "datasource": {
        "type": "prometheus",
        "uid": "ag1UWWfVk"
      },
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "palette-classic"
          },
          "custom": {
            "axisCenteredZero": false,
            "axisColorMode": "text",
            "axisLabel": "",
            "axisPlacement": "auto",
            "barAlignment": 0,
            "drawStyle": "line",
            "fillOpacity": 0,
            "gradientMode": "none",
            "hideFrom": {
              "legend": false,
              "tooltip": false,
              "viz": false
            },
            "lineInterpolation": "linear",
            "lineWidth": 1,
            "pointSize": 5,
            "scaleDistribution": {
              "type": "linear"
            },
            "showPoints": "auto",
            "spanNulls": false,
            "stacking": {
              "group": "A",
              "mode": "none"
            },
            "thresholdsStyle": {
              "mode": "off"
            }
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "red",
                "value": 80
              }
            ]
          }
        },
        "overrides": []
      },
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 0,
        "y": 0
      },
      "id": 10,
      "options": {
        "legend": {
          "calcs": [],
          "displayMode": "list",
          "placement": "bottom",
          "showLegend": true
        },
        "tooltip": {
          "mode": "single",
          "sort": "none"
        }
      },
      "targets": [
        {
          "datasource": {
            "type": "prometheus",
            "uid": "ag1UWWfVk"
          },
          "editorMode": "code",
          "expr": "sum(minecraft_movement_meters_total{player=\"$player\"}) by (means)",
          "legendFormat": "__auto",
          "range": true,
          "refId": "A"
        }
      ],
      "title": "Movement",
      "type": "timeseries"
    },
    {
      "datasource": {
        "type": "prometheus",
        "uid": "ag1UWWfVk"
      },
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "mappings": [
            {
              "options": {
                "match": "null",
                "result": {
                  "text": "N/A"
                }
              },
              "type": "special"
            }
          ],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "red",
                "value": 80
              }
            ]
          },
          "unit": "none"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 8,
        "w": 4,
        "x": 12,
        "y": 0
      },
      "id": 5,
      "links": [],
      "maxDataPoints": 100,
      "options": {
        "colorMode": "none",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "horizontal",
        "reduceOptions": {
          "calcs": [
            "mean"
          ],
          "fields": "",
          "values": false
        },
        "textMode": "auto"
      },
      "pluginVersion": "9.3.1",
      "targets": [
        {
          "datasource": {
            "uid": "${DS_PROMETHEUS}"
          },
          "editorMode": "code",
          "expr": "sum(minecraft_deaths_total{player=\"$player\"})",
          "instant": true,
          "legendFormat": "",
          "refId": "A"
        }
      ],
      "title": "Deaths",
      "type": "stat"
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": {
        "type": "prometheus",
        "uid": "ag1UWWfVk"
      },
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "gridPos": {
        "h": 8,
        "w": 7,
        "x": 16,
        "y": 0
      },
      "hiddenSeries": false,
      "id": 4,
      "legend": {
        "avg": false,
        "current": false,
        "max": false,
        "min": false,
        "show": true,
        "total": false,
        "values": false
      },
      "lines": true,
      "linewidth": 1,
      "nullPointMode": "null",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "9.3.1",
      "pointradius": 2,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "datasource": {
            "uid": "${DS_PROMETHEUS}"
          },
          "editorMode": "code",
          "expr": "sum(minecraft_item_actions_total{player=\"$player\", action=\"picked_up\"}) by (entity)",
          "legendFormat": "{{block}}",
          "range": true,
          "refId": "A"
        }
      ],
      "thresholds": [],
      "timeRegions": [],
      "title": "Blocks collected",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "individual"
      },
      "type": "graph",
      "xaxis": {
        "mode": "time",
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "format": "short",
          "logBase": 1,
          "show": true
        },
        {
          "format": "short",
          "logBase": 1,
          "show": true
        }
      ],
      "yaxis": {
        "align": false
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": {
        "type": "prometheus",
        "uid": "ag1UWWfVk"
      },
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "gridPos": {
        "h": 7,
        "w": 23,
        "x": 0,
        "y": 8
      },
      "hiddenSeries": false,
      "id": 2,
      "legend": {
        "avg": false,
        "current": false,
        "max": false,
        "min": false,
        "show": true,
        "total": false,
        "values": false
      },
      "lines": true,
      "linewidth": 1,
      "nullPointMode": "null",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "9.3.1",
      "pointradius": 2,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "datasource": {
            "uid": "${DS_PROMETHEUS}"
          },
          "editorMode": "code",
          "expr": "sum(minecraft_blocks_mined_total{player=\"$player\"}) by (block)",
          "legendFormat": "{{block}}",
          "range": true,
          "refId": "A"
        }
      ],
      "thresholds": [],
      "timeRegions": [],
      "title": "Blocks mined",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "individual"
      },
      "type": "graph",
      "xaxis": {
        "mode": "time",
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "format": "short",
          "logBase": 1,
          "show": true
        },
        {
          "format": "short",
          "logBase": 1,
          "show": true
        }
      ],
      "yaxis": {
        "align": false
      }
    }
  ],
  "refresh": false,
  "schemaVersion": 37,
  "style": "dark",
  "tags": [],
  "templating": {
    "list": [
      {
        "current": {
          "selected": false,
          "text": "_diri",
          "value": "_diri"
        },
        "definition": "minecraft_player_online_total",
        "hide": 0,
        "includeAll": false,
        "multi": false,
        "name": "player",
        "options": [],
        "query": {
          "query": "minecraft_player_online_total",
          "refId": "StandardVariableQuery"
        },
        "refresh": 1,
        "regex": "/player=\"(?<text>[^\"]+)/",
        "skipUrlSync": false,
        "sort": 0,
        "tagValuesQuery": "",
        "tagsQuery": "",
        "type": "query",
        "useTags": false
      }
    ]
  },
  "time": {
    "from": "now-6h",
    "to": "now"
  },
  "timepicker": {
    "refresh_intervals": [
      "5s",
      "10s",
      "30s",
      "1m",
      "5m",
      "15m",
      "30m",
      "1h",
      "2h",
      "1d"
    ]
  },
  "timezone": "",
  "title": "minecraft Player stats",
  "uid": "gAy914AZk",
  "version": 1,
  "weekStart": ""
}
```

## Housekeeping

When you are done with recreating this blog post, you can delete the resources you created. You can do this by running
the following commands:

```bash
pulumi destroy -y -f
```

## Conclusion

The new services from Scale way are really great and are very easy to integrate into existing tools
like `kube-prometheus-stack`, `promtail` and `external-secrets`.

I hope you enjoyed this blog post and learned something new. If you have any questions or comments, feel free to reach
out to me on Twitter or via email.
