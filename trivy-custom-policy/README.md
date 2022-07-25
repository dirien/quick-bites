# How to write custom policies for Trivy

## Trivy?

`Trivy` (tri pronounced like trigger, vy pronounced like envy) is a simple and comprehensive
vulnerability/misconfiguration/secret scanner for containers and other artifacts developed by Aqua Security.

To know more about `Trivy`, I highly recommend to check following videos and of course
the [official documentation](https://aquasecurity.github.io/trivy).

%[https://youtu.be/bgYrhQ6rTXA]

%[https://youtu.be/6Vw0QgJ-k5o]

I installed the `Trivy` cli on macOS via `Homebrew` with following command:

```bash
brew install aquasecurity/trivy/trivy
```
Check the [installation docs](https://aquasecurity.github.io/trivy/v0.30.2/getting-started/installation/) for your operating system.

## Introduction to misconfiguration policies

In this short blog article, I want to explain how you can write your own custom policies for `Trivy`. This to detect any misconfiguration in your configuration files.

Currently, `Trivy` supports following types of configuration files:

- Kubernetes
- Dockerfile, Containerfile
- Terraform
- CloudFormation
- Helm Chart
- RBAC

There is already a large set of [build-in policies](https://github.com/aquasecurity/defsec/tree/master/internal/rules) for these configuration files provided by the good people at AquaSecurity and the `Trivy` community.

### Write your first custom policy

Most important item you need to keep in mind is that custom policies in `Trivy` are written in `Rego`. I highly suggest to familiarize yourself with the [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/) language.

As mention above, `Trivy` supports certain configuration files and detect the right policy via the file extension the type of the configuration.

| File extension                                      | Configuration              |
|-----------------------------------------------------|----------------------------|
| *.yaml, *.yml and *.json                            | Kubernetes / Helm          |
| Dockerfile, Dockerfile.*, and *.Dockerfile          | Dockerfile                 |
| Containerfile, Containerfile.*, and *.Containerfile | Containerfile              |
| *.yaml, *.yml and *.json                            | CloudFormation             |
| *.tf and *.tf.json                                  | Terraform / Terraform Plan |

#### Anatomy of a custom policy

I will use a very simple use case to explain the anatomy of a custom policy.

Let's assume following scenario: You want only allow that Pods from a specific container registry are allowed to be deployed to your cluster.

```rego
package user.kubernetes.ED001

import future.keywords
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
```

`package` is a required field and MUST be unique per policy. It must start with namespace name, the rest is up to you as long as it is unique. Here in my example I use `user.kubernetes.ED001` as package name.

- As namespace I chose `user`
- A group name for clarity (`kubernetes`)
- and policy id (`ED001`).

The namespace we will use later as a parameter, when we call `Trivy` to scan the files.

`import future.keywords` is a special import that allows to use future keywords in your policy.

`import data.lib.result` is a special import that allows to use the `result` library to highlight the findings.

`__rego_metadata__`  helps enrich `Trivy`'s scan results with useful information. All fields are optional. Please check
the [official documentation](https://aquasecurity.github.io/trivy/v0.30.2/docs/misconfiguration/custom/#metadata) for
more information on all available fields and their meaning.

`__rego_input__` an optional field that allows to filter the input data. Here in my example I only want to scan `Kubernetes` resources and ignore any other configuration types.

`deny` is a required field. According to AquaSecurity `warn`, `violation` also work for compatibility but `deny` is recommended to use. You can always use `severity` field in the `__rego_metadata__`

So what does my `deny` do in detail?

First, we check that we only apply the rule on type `Pod`. Then we iterate over all containers in the `Pod` and check if the image of the container starts with the `allowedContainerRegistry`.

If not, we build a message pointing the issue and highlight the container with the help of `result.new(msg, container)`

```rego
deny[res] {
    input.kind == "Pod"
    some container in input.spec.containers
    not startswith(container.image, allowedContainerRegistry)
    msg := sprintf("Image '%v' comes from not approved container registry in `%v`", [container.image, allowedContainerRegistry])
    res := result.new(msg, container)
}
```

#### Calling Trivy with our custom policy

I created a basic file structure for my custom policies. Under the `policies` directory I created subdirectories for each configuration file type.

To scan the files, I used the following command:

```bash
trivy conf --config-policy policies --policy-namespaces user config/
```

The subcommand `config` tells `Trivy` to call the scanning of config files. The flag `--config-policy` specify paths to the custom policy files directory and flag `--policy-namespaces` is the namespace we defined above.

As argument to the subcommand `config` I used the path to the directory containing the configuration files I want to scan.

You should see following output

```bash
2022-07-25T18:10:55.035+0200    INFO    Misconfiguration scanning is enabled
2022-07-25T18:10:55.213+0200    INFO    Detected config files: 1

pod.yaml (kubernetes)

Tests: 3 (SUCCESSES: 1, FAILURES: 2, EXCEPTIONS: 0)
Failures: 2 (CRITICAL: 2)

CRITICAL: Image 'nginx' comes from not approved container registry in `docker.io`
════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════
The usage of Docker Hub as container registry is not allowed.
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
 pod.yaml:9-11
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
   9 ┌   - image: nginx
  10 │     name: nginx
  11 └     resources: {}
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────


CRITICAL: Image 'nginx' comes from not approved container registry in `docker.io`
════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════
The usage of Docker Hub as container registry is not allowed.
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
 pod.yaml:12-14
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
  12 ┌   - image: nginx
  13 │     name: nginx
  14 └     resources: {}
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
```
