package main

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		doks, err := pulumi.NewStackReference(ctx, "dirien/00-infrastructure/dev", nil)
		if err != nil {
			return err
		}

		provider, err := kubernetes.NewProvider(ctx, "kubernetes", &kubernetes.ProviderArgs{
			Kubeconfig: doks.GetStringOutput(pulumi.String("kubeConfig")),
		})
		if err != nil {
			return err
		}
		_, err = helm.NewRelease(ctx, "httpbin", &helm.ReleaseArgs{
			Name:    pulumi.String("httpbin"),
			Chart:   pulumi.String("httpbin"),
			Version: pulumi.String("0.1.1"),
			RepositoryOpts: helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://matheusfm.dev/charts"),
			},
			Values: pulumi.Map{},
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}
		return nil
	})
}
