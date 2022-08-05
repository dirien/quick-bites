# How to enable the Cilium Hubble UI in a Civo k3s cluster

In this article I want to show, how to enable the `Hubble UI` in a [Cilium](https://docs.cilium.io/en/stable/intro/) powered `Civo` k3s cluster.

## What is Hubble?

Hubble is a fully distributed networking and security observability platform. It is built on top of `Cilium` and eBPF to enable deep visibility into the communication and behavior of services as well as the networking infrastructure.

By building on top of `Cilium`, Hubble can leverage eBPF for visibility. By relying on eBPF, all visibility is programmable and allows for a dynamic approach that minimizes overhead while providing deep and detailed visibility as required by users.

## Creating the cluster via `Pulumi`

I created my cluster using `Pulumi` and the `civo-provider`. I will not dive into the details of `Pulumi` in this article, but Kunal Kushwaha made a great video about `Pulumi` and the `Civo` provider.

%[https://www.youtube.com/watch?v=bTyOr4kiJp8]

Or check the [official](https://www.pulumi.com/docs/get-started/) documentation.

`Pulumi` supports multiple programming languages, I decided to use `yaml` language. With the commands below, we boostrap our `Pulumi` program using the `yaml` template.

```bash
mkdir civo-hubble && cd civo-hubbble
pulumi new yaml --force
```

For our demo infrastructure, we just need the bare minimum thats why I create only a `Firewall` and the `KubernetesCluster` resource.

```yaml
name: pulumi-civo-cilium-hubble
runtime: yaml
description: Enable Hubble UI on a Civo cluster

variables:
  region: FRA1

resources:
  civo-firewall:
    type: civo:Firewall
    properties:
      name: MyCivoFirewall
      region: ${region}

  civo-k3s-cluster:
    type: civo:KubernetesCluster
    properties:
      name: MyCivoCluster
      region: ${region}
      firewallId: ${civo-firewall.id}
      cni: cilium
      pools:
        nodeCount: 2
        size: g4s.kube.medium

outputs:
  kubeconfig:
    Fn::Secret:
      ${civo-k3s-cluster.kubeconfig}
```

Deploy the `Pulumi` program with following command (and don't forget to set the Civo API Token as environment variable):

```bash
export CIVO_TOKEN=xxxx
pulumi preview
pulumi up -y -f
```

Next we need the kubeconfig to enable the `Hubble UI`.  We can get the `kubeconfig` via a command from our `Pulumi` deployment.

```bash
pulumi stack output kubeconfig --show-secrets > kubeconfig.yaml
```

## Enable Hubble UI

There are two ways to enable the Hubble UI:

- With the `cilium`-cli
- or with the `Helm` chart

Civo itselft, installs the `Cilium` CNI via the `Cilium` helm chart, we can verify this with following command:

```bash
kubectl get secrets -n kube-system 
...
cilium-operator-token-8fdv2                                            kubernetes.io/service-account-token   3      19m
sh.helm.release.v1.cilium.v1                                           helm.sh/release.v1                    1      19m
k3s-mycivocluster-f23a-c594fc-node-pool-59d1-3nd2c.node-password.k3s   Opaque                                1      19m
...
```

The `Cilium` cli method will not work as we get an error message:

```bash
cilium hubble enable

Error: Unable to enable Hubble: unable to retrieve helm values secret kube-system/cilium-cli-helm-values: secrets "cilium-cli-helm-values" not found
```

That means, to enable the Hubble UI we need to use the Helm chart way and upgrade the existing `helm release` by calling the `helm upgrade` function.

> Attention: We use the `reuse-values` flag to avoid delete the values from Civo

```bash
helm upgrade cilium cilium/cilium --version 1.11.7 \
   --namespace kube-system \
   --reuse-values \
   --set hubble.relay.enabled=true \
   --set hubble.ui.enabled=true
```

We will see now a this output and the Hubble relay and UI should be enabled and ready to use.

```bash
Release "cilium" has been upgraded. Happy Helming!
NAME: cilium
LAST DEPLOYED: Fri Aug  5 11:38:11 2022
NAMESPACE: kube-system
STATUS: deployed
REVISION: 2
TEST SUITE: None
NOTES:
You have successfully installed Cilium with Hubble Relay and Hubble UI.

Your release version is 1.11.7.

For any further help, visit https://docs.cilium.io/en/v1.11/gettinghelp
```

To access the UI, we can either port-forward to the `hubble-ui` service or use the Cilium cli to access the Hubble UI in our browser.

I use the cli command:

```bash
cilium hubble ui
```
An browser window will be automtically opened with the Hubble UI ready to use.

```bash
ℹ️  Opening "http://localhost:12000" in your browser...
```

![image.png](https://cdn.hashnode.com/res/hashnode/image/upload/v1659694526854/L4ypcplrI.png align="left")

## Wrap-up

With this little trick, we can use now the Hubble UI in our Civo Cluster and have a great way to get more insight.

See the official docs about [Cilium and Hubble](https://docs.cilium.io/en/stable/intro/) for further details.
