package main

import (
	"github.com/pulumi/pulumi-github/sdk/v4/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"os"

	"github.com/pulumi/pulumi-aws-native/sdk/go/aws/amplify"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		repository, err := github.LookupRepository(ctx, &github.LookupRepositoryArgs{
			FullName: pulumi.StringRef("dirien/hello-jamstack"),
		})
		if err != nil {
			return err
		}

		amplifyYaml, err := os.ReadFile("amplify.yaml")
		if err != nil {
			return err
		}
		app, err := amplify.NewApp(ctx, "hello-amplify-hackathon", &amplify.AppArgs{
			Name:                     pulumi.String("hello-amplify-hackathon"),
			Repository:               pulumi.String(repository.HtmlUrl),
			AccessToken:              config.GetSecret(ctx, "github:token"),
			BuildSpec:                pulumi.String(amplifyYaml),
			EnableBranchAutoDeletion: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		mainBranch, err := amplify.NewBranch(ctx, "default-branch", &amplify.BranchArgs{
			AppId:                    app.AppId,
			BranchName:               pulumi.String(repository.DefaultBranch),
			Stage:                    amplify.BranchStageProduction,
			EnablePullRequestPreview: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		ctx.Export("default-domain", app.DefaultDomain)
		ctx.Export("branch-url", pulumi.Sprintf("https://%s.%s", mainBranch.BranchName, app.DefaultDomain))
		return nil
	})
}
