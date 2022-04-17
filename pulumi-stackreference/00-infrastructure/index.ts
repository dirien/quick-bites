import * as pulumi from "@pulumi/pulumi";
import * as digitalocean from "@pulumi/digitalocean";

let doks = new digitalocean.KubernetesCluster("k8s-cluster", {
  name: "quick-bites-cluster",
  ha: false,
  version: "1.22.8-do.0",
  autoUpgrade: true,
  maintenancePolicy: {
    day: "sunday",
    startTime: "03:00",
  },
  region: "fra1",
  nodePool: {
    name: "default",
    size: "s-2vcpu-4gb",
    autoScale: false,
    nodeCount: 1,
  },
});

export const kubeConfig = pulumi.secret(doks.kubeConfigs[0].rawConfig)
