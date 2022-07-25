package user.kubernetes.ED001

import future.keywords.in
import data.lib.result

default allowedContainerRegistry = "docker.io"

__rego_metadata__ := {
    "id": "ED001",
    "title": "Docker Hub not allowed",
    "severity": "CRITICAL",
    "description": "The usage of Docker Hub as container registry is not allowed.",
}

__rego_input__ := {
    "selector": [
        {"type": "kubernetes"},
    ],
}

deny[res] {
    input.kind == "Pod"
    some container in input.spec.containers
    not startswith(container.image, allowedContainerRegistry)
    msg := sprintf("Image '%v' comes from not approved container registry in `%v`", [container.image, allowedContainerRegistry])
    res := result.new(msg, container)
}
