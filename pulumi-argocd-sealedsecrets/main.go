package main

import (
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-tls/sdk/v4/go/tls"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		kubernetesCluster, err := digitalocean.NewKubernetesCluster(ctx, "pulumi-argocd-sealedsecrets", &digitalocean.KubernetesClusterArgs{
			Name:    pulumi.String("pulumi-argocd-sealedsecrets"),
			Region:  pulumi.String("fra1"),
			Version: pulumi.String("1.25.4-do.0"),
			NodePool: &digitalocean.KubernetesClusterNodePoolArgs{
				Name:      pulumi.String("pulumi-argocd-sealedsecrets"),
				NodeCount: pulumi.Int(1),
				Size:      pulumi.String("s-4vcpu-8gb"),
			},
		})
		if err != nil {
			return err
		}

		key, err := tls.NewPrivateKey(ctx, "pulumi-argocd-sealedsecrets", &tls.PrivateKeyArgs{
			Algorithm: pulumi.String("RSA"),
			RsaBits:   pulumi.Int(4096),
		})
		if err != nil {
			return err
		}
		selfSignedCert, err := tls.NewSelfSignedCert(ctx, "pulumi-argocd-sealedsecrets", &tls.SelfSignedCertArgs{
			Subject: &tls.SelfSignedCertSubjectArgs{
				CommonName:   pulumi.String("sealed-secret/"),
				Organization: pulumi.String("sealed-secret"),
			},
			PrivateKeyPem: key.PrivateKeyPem,
			AllowedUses: pulumi.StringArray{
				pulumi.String("cert_signing"),
			},
			ValidityPeriodHours: pulumi.Int(365 * 24),
		})
		if err != nil {
			return err
		}

		provider, err := kubernetes.NewProvider(ctx, "pulumi-argocd-sealedsecrets", &kubernetes.ProviderArgs{
			Kubeconfig:            kubernetesCluster.KubeConfigs.ToKubernetesClusterKubeConfigArrayOutput().Index(pulumi.Int(0)).RawConfig(),
			EnableServerSideApply: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		namespace, err := v1.NewNamespace(ctx, "sealed-secrets", &v1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("sealed-secrets"),
			},
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}
		secret, err := v1.NewSecret(ctx, "pulumi-argocd-sealedsecrets", &v1.SecretArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("sealed-secret"),
				Namespace: namespace.Metadata.Namespace(),
				Labels: pulumi.StringMap{
					"sealedsecrets.bitnami.com/sealed-secrets-key": pulumi.String("active"),
				},
			},
			Type: pulumi.String("kubernetes.io/tls"),
			StringData: pulumi.StringMap{
				"tls.crt": selfSignedCert.CertPem,
				"tls.key": key.PrivateKeyPem,
			},
		}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{namespace}))
		if err != nil {
			return err
		}
		sealedSecretsRelease, err := helmv3.NewRelease(ctx, "sealed-secrets", &helmv3.ReleaseArgs{
			Name:  pulumi.String("sealed-secrets"),
			Chart: pulumi.String("sealed-secrets"),
			RepositoryOpts: &helmv3.RepositoryOptsArgs{
				Repo: pulumi.String("https://charts.bitnami.com/bitnami"),
			},
			SkipAwait: pulumi.Bool(true),
			Namespace: namespace.Metadata.Namespace(),
			Version:   pulumi.String("1.2.1"),
		}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{namespace, secret}), pulumi.IgnoreChanges([]string{"values", "version"}))
		if err != nil {
			return err
		}

		argoCD, err := helmv3.NewRelease(ctx, "argocd", &helmv3.ReleaseArgs{
			Name:  pulumi.String("argocd"),
			Chart: pulumi.String("argo-cd"),
			RepositoryOpts: &helmv3.RepositoryOptsArgs{
				Repo: pulumi.String("https://argoproj.github.io/argo-helm"),
			},
			SkipAwait:       pulumi.Bool(true),
			Namespace:       pulumi.String("argocd"),
			Version:         pulumi.String("5.16.9"),
			CreateNamespace: pulumi.Bool(true),
			Values: pulumi.Map{
				"server": pulumi.Map{
					"extraArgs": pulumi.Array{
						pulumi.String("--insecure"),
					},
				},
			},
		}, pulumi.Provider(provider), pulumi.IgnoreChanges([]string{"values", "version"}))

		_, err = apiextensions.NewCustomResource(ctx, "sealed-secrets-application", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("argoproj.io/v1alpha1"),
			Kind:       pulumi.String("Application"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("sealed-secrets"),
				Namespace: pulumi.String("argocd"),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"destination": pulumi.Map{
						"namespace": sealedSecretsRelease.Namespace,
						"name":      pulumi.String("in-cluster"),
					},
					"project": pulumi.String("default"),
					"source": pulumi.Map{
						"repoURL":        sealedSecretsRelease.RepositoryOpts.Repo(),
						"targetRevision": sealedSecretsRelease.Version,
						"chart":          sealedSecretsRelease.Chart,
					},
					"syncPolicy": pulumi.Map{
						"syncOptions": pulumi.Array{
							pulumi.String("ServerSideApply=true"),
						},
						"automated": pulumi.Map{
							"prune":    pulumi.Bool(true),
							"selfHeal": pulumi.Bool(true),
						},
					},
				},
			},
		}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{argoCD}))
		if err != nil {
			return err
		}
		_, err = apiextensions.NewCustomResource(ctx, "argocd-application", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("argoproj.io/v1alpha1"),
			Kind:       pulumi.String("Application"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("argocd"),
				Namespace: argoCD.Namespace,
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"destination": pulumi.Map{
						"namespace": argoCD.Namespace,
						"name":      pulumi.String("in-cluster"),
					},
					"project": pulumi.String("default"),
					"source": pulumi.Map{
						"repoURL":        argoCD.RepositoryOpts.Repo(),
						"targetRevision": argoCD.Version,
						"chart":          argoCD.Chart,
						"helm": pulumi.Map{
							"values": pulumi.String(`server:
 extraArgs:
 - --insecure`),
						},
					},
					"syncPolicy": pulumi.Map{
						"syncOptions": pulumi.Array{
							pulumi.String("ServerSideApply=true"),
						},
						"automated": pulumi.Map{
							"prune":    pulumi.Bool(true),
							"selfHeal": pulumi.Bool(true),
						},
					},
				},
			},
		}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{argoCD}))
		if err != nil {
			return err
		}

		_, err = apiextensions.NewCustomResource(ctx, "demo-application", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("argoproj.io/v1alpha1"),
			Kind:       pulumi.String("Application"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("demo-application"),
				Namespace: pulumi.String("argocd"),
			},
			OtherFields: kubernetes.UntypedArgs{
				"spec": pulumi.Map{
					"destination": pulumi.Map{
						"namespace": pulumi.String("default"),
						"name":      pulumi.String("in-cluster"),
					},
					"project": pulumi.String("default"),
					"source": pulumi.Map{
						"repoURL":        pulumi.String("https://github.com/dirien/very-very-simple-k8s-deployment"),
						"targetRevision": pulumi.String("HEAD"),
						"path":           pulumi.String("."),
					},
					"syncPolicy": pulumi.Map{
						"syncOptions": pulumi.Array{
							pulumi.String("ServerSideApply=true"),
						},
						"automated": pulumi.Map{
							"prune":    pulumi.Bool(true),
							"selfHeal": pulumi.Bool(true),
						},
					},
				},
			},
		}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{argoCD}))
		if err != nil {
			return err
		}

		ctx.Export("clusterName", kubernetesCluster.Name)
		ctx.Export("certPem", selfSignedCert.CertPem)
		ctx.Export("kubeconfig", pulumi.ToSecret(kubernetesCluster.KubeConfigs.ToKubernetesClusterKubeConfigArrayOutput().Index(pulumi.Int(0)).RawConfig()))
		return nil
	})
}
