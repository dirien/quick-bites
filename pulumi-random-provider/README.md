# Quick Bites of Pulumi: Random Provider

This is a very quick overview of the Random Provider from Pulumi, similar to the Quick Bites of Cloud Engineering videos
from Laura Santamaria (@nimbinatus)

## Random Provider

The Random Provider is on of many providers that come with Pulumi. Currently, it implements the following functions:

- RandomId
- RandomInteger
- RandomPassword
- RandomPet
- RandomShuffle
- RandomString
- RandomUuid

I will show you three of the in more detail.

### RandomPet

Do you want also this cool names, like when you start a docker container? Or don't want to set a name for your k8s
cluster or virtual machines? Then this function is definitely for you.

Just call the function `NewRandomPet` and you will get a random pet name! Simplez!

Here is a golang example:
```go
serverName, err := random.NewRandomPet(ctx, "pet-server-name", &random.RandomPetArgs{})
ctx.Export("serverName", serverName.ID())
```

And the output is:

```bash
serverName: "correct-turkey"
```

### RandomShuffle

I love RandomShuffle. It is a function that is great to do random permutation of a list of strings. Set the `ResultCount`
for the number of results to return. With the argument `Seed` you can set the seed for the random number generator.

```go
vmSize, err := random.NewRandomShuffle(ctx, "vm-size", &random.RandomShuffleArgs{
    Inputs: pulumi.StringArray{
        pulumi.String("Standard_DS12-1_v2"),
        pulumi.String("Standard_DS3_v2_Promo"),
        pulumi.String("Standard_D4_v3"),
        pulumi.String("Standard_E8d_v4"),
    },
    ResultCount: pulumi.Int(2),
})

ctx.Export("vmSize", vmSize.Results)
```

And the output is:

```bash
vmSize: [
    [0]: "Standard_DS12-1_v2"
    [1]: "Standard_D4_v3"
]
```

### RandomUuid

Last but not least, the RandomUuid function. The classic UUID generator for literally anything and everything.

```go
uuid, err := random.NewRandomUuid(ctx, "random-uuid", nil)

ctx.Export("uuid", uuid.Result)
```

voilà, the output is:

```bash
uuid: "cc222078-6b73-b35a-970c-974e331a9ede"
```

Try this provider for yourself, in your next Pulumi project!

You can find more information about the [Random Provider](https://www.pulumi.com/registry/packages/random/).