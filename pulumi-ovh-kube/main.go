package main

import (
	k8s "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"github.com/scraly/pulumi-ovh/sdk/go/ovh/cloudproject"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		serviceName := config.Require(ctx, "serviceName")

		mykube, err := cloudproject.NewKube(ctx, "mykube", &cloudproject.KubeArgs{
			ServiceName: pulumi.String(serviceName),
			Name:        pulumi.String("myFirstOVHKubernetesCluster"),
			Region:      pulumi.String("SBG5"),
		})
		if err != nil {
			return err
		}
		ctx.Export("kubeconfig", mykube.Kubeconfig)

		nodePool, err := cloudproject.NewKubeNodePool(ctx, "mykubenodepool", &cloudproject.KubeNodePoolArgs{
			ServiceName:   pulumi.String(serviceName),
			KubeId:        mykube.ID(),
			Name:          pulumi.String("default"),
			Autoscale:     pulumi.BoolPtr(true),
			DesiredNodes:  pulumi.Int(1),
			MinNodes:      pulumi.Int(1),
			MaxNodes:      pulumi.Int(2),
			FlavorName:    pulumi.String("d2-8"),
			MonthlyBilled: pulumi.BoolPtr(false),
		})
		if err != nil {
			return err
		}
		ctx.Export("nodePoolId", nodePool.ID())
		k8sProvider, err := k8s.NewProvider(ctx, "k8s", &k8s.ProviderArgs{
			Kubeconfig: mykube.Kubeconfig,
		}, pulumi.DependsOn([]pulumi.Resource{nodePool, mykube}))
		if err != nil {
			return err
		}
		_, err = helm.NewRelease(ctx, "minecraft", &helm.ReleaseArgs{
			Chart:   pulumi.String("minecraft"),
			Version: pulumi.String("4.9.3"),
			RepositoryOpts: &helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://itzg.github.io/minecraft-server-charts"),
			},
			CreateNamespace: pulumi.Bool(true),
			Namespace:       pulumi.String("minecraft"),
			Values: pulumi.Map{
				"minecraftServer": pulumi.Map{
					"eula":        pulumi.Bool(true),
					"motd":        pulumi.String("OVH Strasbourg - Minecraft Server"),
					"serviceType": pulumi.String("LoadBalancer"),
				},
			},
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}
		return nil
	})
}
