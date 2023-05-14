package main

import (
	"encoding/base64"
	"fmt"
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type NodePoolArgs struct {
	Name      string
	Size      string
	NodeCount int
	Label     string
}

type ClusterArgs struct {
	Name     string
	Region   string
	Version  string
	NodePool *[]NodePoolArgs
}

const (
	DefaultNodePoolName = "default"
	DefaultNodePoolSize = "s-2vcpu-4gb"
)

func createDOKSCLuster(ctx *pulumi.Context, args *ClusterArgs) (pulumi.StringOutput, error) {
	// Create a new DOKS cluster
	cluster, err := digitalocean.NewKubernetesCluster(ctx, args.Name, &digitalocean.KubernetesClusterArgs{
		Region:  pulumi.String(args.Region),
		Version: pulumi.String(args.Version),
		NodePool: &digitalocean.KubernetesClusterNodePoolArgs{
			Name:      pulumi.String(DefaultNodePoolName),
			Size:      pulumi.String(DefaultNodePoolSize),
			NodeCount: pulumi.Int(1),
			AutoScale: pulumi.Bool(false),
		},
	})

	if err != nil {
		return pulumi.StringOutput{}, err
	}
	if args.NodePool != nil {
		for _, nodePool := range *args.NodePool {
			_, err := digitalocean.NewKubernetesNodePool(ctx, nodePool.Name, &digitalocean.KubernetesNodePoolArgs{
				ClusterId: cluster.ID(),
				Name:      pulumi.String(nodePool.Name),
				Size:      pulumi.String(nodePool.Size),
				NodeCount: pulumi.Int(nodePool.NodeCount),
				Labels:    pulumi.StringMap{"env": pulumi.String(nodePool.Label)},
			})
			if err != nil {
				return pulumi.StringOutput{}, err
			}
		}
	}

	output, _ := cluster.KubeConfigs.ApplyT(func(kcs []digitalocean.KubernetesClusterKubeConfig) (string, error) {
		if len(kcs) == 0 {
			return "", nil
		}
		return *kcs[0].RawConfig, nil
	}).(pulumi.StringOutput)

	return output, nil
}

func deployProd(ctx *pulumi.Context, cluster pulumi.StringOutput) error {
	productionKubernetesProvider, err := kubernetes.NewProvider(ctx, "production-k8s", &kubernetes.ProviderArgs{
		Kubeconfig:            cluster,
		EnableServerSideApply: pulumi.Bool(true),
	})
	if err != nil {
		return err
	}

	dopplerOperatorHelm, err := helm.NewRelease(ctx, "doppler-operator", &helm.ReleaseArgs{
		Name:            pulumi.String("doppler-operator"),
		Chart:           pulumi.String("doppler-kubernetes-operator"),
		Namespace:       pulumi.String("doppler-operator-system"),
		CreateNamespace: pulumi.Bool(false),
		RepositoryOpts: &helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://helm.doppler.com"),
		},
		Version: pulumi.String("1.2.5"),
	}, pulumi.Provider(productionKubernetesProvider))
	if err != nil {
		return err
	}

	secret, err := v1.NewSecret(ctx, "doppler-token-secret", &v1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("doppler-token-secret"),
			Namespace: pulumi.String("doppler-operator-system"),
		},
		Type: pulumi.String("Opaque"),
		Data: pulumi.StringMap{
			"serviceToken": config.GetSecret(ctx, "doks-production").ApplyT(func(s string) (string, error) {
				return base64.StdEncoding.EncodeToString([]byte(s)), nil
			}).(pulumi.StringOutput),
		},
	}, pulumi.Provider(productionKubernetesProvider), pulumi.DependsOn([]pulumi.Resource{dopplerOperatorHelm}))
	if err != nil {
		return err
	}

	_, err = apiextensions.NewCustomResource(ctx, "dopplersecrets", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("secrets.doppler.com/v1alpha1"),
		Kind:       pulumi.String("DopplerSecret"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("doppler-token-secret"),
			Namespace: secret.Metadata.Namespace().Elem(),
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"tokenSecret": pulumi.Map{
					"name":      secret.Metadata.Name().Elem(),
					"namespace": secret.Metadata.Namespace().Elem(),
				},
				"managedSecret": pulumi.Map{
					"name":      pulumi.String("doppler-secret"),
					"namespace": pulumi.String("default"),
				},
			},
		},
	}, pulumi.Provider(productionKubernetesProvider), pulumi.DependsOn([]pulumi.Resource{dopplerOperatorHelm}))
	if err != nil {
		return err
	}
	_, err = appsv1.NewDeployment(ctx, "hello-server", &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("hello-server"),
			Annotations: pulumi.StringMap{
				"secrets.doppler.com/reload": pulumi.String("true"),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app": pulumi.String("hello-server"),
				},
			},
			Template: &v1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{
						"app": pulumi.String("hello-server"),
					},
				},
				Spec: &v1.PodSpecArgs{
					Containers: v1.ContainerArray{
						v1.ContainerArgs{
							Name:  pulumi.String("hello-server"),
							Image: pulumi.String("ghcr.io/dirien/hello-server/hello-server:latest"),
							Ports: v1.ContainerPortArray{
								v1.ContainerPortArgs{
									ContainerPort: pulumi.Int(8080),
								},
							},
							EnvFrom: v1.EnvFromSourceArray{
								v1.EnvFromSourceArgs{
									SecretRef: &v1.SecretEnvSourceArgs{
										Name: pulumi.String("doppler-secret"),
									},
								},
							},
						},
					},
				},
			},
		},
	}, pulumi.Provider(productionKubernetesProvider))
	if err != nil {
		return err
	}
	return nil
}

func deployPreProd(ctx *pulumi.Context, cluster pulumi.StringOutput, stage string) error {
	preProdKubernetesProvider, err := kubernetes.NewProvider(ctx, fmt.Sprintf("%s-k8s", stage), &kubernetes.ProviderArgs{
		Kubeconfig:            cluster,
		EnableServerSideApply: pulumi.Bool(true),
	})
	if err != nil {
		return err
	}
	_, err = helm.NewRelease(ctx, fmt.Sprintf("%s-vcluster", stage), &helm.ReleaseArgs{
		Name:  pulumi.String(fmt.Sprintf("%s-vcluster", stage)),
		Chart: pulumi.String("vcluster"),
		RepositoryOpts: &helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://charts.loft.sh"),
		},
		Values: pulumi.Map{
			"sync": pulumi.Map{
				"nodes": pulumi.Map{
					"enabled":         pulumi.Bool(true),
					"enableScheduler": pulumi.Bool(true),
					"nodeSelector":    pulumi.Sprintf("env=%s", stage),
				},
			},
			"nodeSelector": pulumi.Map{
				"env": pulumi.String(stage),
			},
			"init": pulumi.Map{
				"helm": pulumi.Array{
					pulumi.Map{
						"chart": pulumi.Map{
							"name":    pulumi.String("doppler-kubernetes-operator"),
							"repo":    pulumi.String("https://helm.doppler.com"),
							"version": pulumi.String("1.2.5"),
						},
						"release": pulumi.Map{
							"name":      pulumi.String("doppler-operator"),
							"namespace": pulumi.String("doppler-operator-system"),
						},
					},
				},
			},
		},
	}, pulumi.Provider(preProdKubernetesProvider))
	if err != nil {
		return err
	}
	return nil
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create the DOKS cluster for development and staging
		preProd, err := createDOKSCLuster(ctx, &ClusterArgs{
			Name:    "pre-prod-cluster",
			Region:  "fra1",
			Version: "1.26.3-do.0",
			NodePool: &[]NodePoolArgs{
				{
					Name:      "development",
					Size:      "s-2vcpu-4gb",
					NodeCount: 1,
					Label:     "development",
				},
				{
					Name:      "staging",
					Size:      "s-2vcpu-4gb",
					NodeCount: 1,
					Label:     "staging",
				},
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("preProdKubeConfig", pulumi.ToSecret(preProd))

		// Deploy the vcluster to development and staging
		err = deployPreProd(ctx, preProd, "development")
		if err != nil {
			return err
		}
		err = deployPreProd(ctx, preProd, "staging")
		if err != nil {
			return err
		}

		// Create the DOKS cluster for production
		prod, err := createDOKSCLuster(ctx, &ClusterArgs{
			Name:    "prod-cluster",
			Region:  "fra1",
			Version: "1.25.8-do.0",
		})
		if err != nil {
			return err
		}
		ctx.Export("prodKubeConfig", pulumi.ToSecret(prod))

		err = deployProd(ctx, prod)
		if err != nil {
			return err
		}
		return nil
	})
}
