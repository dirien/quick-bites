import * as aws from "@pulumi/aws";
import * as pulumi from "@pulumi/pulumi";
import * as k8s from "@pulumi/kubernetes";
import {getIssuerCAThumbprint} from '@pulumi/eks/cert-thumprint'
import * as https from "https";

const vpc = new aws.ec2.Vpc("aws-vpc", {
    cidrBlock: "10.0.0.0/16",
    enableDnsHostnames: true,
    enableDnsSupport: true,
})


const gw = new aws.ec2.InternetGateway("aws-igw", {
    vpcId: vpc.id,
})

const rt = new aws.ec2.RouteTable("aws-rt", {
    vpcId: vpc.id,
    routes: [{
        gatewayId: gw.id,
        cidrBlock: "0.0.0.0/0",
    }],
})

const subnet1 = new aws.ec2.Subnet("aws-subnet-1", {
    vpcId: vpc.id,
    cidrBlock: "10.0.48.0/20",
    mapPublicIpOnLaunch: true,
    availabilityZone: "eu-central-1a",
})

const subnet2 = new aws.ec2.Subnet("aws-subnet-2", {
    vpcId: vpc.id,
    cidrBlock: "10.0.64.0/20",
    mapPublicIpOnLaunch: true,
    availabilityZone: "eu-central-1b",
})

const subnets = [subnet1, subnet2]

for (let i = 0; i < subnets.length; i++) {
    new aws.ec2.RouteTableAssociation('aws-rt-association-' + i, {
        subnetId: subnets[i].id,
        routeTableId: rt.id,
    })
    const eip = new aws.ec2.Eip("aws-eip-" + i, {
        vpc: true,
    })

    new aws.ec2.NatGateway("aws-nat-" + i, {
        subnetId: subnets[i].id,
        allocationId: eip.id,
    })
}


const eksPolicy = aws.iam.getPolicyDocumentOutput({
    version: "2012-10-17",
    statements: [{
        actions: ["sts:AssumeRole"],
        principals: [{
            identifiers: ["eks.amazonaws.com"],
            type: "Service",
        }],
        effect: "Allow",
    }],
})


const eksRole = new aws.iam.Role("aws-eks-role", {
    assumeRolePolicy: eksPolicy.json,
})

new aws.iam.RolePolicyAttachment("aws-iam-rpa-1", {
    role: eksRole.name,
    policyArn: "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
})

new aws.iam.RolePolicyAttachment("aws-iam-rpa-2", {
    role: eksRole.name,
    policyArn: "arn:aws:iam::aws:policy/AmazonEKSServicePolicy",
})

const ec2Policy = aws.iam.getPolicyDocumentOutput({
    version: "2012-10-17",
    statements: [{
        actions: ["sts:AssumeRole"],
        principals: [{
            identifiers: ["ec2.amazonaws.com"],
            type: "Service",
        }],
        effect: "Allow",
    }],
})

const nodeRole = new aws.iam.Role("aws-ec2-role", {
    assumeRolePolicy: ec2Policy.json,
})

new aws.iam.RolePolicyAttachment("aws-iam-rpa-3", {
    role: nodeRole.name,
    policyArn: "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
})

new aws.iam.RolePolicyAttachment("aws-rpa-4", {
    role: nodeRole.name,
    policyArn: "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
})

new aws.iam.RolePolicyAttachment("aws-rpa-5", {
    role: nodeRole.name,
    policyArn: "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
})

const securityGroup = new aws.ec2.SecurityGroup("aws-sg", {
    description: "EKS Security Group",
    vpcId: vpc.id,
    ingress: [{
        description: "Allow HTTP from VPC",
        fromPort: 80,
        toPort: 80,
        protocol: "tcp",
        cidrBlocks: [
            "0.0.0.0/0",
        ],
    }],
    egress: [{
        description: "Allow all outbound traffic",
        fromPort: 0,
        toPort: 0,
        protocol: "-1",
        cidrBlocks: [
            "0.0.0.0/0",
        ],
    }]
})

const eks = new aws.eks.Cluster("aws-eks", {
    roleArn: eksRole.arn,
    version: "1.24",
    vpcConfig: {
        subnetIds: [
            subnet1.id,
            subnet2.id,
        ],
        securityGroupIds: [
            securityGroup.id
        ],
        publicAccessCidrs: [
            "0.0.0.0/0",
        ],
    },
    tags: {
        "Name": "pulumi-eks",
    }
})

new aws.eks.Addon("aws-eks-addon", {
    clusterName: eks.name,
    addonName: "vpc-cni",
    addonVersion: "v1.12.0-eksbuild.2",
    resolveConflicts: "OVERWRITE",
})

const nodeGroup = new aws.eks.NodeGroup("aws-eks-ng", {
    clusterName: eks.name,
    nodeRoleArn: nodeRole.arn,
    subnetIds: [
        subnet1.id,
        subnet2.id,
    ],
    nodeGroupName: "aws-eks-ng",
    scalingConfig: {
        desiredSize: 1,
        maxSize: 2,
        minSize: 1,
    }
})

const kubeconfig = pulumi.all([eks.name, eks.endpoint, eks.certificateAuthority]).apply(([name, endpoint, certificateAuthority]) => {
    const context = `${aws.config.region}/${name}`;
    return `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: ${certificateAuthority.data}
    server: ${endpoint}
  name: ${context}
contexts:
- context:
    cluster: ${context}
    user: admin
  name: admin
current-context: admin
kind: Config
users:
- name: admin
  user:
   exec:
        apiVersion: client.authentication.k8s.io/v1beta1
        command: aws-iam-authenticator
        args:
        - token
        - -i
        - ${name}
preferences: {}
`;
})

const secret = new aws.secretsmanager.Secret("aws-sm", {
    name: "pulumi-secret-demo",
    description: "Pulumi Secret",
})


const secretVersion = new aws.secretsmanager.SecretVersion("aws-sv", {
    secretId: secret.id,
    secretString: `{"username":"hello", "password":"world"}`
})

const fingerprint = getIssuerCAThumbprint(eks.identities[0].oidcs[0].issuer, new https.Agent({
        maxCachedSessions: 0
    }
))

new aws.iam.OpenIdConnectProvider("aws-eks-oidc-provider", {
    url: eks.identities[0].oidcs[0].issuer,
    clientIdLists: ["sts.amazonaws.com"],
    thumbprintLists: [fingerprint],
}, {dependsOn: eks})

const current = aws.getCallerIdentity({});
export const accountId = current.then(current => current.accountId);

let oidcId = pulumi.interpolate`${eks.identities[0].oidcs[0].issuer}`.apply(id => {
    return id.replace("https://", "")
})

let trust = pulumi.interpolate`{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Federated": "arn:aws:iam::${accountId}:oidc-provider/${oidcId}"
            },
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Condition": {
                "StringEquals": {
                    "${oidcId}:sub": "system:serviceaccount:default:test",
                    "${oidcId}:aud": "sts.amazonaws.com"
                }
            }
        }
    ]
}`

const role = new aws.iam.Role("aws-secret-reader-role", {
    name: "secret-reader",
    assumeRolePolicy: trust,
})

const secretPolicyDocument = aws.iam.getPolicyDocumentOutput({
    version: "2012-10-17",
    statements: [
        {
            actions: [
                "secretsmanager:GetSecretValue",
                "secretsmanager:DescribeSecret",
            ],
            resources: [
                secret.id
            ],
            effect: "Allow",
        },
    ],
})

const secretPolicy = new aws.iam.Policy("aws-secret-policy", {
    policy: secretPolicyDocument.json,
})


new aws.iam.RolePolicyAttachment("aws-iam-rpa-6", {
    role: role.name,
    policyArn: pulumi.interpolate`arn:aws:iam::${accountId}:policy/${secretPolicy.name}`,
})

const provider = new k8s.Provider("k8s-provider", {
    kubeconfig: kubeconfig,
    enableServerSideApply: true,
}, {dependsOn: [eks, nodeGroup]})

const csiStoreDriver = new k8s.helm.v3.Release("k8s-secrets-store-csi-driver", {
    chart: "secrets-store-csi-driver",
    namespace: "kube-system",
    repositoryOpts: {
        repo: "https://kubernetes-sigs.github.io/secrets-store-csi-driver/charts",
    }
}, {provider: provider})

new k8s.helm.v3.Release("k8s-secrets-store-csi-driver-provider-aws", {
    chart: "secrets-store-csi-driver-provider-aws",
    namespace: "kube-system",
    repositoryOpts: {
        repo: "https://aws.github.io/secrets-store-csi-driver-provider-aws",
    }
}, {provider: provider, dependsOn: [csiStoreDriver]})


new k8s.core.v1.ServiceAccount("k8s-sa", {
    metadata: {
        name: "test",
        namespace: "default",
        annotations: {
            "eks.amazonaws.com/role-arn": role.arn,
        }
    }
}, {provider: provider})

const secretProviderClass = new k8s.apiextensions.CustomResource("k8s-cr", {
    apiVersion: "secrets-store.csi.x-k8s.io/v1alpha1",
    kind: "SecretProviderClass",
    metadata: {
        name: "aws-secret-provider",
        namespace: "default",
    },
    spec: {
        provider: "aws",
        parameters: {
            objects: pulumi.interpolate`- objectName: "${secret.arn}"
  objectType: "secretsmanager"
  objectAlias: "${secret.name}"`,
        }
    }
}, {provider: provider})

new k8s.apps.v1.Deployment("k8s-demo-deployment-authorized", {
    apiVersion: "apps/v1",
    kind: "Deployment",
    metadata: {
        name: "hello-server-deployment-authorized",
        labels: {
            app: "hello-server-authorized",
        },
        annotations: {
            "pulumi.com/skipAwait": "true",
        }
    },
    spec: {
        replicas: 1,
        selector: {
            matchLabels: {
                app: "hello-server-authorized",
            },
        },
        template: {
            metadata: {
                labels: {
                    app: "hello-server-authorized",
                },
            },
            spec: {
                serviceAccountName: "test",
                volumes: [{
                    name: "secrets-store-inline",
                    csi: {
                        driver: "secrets-store.csi.k8s.io",
                        readOnly: true,
                        volumeAttributes: {
                            secretProviderClass: secretProviderClass.metadata.name,
                        },
                    },
                }],
                containers: [{
                    name: "hello-server-authorized",
                    image: "ghcr.io/dirien/hello-server/hello-server:latest",
                    ports: [{
                        containerPort: 8080,
                    }],
                    env: [{
                        name: "FILE",
                        value: pulumi.interpolate`/mnt/secrets-store/${secret.name}`,
                    }],
                    volumeMounts: [{
                        name: "secrets-store-inline",
                        mountPath: "/mnt/secrets-store",
                        readOnly: true,
                    }],
                }],
            },
        },
    },
}, {provider: provider});

new k8s.apps.v1.Deployment("k8s-demo-deployment-unauthorized", {
    apiVersion: "apps/v1",
    kind: "Deployment",
    metadata: {
        name: "hello-server-deployment-unauthorized",
        labels: {
            app: "hello-server-unauthorized",
        },
        annotations: {
            "pulumi.com/skipAwait": "true",
        }
    },
    spec: {
        replicas: 1,
        selector: {
            matchLabels: {
                app: "hello-server-unauthorized",
            },
        },
        template: {
            metadata: {
                labels: {
                    app: "hello-server-unauthorized",
                },
            },
            spec: {
                serviceAccountName: "default",
                volumes: [{
                    name: "secrets-store-inline",
                    csi: {
                        driver: "secrets-store.csi.k8s.io",
                        readOnly: true,
                        volumeAttributes: {
                            secretProviderClass: secretProviderClass.metadata.name,
                        },
                    },
                }],
                containers: [{
                    name: "hello-server-unauthorized",
                    image: "ghcr.io/dirien/hello-server/hello-server:latest",
                    ports: [{
                        containerPort: 8080,
                    }],
                    env: [{
                        name: "FILE",
                        value: pulumi.interpolate`/mnt/secrets-store/${secret.name}`,
                    }],
                    volumeMounts: [{
                        name: "secrets-store-inline",
                        mountPath: "/mnt/secrets-store",
                        readOnly: true,
                    }],
                }],
            },
        },
    },
}, {provider: provider});

export const kubeConfig = pulumi.secret(kubeconfig)
