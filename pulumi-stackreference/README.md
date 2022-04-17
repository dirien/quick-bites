# Quick Bites of Pulumi: Stack References

## TL;DR Where code?

-> Here you go https://github.com/dirien/quick-bites

This is a very quick overview of the Random Provider from Pulumi, similar to
the `Quick Bites of Cloud Engineering` [videos](https://www.youtube.com/playlist?list=PLyy8Vx2ZoWlohOiedbaQqT5xYRkcDsm10) from `Laura Santamaria` (@nimbinatus)

## What are Stack References?

Stack References are a very interesting concept in Pulumi. They provide a way to access the output of one stack, mostly written with the `export` keyword or methods from another stack.

Let us check the quick example, in the folder `00-infrastructure`, I am going to create a DigitalOcean's managed Kubernetes Service, or DOKS. The actual `Pulumi` program is written in TypeScript, to demonstrate that is doesn't matter which language you are using to reference the stack output

Swiftly done via following commands:

```bash
pulumi new digitalocean-typescript
pulumi config set digitalocean:token $DIGITALOCEAN_TOKEN --secret
pulumi up --yes --skip-preview
```

And the TypeScript code is:

```typescript
...
let doks = new digitalocean.KubernetesCluster("k8s-cluster", {
  name: "quick-bites-cluster",
  ...
});
...
export const kubeConfig = pulumi.secret(doks.kubeConfigs[0].rawConfig)
```

It may take some time to come up with the deployment but at the end you should see something like this:

```bash
Outputs:
  + kubeConfig: [secret]

Resources:
    + 1 created
```

Now we create the second `Pulumi` program in the folder `01-kubernetes`. This one will be written in Go and deploy a simple `httpbin` via a Helm chart.

To reference the stack names must be fully qualified, including the organization, project, and stack name components, in the format `<org-name>/<project>/<stack>`. If you own an individual account you have to change the `<org-name>` part with your account name.

For me, it would be: `dirien/00-infrastructure/dev`

Now lets reference it! To create the next `Pulumi` program in the `01-kubernetes` folder just type following commands:

```bash
pulumi new kubernetes-go
pulumi up --yes --skip-preview
```

The important part of the code is, where we create StackReference object. The constructor takes as input a string in the form of  `<org-name>/<project>/<stack>`.

```go
...
doks, err := pulumi.NewStackReference(ctx, "dirien/00-infrastructure/dev", nil)
if err != nil {
    return err
}

provider, err := kubernetes.NewProvider(ctx, "kubernetes", &kubernetes.ProviderArgs{
    Kubeconfig: doks.GetStringOutput(pulumi.String("kubeConfig")),
})
...
```

To test that it works, let us export the `kubeConfig` into a `kubeconfig.yaml` file:

```bash
cd 00-infrastructure
pulumi stack output kubeconfig --show-secrets  > kubeconfig.yaml

export KUBECONFIG=kubeconfig.yaml

kubectl get svc
NAME         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
httpbin      ClusterIP   10.245.179.59   <none>        80/TCP    43s
kubernetes   ClusterIP   10.245.0.1      <none>        443/TCP   10m

kubectl port-forward svc/httpbin 8080:80
Forwarding from 127.0.0.1:8080 -> 8080
Forwarding from [::1]:8080 -> 8080
Handling connection for 8080
Handling connection for 8080
```

![image.png](https://cdn.hashnode.com/res/hashnode/image/upload/v1650233795205/xWudUDZjZ.png)


## Wrap up

What are the advantages of using Stack References?

- Now you can cut your infrastructure code into separate pulumi programs, and define an opinionated way of which values you want to expose for other stacks to consume.

- You created a boundary, where you can work on your own lifecycle needs. Any updates to the infrastructure code will be isolated to the other Pulumi programs.

- This self-contained Pulumi programs can have different stacks (dev/staging/prod) you can provide to others.

- Different Pulumi programs may be ownd from different people inside on team or whole teams inside an organization.

- Application developers, don't need to know about the detail infrastructure implementation. What they need are only the endpoints. For example the kubeconfig or the database connection string.

## Clean up

Don't forget to destroy both `Pulumi` programs with the following command:

```bash
pulumi destroy --yes
```
