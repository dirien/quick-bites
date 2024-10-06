"""A DigitalOcean Python Pulumi program"""

import pulumi
import pulumi_digitalocean as digitalocean
import pulumi_kubernetes as kubernetes

do_cluster = digitalocean.KubernetesCluster("do_cluster",
    name="esc-cluster",
    region="nyc1",
    version="1.31.1-do.1",
    destroy_all_associated_resources=True,
    node_pool=digitalocean.KubernetesClusterNodePoolArgs(
        name="default",
        size="s-2vcpu-2gb",
        node_count=1
    )
)

do_k8s_provider = kubernetes.Provider("do_k8s_provider",
    enable_server_side_apply=True,
    kubeconfig=do_cluster.kube_configs[0].apply(lambda config: config.raw_config)
)

namespace = kubernetes.core.v1.Namespace("external-secrets",
    metadata={
        "name": "external-secrets",
    },
opts=pulumi.ResourceOptions(provider=do_k8s_provider))

# Deploy a Helm release into the namespace
external_secrets = kubernetes.helm.v3.Release("external-secrets",
    chart="external-secrets",
    version="0.10.4",  # Specify the version of the chart
    namespace=namespace.metadata["name"],
    repository_opts={
        "repo": "https://charts.external-secrets.io",
    },
opts=pulumi.ResourceOptions(provider=do_k8s_provider))

# Deploy a secret into the namespace
pulumi_access_token = pulumi.Config().require("pulumi-pat")

my_secret = kubernetes.core.v1.Secret("my-secret",
    metadata={
        "namespace": namespace.metadata["name"],
        "name": "pulumi-access-token",
    },
    string_data={
        "PULUMI_ACCESS_TOKEN": pulumi_access_token,
    },
    type="Opaque",
opts=pulumi.ResourceOptions(provider=do_k8s_provider))

pulumi.export("kubeconfig", do_k8s_provider.kubeconfig)

cluster_secret_store = kubernetes.apiextensions.CustomResource("cluster-secret-store",
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
opts=pulumi.ResourceOptions(provider=do_k8s_provider))

external_secret = kubernetes.apiextensions.CustomResource("external-secret",
    api_version="external-secrets.io/v1beta1",
    kind="ExternalSecret",
    metadata=kubernetes.meta.v1.ObjectMetaArgs(
        name= "esc-secret-store",
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
        }
    },
opts=pulumi.ResourceOptions(provider=do_k8s_provider))


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
                        ports=[kubernetes.core.v1.ContainerPortArgs(
                            container_port=8080,
                        )],
                        resources=kubernetes.core.v1.ResourceRequirementsArgs(
                            limits=None,
                            requests=None,
                        ),
                    )
                ],
            ),
        ),
    ),
opts=pulumi.ResourceOptions(provider=do_k8s_provider))