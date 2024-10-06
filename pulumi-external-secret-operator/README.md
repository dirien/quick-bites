# Advanced Secret Management on Kubernetes with Pulumi: External Secrets Operator
## TL;DR Le code

## Introduction

This article is part three of my series on secret management on Kubernetes with the help of [Pulumi](https://www.pulumi.com/). In my first article, we talked about the <mark>Sealed Secrets</mark> controller. The second article, we talked about the <mark>Secrets Store CSI Driver</mark> and how it compares to the <mark>Sealed Secrets</mark> controller.

If you haven't read those articles yet, I recommend you do so before continuing with this one:

%[https://blog.ediri.io/advanced-secret-management-on-kubernetes-with-pulumi-and-gitops-sealed-secrets-controller] 

%[https://blog.ediri.io/advanced-secret-management-on-kubernetes-with-pulumi-secrets-store-csi-driver] 

And as we saw in the previous articles, managing secrets in [Kubernetes](https://kubernetes.io/) can be a pain in the neck. But we all agree. Proper secret management is vital for our apps and infrastructure security. Especially, and now comes the important part, if we store our secrets in a Git repository. This is also an essential part of the [GitOps](https://opengitops.dev/) workflow. We saw that the <mark>Sealed Secrets</mark> controller could encrypt our secrets and store them in Git. Yet, we soon recognized its flaws. Think about the nightmare of managing keys for many projects, clusters, and secrets used by various teams. And, rotating them.

The second solution we saw was the <mark>Secrets Store CSI Driver</mark>. This solution is great. But you must build your app with the <mark>Secrets Store CSI Driver</mark> in mind to retrieve the secrets from a file. You could argue this is manageable.

But what about external tools that don't support this driver?

### What is the External Secrets Operator?

Here comes the <mark>External Secrets Operator (ESO)</mark> to the rescue. The ESO is build following the [Kubernetes operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) and manages secrets in a highly secure and scalable way. ESO syncs secrets in external systems, like Pulumi ESC, HashiCorp Vault, AWS Secrets Manager, and Azure Key Vault, into Kubernetes secrets. This lets us tame secret sprawl. It centralizes secret management in one place. It also provides a secure, controlled way to access them.

ESO offers us the following benefits:

* By leveraging external secret management systems, ESO acts as a bridge between Kubernetes and external secret management.

* With ESO, we don't have any manual intervention to manage secrets in Kubernetes. Everything is automated and managed by ESO.

* ESO providers support secret rotation when the external secret management system supports it.

* Through the unified interface ESO provides, handling cross-cluster and cross-cloud secret management becomes straightforward and independent of the underlying infrastructure.


### The Architecture of the External Secrets Operator

ESO is built on top of the Kubernetes operator pattern and extends the Kubernetes API with several different custom resource definitions (CRDs) to manage secrets. ESO is watching for these CRDs, and when secrets at the external secret management system match the defined CRDs, ESO will synchronize the secrets into Kubernetes secrets. Now we have the secret as part of the Kubernetes reconciliation loop. This means that the secret is always up to date and in sync with the external secret management system.

![high-level](https://external-secrets.io/latest/pictures/diagrams-high-level-simple.png align="left")

Custom Resource Definitions (CRDs) of the External Secrets Operator

Following CRDs, you should know when working with ESO:

* `(Cluster)SecretStore`: This CRD defines the external secret management system that ESO should use to manage secrets. It contains the configuration to connect to the external secret management system. `SecretStore` is namespaced, while the `ClusterSecretStore` is cluster-scoped, which means that it can be used across many namespaces.

* `(Cluster)ExternalSecret`: This CRD defines the secret that should be synchronized into Kubernetes. It contains the reference to the `SecretStore` and the path to the secret in the external secret management system. `ExternalSecret` is namespaced, while the `ClusterExternalSecret` is cluster-scoped, which means that it can be used across many namespaces.


![Component Overview](https://external-secrets.io/latest/pictures/diagrams-component-overview.png align="left")

## Setting Up the External Secrets Operator

Like the other two articles, we will use <mark>Pulumi</mark> to deploy the External Secrets Operator to our Kubernetes cluster. This time, I am going to use <mark>DigitalOcean</mark> as the cloud provider and <mark>Pulumi ESC</mark> as our external secret management system. But you can use any other cloud provider and external secret management system you like.

### What is Pulumi ESC?

Pulumi ESC provides a comprehensive solution for managing secrets, environments, and configurations. While it integrates seamlessly with Pulumi IaC projects through pulumiConfig, exposing stored values to your Pulumi stacks, its functionality extends beyond this use case.

As a standalone service, Pulumi ESC supports diverse applications with dedicated SDKs for various programming languages. Additionally, it offers a CLI for command-line management of secrets and configurations, enabling context creation for CLI tools like Terraform.

This new secrets management and orchestration service can be employed both within and outside of Pulumi's Infrastructure as Code ecosystem. To explore Pulumi ESC's capabilities further, consult the official [documentation](https://www.pulumi.com/docs/esc/).

### Prerequisites

To follow this article, you will need the following:

* [Pulumi CLI](https://www.pulumi.com/docs/get-started/install/) is installed.

* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) installed.

* optional [K9s](https://k9scli.io/topics/install/), if you want to quickly interact with your cluster.

* [DigitalOcean](https://try.digitalocean.com/cloud/) account.


I will skip the whole Pulumi installation and config part. I explained this in my previous posts.

### Create a new Pulumi project

In the previous articles, I used `TypeScript` and `Go`, but to show you that you can use a lot of different languages with Pulumi, it's time for <mark>Python</mark>.

We start by creating a new Pulumi project:

```shell
mkdir pulumi-external-secret-operator
cd pulumi-external-secret-operator
pulumi new python --force
```

Leave the default values and at the end you should see something like this:

```shell
pulumi new digitalocean-python --force
This command will walk you through creating a new Pulumi project.

Enter a value or leave blank to accept the (default), and press <ENTER>.
Press ^C at any time to quit.

Project name (pulumi-external-secret-operator):
Project description (A minimal DigitalOcean Python Pulumi program):
Created project 'pulumi-external-secret-operator'

Please enter your desired stack name.
To create a stack in an organization, use the format <org-name>/<stack-name> (e.g. `acmecorp/dev`).
Stack name (dev):
Created stack 'dev'

The toolchain to use for installing dependencies and running the program pip
Installing dependencies...

Creating virtual environment...
Finished creating virtual environment
Updating pip, setuptools, and wheel in virtual environment...
Requirement already satisfied: pip in ./venv/lib/python3.12/site-packages (24.2)
Collecting setuptools
  Using cached setuptools-75.1.0-py3-none-any.whl.metadata (6.9 kB)
Collecting wheel
  Using cached wheel-0.44.0-py3-none-any.whl.metadata (2.3 kB)
Using cached setuptools-75.1.0-py3-none-any.whl (1.2 MB)
Using cached wheel-0.44.0-py3-none-any.whl (67 kB)
Installing collected packages: wheel, setuptools
Successfully installed setuptools-75.1.0 wheel-0.44.0
Finished updating
Installing dependencies in virtual environment...
Collecting pulumi<4.0.0,>=3.0.0 (from -r requirements.txt (line 1))
Downloading pulumi-3.135.1-py3-none-any.whl (273 kB)
Downloading pulumi_digitalocean-4.33.0-py3-none-any.whl (377 kB)
Installing collected packages: arpeggio, six, semver, pyyaml, protobuf, grpcio, dill, debugpy, attrs, pulumi, parver, pulumi-digitalocean
Successfully installed arpeggio-2.0.2 attrs-24.2.0 debugpy-1.8.6 dill-0.3.9 grpcio-1.60.2 parver-0.5 protobuf-4.25.5 pulumi-3.135.1 pulumi-digitalocean-4.33.0 pyyaml-6.0.2 semver-2.13.0 six-1.16.0
Finished installing dependencies
Finished installing dependencies

Your new project is ready to go! ✨

To perform an initial deployment, run `pulumi up`
```

### Writing the Pulumi code to deploy the External Secrets Operator

After the project is created, we can start writing our code. You should see the following files in your project:

```shell
tree -L 1
.
├── Pulumi.yaml
├── __main__.py
├── requirements.txt
└── venv

2 directories, 3 files
```

Add the `pulumi-kubernetes` dependencies to your `requirements.txt` file:

```shell
pulumi>=3.0.0,<4.0.0
pulumi-digitalocean>=4.0.0,<5.0.0
pulumi-kubernetes==4.18.1
```

Install the dependencies:

```shell
./venv/bin/pip install -r requirements.txt
```

Replace the content of the `__main__.py` file with the following code:

```python
"""A DigitalOcean Python Pulumi program"""

import pulumi
import pulumi_digitalocean as digitalocean
import pulumi_kubernetes as kubernetes

do_cluster = digitalocean.KubernetesCluster(
    "do_cluster",
    name="esc-cluster",
    region="nyc1",
    version="1.31.1-do.1",
    destroy_all_associated_resources=True,
    node_pool=digitalocean.KubernetesClusterNodePoolArgs(
        name="default", size="s-2vcpu-2gb", node_count=1
    ),
)

do_k8s_provider = kubernetes.Provider(
    "do_k8s_provider",
    enable_server_side_apply=True,
    kubeconfig=do_cluster.kube_configs[0].apply(lambda config: config.raw_config),
)

namespace = kubernetes.core.v1.Namespace(
    "external-secrets",
    metadata={
        "name": "external-secrets",
    },
    opts=pulumi.ResourceOptions(provider=do_k8s_provider),
)

# Deploy a Helm release into the namespace
external_secrets = kubernetes.helm.v3.Release(
    "external-secrets",
    chart="external-secrets",
    version="0.10.4",  # Specify the version of the chart
    namespace=namespace.metadata["name"],
    repository_opts={
        "repo": "https://charts.external-secrets.io",
    },
    opts=pulumi.ResourceOptions(provider=do_k8s_provider),
)

# Deploy a secret into the namespace
pulumi_access_token = pulumi.Config().require("pulumi-pat")

my_secret = kubernetes.core.v1.Secret(
    "my-secret",
    metadata={
        "namespace": namespace.metadata["name"],
        "name": "pulumi-access-token",
    },
    string_data={
        "PULUMI_ACCESS_TOKEN": pulumi_access_token,
    },
    type="Opaque",
    opts=pulumi.ResourceOptions(provider=do_k8s_provider),
)

pulumi.export("kubeconfig", do_k8s_provider.kubeconfig)
```

This code creates a new DigitalOcean Kubernetes cluster. It has one node and a size of s-2vcpu-2gb. We also create a dedicated Kubernetes provider for this cluster as we need it later to deploy the External Secrets Operator, the ClusterSecretStore, and ExternalSecret.

Before we can deploy and create the cluster, we need to provide the DigitalOcean token. There are different ways to achieve this. You can set the token as an environment variable or use the Pulumi configuration.

I am going to use the most secure way and use Pulumi ESC to store the token and provide it to the Pulumi stack. To do this, you need to create a new Pulumi ESC environment. You can do this by running the following command:

```shell
pulumi env init <your-org>/eso-do-cluster/eso-dev
```

We define the `DigitalOcean` token inside the `Pulumi ESC` environment using following sytnax:

```yaml
values:
  pulumiConfig:
    digitalocean:token:
      fn::secret: <your-do-token>
    pulumi-pat:
      fn::secret: <your-pulumi-pat>
```

Replace &lt;your-do-pat&gt; with your DigitalOcean token. You can find the token in your DigitalOcean account settings. And replace &lt;your-pulumi-pat&gt; with your [Pulumi Personal Access Token](https://www.pulumi.com/docs/pulumi-cloud/access-management/access-tokens/).

To apply the configuration, run the pulumi env edit command and copy the above YAML into the editor:

```bash
pulumi env edit <your-org>/eso-do-cluster/eso-dev
```

Last step is to link the `Pulumi ESC` environment to the Pulumi stack by creating a new `Pulumi.dev.yaml` file:

```shell
cat <<EOF > Pulumi.dev.yaml
environment:
- eso-do-cluster/eso-dev
EOF
```

Now we can deploy the stack:

```shell
pulumi up
```

You can check that everything is running by running the following command:

```shell
pulumi stack output kubeconfig --show-secrets > kubeconfig.yaml
kubectl --kubeconfig kubeconfig.yaml get secret -n external-secrets pulumi-access-token -o jsonpath='{.data.PULUMI_ACCESS_TOKEN}' | base64 -d
```

You should see your Pulumi Personal Access Token printed to the console.

### Create an external secret and use it in a deployment.

Now that the External Secrets Operator is running, we can create an ExternalSecret and use it in a deployment.

For this, we are going to create a new ESC project called eso-to-esc-app and a development environment:

```shell
pulumi env init <your-org>/eso-to-esc-app/dev
```

Add the following values to the `Pulumi ESC` environment:

```bash
values:
  app:
    hello: world
    hello-secret:
      fn::secret: world
```

Then run this command:

```shell
pulumi env edit <your-org>/eso-to-esc-app/dev
```

Now we can create the `ClusterSecretStore`:

```python
"""A DigitalOcean Python Pulumi program"""

import pulumi
import pulumi_digitalocean as digitalocean
import pulumi_kubernetes as kubernetes

# Cut for brevity

cluster_secret_store = kubernetes.apiextensions.CustomResource(
    "cluster-secret-store",
    api_version="external-secrets.io/v1beta1",
    kind="ClusterSecretStore",
    metadata=kubernetes.meta.v1.ObjectMetaArgs(
        name="secret-store",
    ),
    spec={
        "provider": {
            "pulumi": {
                "organization": pulumi.get_organization(),
                "project": "eso-to-esc-app",
                "environment": "dev",
                "accessToken": {
                    "secretRef": {
                        "name": my_secret.metadata.name,
                        "key": "PULUMI_ACCESS_TOKEN",
                        "namespace": my_secret.metadata.namespace,
                    },
                },
            },
        },
    },
    opts=pulumi.ResourceOptions(provider=do_k8s_provider),
)
```

Finally we can create the `ExternalSecret` and wire it up to our demo application:

```python
"""A DigitalOcean Python Pulumi program"""

import pulumi
import pulumi_digitalocean as digitalocean
import pulumi_kubernetes as kubernetes

# Cut for brevity

external_secret = kubernetes.apiextensions.CustomResource(
    "external-secret",
    api_version="external-secrets.io/v1beta1",
    kind="ExternalSecret",
    metadata=kubernetes.meta.v1.ObjectMetaArgs(
        name="esc-secret-store",
    ),
    spec={
        "dataFrom": [
            {
                "extract": {
                    "conversionStrategy": "Default",
                    "key": "app",
                }
            }
        ],
        "refreshInterval": "10s",
        "secretStoreRef": {
            "kind": cluster_secret_store.kind,
            "name": cluster_secret_store.metadata["name"],
        },
    },
    opts=pulumi.ResourceOptions(provider=do_k8s_provider),
)

hello_server_deployment = kubernetes.apps.v1.Deployment(
    "hello-server-deployment",
    metadata=kubernetes.meta.v1.ObjectMetaArgs(
        name="hello",
        labels={"app": "hello"},
    ),
    spec=kubernetes.apps.v1.DeploymentSpecArgs(
        replicas=1,
        selector=kubernetes.meta.v1.LabelSelectorArgs(
            match_labels={"app": "hello"},
        ),
        template=kubernetes.core.v1.PodTemplateSpecArgs(
            metadata=kubernetes.meta.v1.ObjectMetaArgs(
                labels={"app": "hello"},
            ),
            spec=kubernetes.core.v1.PodSpecArgs(
                containers=[
                    kubernetes.core.v1.ContainerArgs(
                        name="hello-server",
                        image="ghcr.io/dirien/hello-server/hello-server:latest",
                        env_from=[
                            kubernetes.core.v1.EnvFromSourceArgs(
                                secret_ref=kubernetes.core.v1.SecretEnvSourceArgs(
                                    name=external_secret.metadata["name"]
                                )
                            )
                        ],
                        ports=[
                            kubernetes.core.v1.ContainerPortArgs(
                                container_port=8080,
                            )
                        ],
                        resources=kubernetes.core.v1.ResourceRequirementsArgs(
                            limits=None,
                            requests=None,
                        ),
                    )
                ],
            ),
        ),
    ),
    opts=pulumi.ResourceOptions(provider=do_k8s_provider),
)
```

Deploy all the changes by running:

```shell
pulumi up
```

After the deployment is finished, you can check the logs of the `hello-server` pod by running:

```shell
pod_name=$(kubectl --kubeconfig kubeconfig.yaml get pods -l app=hello -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward --kubeconfig kubeconfig.yaml $pod_name 8080:8080
```

Now you can access the `hello-server` by running:

```shell
curl http://localhost:8080/env/hello
curl http://localhost:8080/env/hello-secret

hello=world
hello-secret=world%
```

Bingo! We successfully retrieved the secret from the `External Secrets Operator` and used it in our application.

### Housekeeping

Don't forget to clean up your resources by running:

```shell
pulumi destroy
```

## Conclusion

The External Secrets Operator is a great tool to manage secrets in Kubernetes. It provides a secure and efficient way to manage secrets in a cloud-native environment, improving security, efficiency, and compliance when consuming secrets in your Kubernetes cluster. All this is done by not adding any more complexity to your applications and operations. Especially from an operational perspective, the External Secrets Operator is great as it follows the Kubernetes way of providing a declarative API to manage secrets and how it plays well with the GitOps workflow.

All in all, I highly recommend you give the External Secrets Operator a try and see how it can help your organization manage secrets in a secure and efficient manner.

## Resources

* [External Secrets Operator](https://external-secrets.io/latest/)

* [Pulumi ESC](https://www.pulumi.com/docs/esc/)
