package user.kubernetes.ED001

import future.keywords.in
import data.lib.kubernetes
import data.lib.result

default allowedRegistries = ["quay.io","ghcr.io","gcr.io"]

__rego_metadata__ := {
  "id": "ED001",
  "title": "Allowed container registry checks",
  "severity": "CRITICAL",
  "description": "The usage of non allowed container registries is not allowed",
}

__rego_input__ := {
  "combine": false,
  "selector": [{"type": "kubernetes"}],
}

allowedRegistry(image) {
  registry := allowedRegistries[_]
  startswith(image, registry)
}

deny[res] {
  container := kubernetes.containers[_]
  not allowedRegistry(container.image)
  msg :=  kubernetes.format(sprintf("Container '%s' of %s '%s' comes from not approved container registry", [container.name, kubernetes.kind, kubernetes.name]))
  res := result.new(msg, container)
}
