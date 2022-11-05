package main

import "exoscale.pulumi.com/minecraft/sks:sks"

servers: [ sks.#ExoscaleMinecraft & {
	resourceName:    "server-1"
	sksZone:         "de-fra-1"
	sksCNI:          "cilium"
	sksNodePoolSize: 3
	minecraftServerProperties: {
		type: "paper"
	}
},
	sks.#ExoscaleMinecraft & {
		resourceName:    "server-2"
		sksZone:         "de-fra-1"
		sksCNI:          "calico"
		sksNodePoolSize: 3
		minecraftServerProperties: {
			type: "paper"
		}
	},
]

for i, server in servers {
	variables: {
		server.variables
	}

	resources: {
		server.resources
	}

	outputs: {
		server.outputs
	}
}
