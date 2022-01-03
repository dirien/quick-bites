package main

import (
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		serverName, err := random.NewRandomPet(ctx, "pet-server-name", &random.RandomPetArgs{})
		if err != nil {
			return err
		}

		ctx.Export("serverName", serverName.ID())

		vmSize, err := random.NewRandomShuffle(ctx, "vm-size", &random.RandomShuffleArgs{
			Inputs: pulumi.StringArray{
				pulumi.String("Standard_DS12-1_v2"),
				pulumi.String("Standard_DS3_v2_Promo"),
				pulumi.String("Standard_D4_v3"),
				pulumi.String("Standard_E8d_v4"),
			},
			ResultCount: pulumi.Int(2),
		})
		if err != nil {
			return err
		}

		ctx.Export("vmSize", vmSize.Results)

		uuid, err := random.NewRandomUuid(ctx, "random-uuid", nil)
		if err != nil {
			return err
		}
		ctx.Export("uuid", uuid.Result)

		return nil
	})
}
