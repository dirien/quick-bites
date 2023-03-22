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

		mcNamespace, err := v1.NewNamespace(ctx, "minecraft", &v1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("minecraft"),
			},
		}, pulumi.Provider(kubernetesProvider))
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

		mcExternalSecret, err := apiextensions.NewCustomResource(ctx, "external-secrets", &apiextensions.CustomResourceArgs{
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
		if err != nil {
			return err
		}

		mcHelmChart, err := helm.NewRelease(ctx, "minecraft", &helm.ReleaseArgs{
			Chart:   pulumi.String("minecraft"),
			Version: pulumi.String("4.6.0"),
			RepositoryOpts: &helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://itzg.github.io/minecraft-server-charts"),
			},
			Namespace: mcNamespace.Metadata.Name(),
			SkipAwait: pulumi.Bool(true),
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
		}, pulumi.Provider(kubernetesProvider), pulumi.DependsOn([]pulumi.Resource{mcNamespace, kubePrometheusStack, mcExternalSecret, mcSecretStore, externalSecrets}))
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
							"app": pulumi.All(mcHelmChart.Name, mcHelmChart.Namespace).ApplyT(func(args []interface{}) string {
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
		}, pulumi.Provider(kubernetesProvider), pulumi.DependsOn([]pulumi.Resource{mcNamespace, mcHelmChart, kubePrometheusStack}))

		ctx.Export("kubeconfig", pulumi.ToSecret(k8sCluster.Kubeconfigs.Index(pulumi.Int(0)).ConfigFile()))
		return nil
	})
}
