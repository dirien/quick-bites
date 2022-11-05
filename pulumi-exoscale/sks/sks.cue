package sks

import "strings"

#MinecraftServerProperties: {
	version: *"latest" | string
	type:    *"VANILLA" | string
	motd:    *"Hello from Exoscale" | string
}

#ExoscaleMinecraft: {
	resourceName:              strings.MaxRunes(10) & =~"^[a-z0-9-]+$"
	sksZone:                   string
	sksVersion:                *"1.25.1" | string
	sksCNI:                    *"calico" | string
	sksNodePoolSize:           int & >=1 & <=10
	sksNodePoolInstanceType:   *"standard.medium" | string
	mcNamespace:               *"minecraft" | string
	mcChartVersion:            *"4.4.0" | string
	minecraftServerProperties: #MinecraftServerProperties

	variables: {
		"default-sec-group": {
			"fn::invoke": {
				function: "exoscale:index/getSecurityGroup:getSecurityGroup"
				arguments: {
					name: "default"
				}
			}
		}
	}

	resources: {
		"\(resourceName)-mc-cluster": {
			type: "exoscale:SKSCluster"
			properties: {
				zone:          sksZone
				name:          "\(resourceName)-mc-cluster"
				exoscaleCcm:   true
				metricsServer: true
				version:       sksVersion
				cni:           sksCNI
			}
		}

		"\(resourceName)-mc-sks-anti-affinity-group": {
			type: "exoscale:AntiAffinityGroup"
			properties:
				name: "\(resourceName)-mc-sks-anti-affinity-group"
		}

		"\(resourceName)-mc-sks-security-group": {
			type: "exoscale:SecurityGroup"
			properties:
				name: "\(resourceName)-mc-sks-security-group"
		}

		"\(resourceName)-kubelet-security-group-rule": {
			type: "exoscale:SecurityGroupRule"
			properties: {
				securityGroupId:     "${\(resourceName)-mc-sks-security-group.id}"
				description:         "Kubelet"
				userSecurityGroupId: "${\(resourceName)-mc-sks-security-group.id}"
				type:                "INGRESS"
				protocol:            "TCP"
				startPort:           10250
				endPort:             10250
			}
		}
		"\(resourceName)-nodeport-tcp-security-group-rule": {
			type: "exoscale:SecurityGroupRule"
			properties: {
				securityGroupId: "${\(resourceName)-mc-sks-security-group.id}"
				description:     "Nodeport TCP services"
				cidr:            "0.0.0.0/0"
				type:            "INGRESS"
				protocol:        "TCP"
				startPort:       30000
				endPort:         32767
			}
		}
		"\(resourceName)-nodeport-udp-security-group-rule": {
			type: "exoscale:SecurityGroupRule"
			properties: {
				securityGroupId: "${\(resourceName)-mc-sks-security-group.id}"
				description:     "Nodeport UDP services"
				cidr:            "0.0.0.0/0"
				type:            "INGRESS"
				protocol:        "UDP"
				startPort:       30000
				endPort:         32767
			}
		}

		"\(resourceName)-cilium-healthcheck-icmp-security-group-rule": {
			type: "exoscale:SecurityGroupRule"
			properties: {
				securityGroupId:     "${\(resourceName)-mc-sks-security-group.id}"
				description:         "Cilium (healthcheck)"
				userSecurityGroupId: "${\(resourceName)-mc-sks-security-group.id}"
				type:                "INGRESS"
				protocol:            "ICMP"
				icmpType:            8
				icmpCode:            0
			}
		}

		if sksCNI == "cilium" {
			"\(resourceName)-cilium-vxlan-security-group-rule": {
				type: "exoscale:SecurityGroupRule"
				properties: {
					securityGroupId:     "${\(resourceName)-mc-sks-security-group.id}"
					description:         "Cilium (vxlan)"
					userSecurityGroupId: "${\(resourceName)-mc-sks-security-group.id}"
					type:                "INGRESS"
					protocol:            "UDP"
					startPort:           8472
					endPort:             8472
				}
			}
			"\(resourceName)-cilium-healthcheck-tcp-security-group-rule": {
				type: "exoscale:SecurityGroupRule"
				properties: {
					securityGroupId:     "${\(resourceName)-mc-sks-security-group.id}"
					description:         "Cilium (healthcheck)"
					userSecurityGroupId: "${\(resourceName)-mc-sks-security-group.id}"
					type:                "INGRESS"
					protocol:            "TCP"
					startPort:           4240
					endPort:             4240
				}
			}
		}

		if sksCNI == "calico" {
			"\(resourceName)-calico-vxlan-security-group-rule": {
				type: "exoscale:SecurityGroupRule"
				properties: {
					securityGroupId:     "${\(resourceName)-mc-sks-security-group.id}"
					description:         "Calico (vxlan)"
					userSecurityGroupId: "${\(resourceName)-mc-sks-security-group.id}"
					type:                "INGRESS"
					protocol:            "UDP"
					startPort:           4789
					endPort:             4789
				}
			}
		}

		"\(resourceName)-minecraft-tcp-security-group-rule": {
			type: "exoscale:SecurityGroupRule"
			properties: {
				securityGroupId: "${\(resourceName)-mc-sks-security-group.id}"
				description:     "Minecraft TCP services"
				cidr:            "0.0.0.0/0"
				type:            "INGRESS"
				protocol:        "TCP"
				startPort:       25565
				endPort:         25565
			}
		}
		"\(resourceName)-mc-nodepool": {
			type: "exoscale:SKSNodepool"
			properties: {
				name:         "\(resourceName)-mc-nodepool"
				zone:         sksZone
				clusterId:    "${\(resourceName)-mc-cluster.id}"
				instanceType: sksNodePoolInstanceType
				size:         sksNodePoolSize
				diskSize:     100
				antiAffinityGroupIds: [
					"${\(resourceName)-mc-sks-anti-affinity-group.id}",
				]
				securityGroupIds: [
					"${default-sec-group.id}",
					"${\(resourceName)-mc-sks-security-group.id}",
				]
			}
		}
		"\(resourceName)-mc-sks-kubeconfig": {
			type: "exoscale:SKSKubeconfig"
			properties: {
				zone:      sksZone
				clusterId: "${\(resourceName)-mc-cluster.id}"
				user:      "kubernetes-admin"
				groups: [
					"system:masters",
				]
			}
		}
		"\(resourceName)-mc-k8s-provider": {
			type: "pulumi:providers:kubernetes"
			properties: {
				kubeconfig:            "${\(resourceName)-mc-sks-kubeconfig.kubeconfig}"
				enableServerSideApply: true
			}
		}
		"\(resourceName)-minecraft-namespace": {
			type: "kubernetes:core/v1:Namespace"
			properties: {
				metadata:
					name: mcNamespace
			}
			options:
				provider: "${\(resourceName)-mc-k8s-provider}"
		}
		"\(resourceName)-minecraft-chart": {
			type: "kubernetes:helm.sh/v3:Release"
			properties: {
				name:      "minecraft"
				namespace: "${\(resourceName)-minecraft-namespace}"
				chart:     "minecraft"
				version:   mcChartVersion
				repositoryOpts:
					repo: "https://itzg.github.io/minecraft-server-charts/"
				values: {
					resources: {
						requests: {
							memory: "128Mi"
							cpu:    "100m"
						}
					}
					minecraftServer: {
						eula:        true
						version:     minecraftServerProperties.version
						type:        minecraftServerProperties.type
						motd:        minecraftServerProperties.motd
						serviceType: "LoadBalancer"
					}
				}
			}
			options:
				provider: "${\(resourceName)-mc-k8s-provider}"
		}
	}

	outputs: {
		"\(resourceName)-kubeconfig": {
			"fn::secret": "${\(resourceName)-mc-sks-kubeconfig.kubeconfig}"
		}
	}
}
