# Leveraging Pulumi to Incorporate AWS Cognito as an Identity Provider for ArgoCD

## Introduction

In this blog post, I want to show you how to create and use [AWS Cognito](https://aws.amazon.com/cognito/) as an OAuth2 provider for ArgoCD. And this will be all done by using [Pulumi](https://www.pulumi.com/). This is very convenient, as you can do not need any manual steps to configure the ArgCD deployment with properties of the Cognito service. The Pulumi code will do all the work for you.

The demo infrastructure will be deployed to AWS and will look like this:

* AWS EKS Cluster

* AWS Cognito User Pool with a Client and Cognito User.

* AWS Load Balancer Controller

* External DNS (as I host my domains on DigitalOcean)

* ArgoCD


So let's get started!

## Prerequisites

To follow this article, you will need the following:

* [Pulumi CLI](https://www.pulumi.com/docs/get-started/install/) installed.

* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) installed.

* optional [K9s](https://k9scli.io/topics/install/), if you want to quickly interact with your cluster.

* AWS: [AWS account](https://aws.amazon.com/)


I will not go into detail on how to install the prerequisites. If you need help, please refer to the links above.

## Creating the infrastructure with Pulumi

### Create your Pulumi project

`Pulumi` is a multi-language infrastructure as Code tool using imperative languages to create a declarative infrastructure description.

You have a wide range of programming languages available, and you can use the one you and your team are the most comfortable with. Currently, (11/2022) `Pulumi` supports the following languages:

* Node.js (JavaScript / TypeScript)

* Python

* Go

* Java

* .NET (C#, VB, F#)

* YAML


In this article, we will use `Go` as our programming language. You can of course use any other language supported by `Pulumi`.

Create a project folder (for example `pulumi-cognito-argocd`) and navigate into the newly created directory:

```bash
mkdir pulumi-cognito-argocd
cd pulumi-cognito-argocd
```

Now, we need to initialize our project. We will use the `pulumi new` command to do this. This command will create a new Pulumi project in the current directory. We will use the `aws-go` template, which will create a new project with a `Go` template for AWS.

```bash
pulumi new aws-go --force
```

You can leave the default values in the prompt but maybe adjust the AWS region to your preference. I chose `eu-central-1` as my region for this demo.

As we're going to deploy several Helm charts, we need to install the `pulumi-kubernetes` provider. To install providers, we need to add the following go libarie to our `go.mod` file:

```bash
go get github.com/pulumi/pulumi-kubernetes/sdk/v3
```

And, as I don't want to create an EKS cluster from scratch, we will use the `pulumi-eks` provider to create the EKS cluster for us. We can configure the EKS cluster with some convenient options, like the Kubernetes version, OIDC support and more.

```bash
go get github.com/pulumi/pulumi-eks/sdk
```

### Creating the network infrastructure

With all libraries installed, we can start to create our infrastructure. First, we need to create the network infrastructure. To do this, head over to the `main.go` file and add the following code:

```go
package main

// omitted for brevity

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		vpc, err := ec2.NewVpc(ctx, "cognito-argocd-vpc", &ec2.VpcArgs{
			CidrBlock: pulumi.String("172.31.0.0/16"),
		})
		if err != nil {
			return err
		}
		igw, err := ec2.NewInternetGateway(ctx, "cognito-argocd-igw", &ec2.InternetGatewayArgs{
			VpcId: vpc.ID(),
		})
		if err != nil {
			return err
		}
		rt, err := ec2.NewRouteTable(ctx, "cognito-argocd-rt", &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: igw.ID(),
				},
			},
		})
		if err != nil {
			return err
		}

		var publicSubnetIDs pulumi.StringArray

		// Create a subnet for each availability zone
		for i, az := range availabilityZones {
			publicSubnet, err := ec2.NewSubnet(ctx, fmt.Sprintf("cognito-argocd-subnet-%d", i), &ec2.SubnetArgs{
				VpcId:                       vpc.ID(),
				CidrBlock:                   pulumi.String(publicSubnetCidrs[i]),
				MapPublicIpOnLaunch:         pulumi.Bool(true),
				AssignIpv6AddressOnCreation: pulumi.Bool(false),
				AvailabilityZone:            pulumi.String(az),
				Tags: pulumi.StringMap{
					"Name":                   pulumi.Sprintf("eks-public-subnet-%d", az),
					clusterTag:               pulumi.String("owned"),
					"kubernetes.io/role/elb": pulumi.String("1"),
				},
			})
			if err != nil {
				return err
			}
			_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("cognito-argocd-rt-association-%s", az), &ec2.RouteTableAssociationArgs{
				RouteTableId: rt.ID(),
				SubnetId:     publicSubnet.ID(),
			})
			if err != nil {
				return err
			}
			publicSubnetIDs = append(publicSubnetIDs, publicSubnet.ID())
		}

		return nil
	})
}
```

This code will create a new VPC with a public subnet in each availability zone. An internet gateway and a route table will be created as well. The route table will be associated with the public subnets.

### Creating the AWS Cognito infrastructure

With the network infrastructure created, we can work on the creation of the AWS Cognito infrastructure. Add the following code to the `main.go` file:

```go
package main

// omitted for brevity

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// omitted for brevity
		userPool, err := cognito.NewUserPool(ctx, "cognito-argocd-user-pool", &cognito.UserPoolArgs{
			AliasAttributes: pulumi.StringArray{
				pulumi.String("email"),
				pulumi.String("preferred_username"),
			},
			AutoVerifiedAttributes: pulumi.StringArray{
				pulumi.String("email"),
			},
			Schemas: cognito.UserPoolSchemaArray{
				&cognito.UserPoolSchemaArgs{
					AttributeDataType:      pulumi.String("String"),
					DeveloperOnlyAttribute: pulumi.Bool(false),
					Mutable:                pulumi.Bool(true),
					Name:                   pulumi.String("name"),
					Required:               pulumi.Bool(true),
					StringAttributeConstraints: &cognito.UserPoolSchemaStringAttributeConstraintsArgs{
						MinLength: pulumi.String("3"),
						MaxLength: pulumi.String("70"),
					},
				},
				&cognito.UserPoolSchemaArgs{
					AttributeDataType:      pulumi.String("String"),
					DeveloperOnlyAttribute: pulumi.Bool(false),
					Mutable:                pulumi.Bool(true),
					Name:                   pulumi.String("email"),
					Required:               pulumi.Bool(true),
					StringAttributeConstraints: &cognito.UserPoolSchemaStringAttributeConstraintsArgs{
						MinLength: pulumi.String("3"),
						MaxLength: pulumi.String("70"),
					},
				},
			},
			AdminCreateUserConfig: &cognito.UserPoolAdminCreateUserConfigArgs{
				AllowAdminCreateUserOnly: pulumi.Bool(true),
			},
		})
		if err != nil {
			return err
		}

		_, err = cognito.NewUser(ctx, "cognito-argocd-admin", &cognito.UserArgs{
			UserPoolId:        userPool.ID(),
			Username:          pulumi.String("admin"),
			TemporaryPassword: pulumi.String("Admin123!"),
			Attributes: pulumi.StringMap{
				"email":          pulumi.String(adminUserEmail),
				"email_verified": pulumi.String("true"),
			},
		})
		if err != nil {
			return err
		}

		_, err = cognito.NewUserPoolDomain(ctx, "cognito-argocd-user-pool-domain", &cognito.UserPoolDomainArgs{
			Domain:     pulumi.String("argocd-ui"),
			UserPoolId: userPool.ID(),
		})
		if err != nil {
			return err
		}

		userPoolClient, err := cognito.NewUserPoolClient(ctx, "cognito-argocd-user-pool-client", &cognito.UserPoolClientArgs{
			UserPoolId: userPool.ID(),
			AllowedOauthFlows: pulumi.StringArray{
				pulumi.String("code"),
				pulumi.String("implicit"),
			},
			AllowedOauthFlowsUserPoolClient: pulumi.Bool(true),
			AllowedOauthScopes: pulumi.StringArray{
				pulumi.String("openid"),
				pulumi.String("email"),
				pulumi.String("profile"),
			},
			SupportedIdentityProviders: pulumi.StringArray{
				pulumi.String("COGNITO"),
			},
			GenerateSecret: pulumi.Bool(true),
			CallbackUrls: pulumi.StringArray{
				pulumi.String("https://argocd.ediri.online/auth/callback"),
			},
		})
		if err != nil {
			return err
		}

		return nil
	})
}
```

This code will create the AWS Cognito user pool, an admin user, and a user pool client. Some important things to note here:

* The `adminUser` resource is created with a temporary password. This will be used to log in for the first time to the Argo CD UI. You will need to change the password after the first login.

* The domain name for the Argo CD UI is set to `argocd-ui`. This is a must to get OAuth2 working with Argo CD. You will need to change this to match your domain name or you can use one that is provided by AWS Cognito, which will be in the format `https://<domain_name>.auth.<region>.amazoncognito.com`.

* The `userPoolClient` resource is created with a callback URL. This is the URL that the user will be redirected to after they have authenticated with AWS Cognito. This URL will be used by the Argo CD UI to authenticate the user. The URL is set to `https://argocd.ediri.online/auth/callback` in this example, but you will need to change it to match your domain name.


When we later create the whole infrastructure, you will end up with a whole lot of endpoints:

* The AWS Cognito Auth endpoint: `https://<domain_name>.auth.<region>.amazoncognito.com/oauth2/authorize`

* The AWS Cognito Token endpoint: `https://<domain_name>.auth.<region>.amazoncognito.com/oauth2/token`

* The AWS Cognito User info endpoint: `https://<domain_name>.auth.<region>.amazoncognito.com/oauth2/userInfo`

* The AWS Cognito End session endpoint: `https://<domain_name>.auth.<region>.amazoncognito.com/logout`

* The AWS Cognito JWKS endpoint: `https://<domain_name>.auth.<region>.amazoncognito.com/.well-known/jwks.json`

* The AWS Cognito Issuer: `https://cognito-idp.<region>.amazonaws.com/<user_pool_id>`


The issuer, we will use later when we create the Argo CD configuration.

### Create the EKS cluster

After creating the network infrastructure and the AWS Cognito infrastructure, we can now create the EKS cluster. Add the following code to the `main.go` file:

```go
package main

// omitted for brevity

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// omitted for brevity
		cluster, err := eks.NewCluster(ctx, clusterName, &eks.ClusterArgs{
			Name:            pulumi.String(clusterName),
			VpcId:           vpc.ID(),
			SubnetIds:       publicSubnetIDs,
			InstanceType:    pulumi.String("t3.medium"),
			DesiredCapacity: pulumi.Int(2),
			MinSize:         pulumi.Int(1),
			MaxSize:         pulumi.Int(3),
			ProviderCredentialOpts: eks.KubeconfigOptionsArgs{
				ProfileName: pulumi.String("default"),
			},
			CreateOidcProvider: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		ctx.Export("kubeconfig", pulumi.ToSecret(cluster.Kubeconfig))

		return nil
	})
}
```

That's it! This very short code will create a full-blown EKS cluster with 2 worker nodes. The `kubeconfig` output will be made available as a secret in the Pulumi stack. We will use this later to connect to the cluster via `kubectl` or `k9s`.

But now comes the tricky part! We need to deploy several services to the cluster, to get Argo CD up and running. These services are:

* AWS Load Balancer Controller

* External DNS

* Argo CD


> For the TLS certificate, we will use the AWS Certificate Manager. I created everything beforehand, and this will be not covered in this post. In short, I created a CNAME record and my DNS provider, and the AWS Certificate Manager created a certificate for me. I use the `arn` of the certificate in the annotations of the Argo CD ingress.

### Deploy Argo CD

We reach the final part of this blog post! Add this code to the `main.go` file:

```go
package main

// omitted for brevity

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// omitted for brevity
		albRole, err := iam.NewRole(ctx, "cognito-argocd-alb-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.All(cluster.Core.OidcProvider().Arn(), cluster.Core.OidcProvider().Url()).ApplyT(func(args []interface{}) (string, error) {
				arn := args[0].(string)
				url := args[1].(string)
				return fmt.Sprintf(`{
						"Version": "2012-10-17",
						"Statement": [
							{
								"Effect": "Allow",
								"Principal": {
									"Federated": "%s"
								},
								"Action": "sts:AssumeRoleWithWebIdentity",
								"Condition": {
									"StringEquals": {
										"%s:sub": "%s"
									}
								}
							}
						]
					}`, arn, url, albServiceAccount), nil
			}).(pulumi.StringOutput),
		})
		if err != nil {
			return err
		}

		albPolicyFile, _ := os.ReadFile("../policies/alb-iam-policy.json")
		albIAMPolicy, err := iam.NewPolicy(ctx, "cognito-argocd-alb-policy", &iam.PolicyArgs{
			Policy: pulumi.String(string(albPolicyFile)),
		}, pulumi.DependsOn([]pulumi.Resource{albRole}))
		if err != nil {
			return err
		}
		_, err = iam.NewRolePolicyAttachment(ctx, "cognito-argocd-alb-role-attachment", &iam.RolePolicyAttachmentArgs{
			PolicyArn: albIAMPolicy.Arn,
			Role:      albRole.Name,
		}, pulumi.DependsOn([]pulumi.Resource{albIAMPolicy}))
		if err != nil {
			return err
		}

		provider, err := kubernetes.NewProvider(ctx, "my-provider", &kubernetes.ProviderArgs{
			Kubeconfig:            cluster.KubeconfigJson,
			EnableServerSideApply: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		ns, err := corev1.NewNamespace(ctx, albNamespace, &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(albNamespace),
				Labels: pulumi.StringMap{
					"app.kubernetes.io/name": pulumi.String("aws-load-balancer-controller"),
				},
			},
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}
		sa, err := corev1.NewServiceAccount(ctx, "aws-lb-controller-sa", &corev1.ServiceAccountArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("aws-lb-controller-serviceaccount"),
				Namespace: ns.Metadata.Name(),
				Annotations: pulumi.StringMap{
					"eks.amazonaws.com/role-arn": albRole.Arn,
				},
			},
		}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{ns}))
		if err != nil {
			return err
		}

		_, err = helm.NewRelease(ctx, "aws-load-balancer-controller", &helm.ReleaseArgs{
			Chart:     pulumi.String("aws-load-balancer-controller"),
			Version:   pulumi.String("1.4.8"),
			Namespace: ns.Metadata.Name(),
			RepositoryOpts: helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://aws.github.io/eks-charts"),
			},
			Values: pulumi.Map{
				"clusterName": cluster.EksCluster.ToClusterOutput().Name(),
				"region":      pulumi.String("eu-central-1"),
				"serviceAccount": pulumi.Map{
					"create": pulumi.Bool(false),
					"name":   sa.Metadata.Name(),
				},
				"vpcId": cluster.EksCluster.VpcConfig().VpcId(),
				"podLabels": pulumi.Map{
					"stack": pulumi.String("eks"),
					"app":   pulumi.String("aws-lb-controller"),
				},
			},
		}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{ns, sa}))
		if err != nil {
			return err
		}

		externalDNS, err := helm.NewRelease(ctx, "external-dns", &helm.ReleaseArgs{
			Chart:           pulumi.String("external-dns"),
			Version:         pulumi.String("1.12.2"),
			Namespace:       pulumi.String("external-dns"),
			CreateNamespace: pulumi.Bool(true),
			RepositoryOpts: helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://kubernetes-sigs.github.io/external-dns/"),
			},
			Values: pulumi.Map{
				"provider": pulumi.String("digitalocean"),
				"sources": pulumi.Array{
					pulumi.String("ingress"),
				},
				"env": pulumi.Array{
					pulumi.Map{
						"name":  pulumi.String("DO_TOKEN"),
						"value": config.GetSecret(ctx, "do"),
					},
				},
			},
		}, pulumi.Provider(provider))

		_, err = helm.NewRelease(ctx, "argocd", &helm.ReleaseArgs{
			Chart:           pulumi.String("argo-cd"),
			Version:         pulumi.String("5.28.0"),
			Namespace:       pulumi.String("argocd"),
			CreateNamespace: pulumi.Bool(true),
			RepositoryOpts: helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://argoproj.github.io/argo-helm"),
			},
			Values: pulumi.Map{
				"dex": pulumi.Map{
					"enabled": pulumi.Bool(false),
				},
				"server": pulumi.Map{
					"extraArgs": pulumi.Array{
						pulumi.String("--insecure"),
						pulumi.String("--enable-gzip"),
					},
					"ingress": pulumi.Map{
						"enabled":          pulumi.Bool(true),
						"hosts":            pulumi.Array{pulumi.String("argocd.ediri.online")},
						"ingressClassName": pulumi.String("alb"),
						"annotations": pulumi.Map{
							"alb.ingress.kubernetes.io/target-type":        pulumi.String("ip"),
							"alb.ingress.kubernetes.io/scheme":             pulumi.String("internet-facing"),
							"alb.ingress.kubernetes.io/load-balancer-name": pulumi.String("argocd"),
							"alb.ingress.kubernetes.io/certificate-arn":    config.GetSecret(ctx, "cert_arn"),
						},
					},
				},
				"configs": pulumi.Map{
					"rbac": pulumi.Map{
						"policy.default": pulumi.String("role:readonly"),
						"policy.csv":     pulumi.Sprintf(`g, %s, role:admin`, adminUserEmail),
						"scopes":         pulumi.String(`[email]`),
					},
					"cm": pulumi.Map{
						"admin.enabled": pulumi.Bool(false),
						"url":           pulumi.String("https://argocd.ediri.online"),
						"oidc.config": pulumi.Sprintf(`name: Cognito
issuer: https://cognito-idp.eu-central-1.amazonaws.com/%s
clientID: %s
clientSecret: %s
requestedScopes: ["openid", "profile", "email"]
requestedIDTokenClaims: {"email": {"essential": true}}`, userPool.ID(), userPoolClient.ID(), userPoolClient.ClientSecret),
					},
				},
			},
		}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{ns, externalDNS}))
		if err != nil {
			return err
		}

		return nil
	})
}
```

There is a lot, and I mean really a lot going on here. I will try to explain it as best as I can. Let's start with the ALB Controller part:

```go
package main

// omitted for brevity

func main() {
	albRole, err := iam.NewRole(ctx, "cognito-argocd-alb-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.All(cluster.Core.OidcProvider().Arn(), cluster.Core.OidcProvider().Url()).ApplyT(func(args []interface{}) (string, error) {
			arn := args[0].(string)
			url := args[1].(string)
			return fmt.Sprintf(`{
						"Version": "2012-10-17",
						"Statement": [
							{
								"Effect": "Allow",
								"Principal": {
									"Federated": "%s"
								},
								"Action": "sts:AssumeRoleWithWebIdentity",
								"Condition": {
									"StringEquals": {
										"%s:sub": "%s"
									}
								}
							}
						]
					}`, arn, url, albServiceAccount), nil
		}).(pulumi.StringOutput),
	})
	if err != nil {
		return err
	}

	albPolicyFile, _ := os.ReadFile("../policies/alb-iam-policy.json")
	albIAMPolicy, err := iam.NewPolicy(ctx, "cognito-argocd-alb-policy", &iam.PolicyArgs{
		Policy: pulumi.String(string(albPolicyFile)),
	}, pulumi.DependsOn([]pulumi.Resource{albRole}))
	if err != nil {
		return err
	}
	_, err = iam.NewRolePolicyAttachment(ctx, "cognito-argocd-alb-role-attachment", &iam.RolePolicyAttachmentArgs{
		PolicyArn: albIAMPolicy.Arn,
		Role:      albRole.Name,
	}, pulumi.DependsOn([]pulumi.Resource{albIAMPolicy}))
	if err != nil {
		return err
	}
}
```

We define the role that will be assumed by the ALB Controller. The `AssumeRolePolicy` field specifies the trust policy for the role, which allows the ALB to assume the role using the OpenID Connect (OIDC) provider for the EKS cluster.

The code then creates an AWS IAM policy for the ALB to use with the `iam.NewPolicy` function. The Policy field specifies the contents of the policy, which is read from a JSON file using the `os.ReadFile` function. The role is then attached to the policy using the `iam.NewRolePolicyAttachment` function.

Next, we create the ALB Controller itself:

```go
package main

// omitted for brevity

func main() {
	// omitted for brevity
	provider, err := kubernetes.NewProvider(ctx, "my-provider", &kubernetes.ProviderArgs{
		Kubeconfig:            cluster.KubeconfigJson,
		EnableServerSideApply: pulumi.Bool(true),
	})
	if err != nil {
		return err
	}
	ns, err := corev1.NewNamespace(ctx, albNamespace, &corev1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String(albNamespace),
			Labels: pulumi.StringMap{
				"app.kubernetes.io/name": pulumi.String("aws-load-balancer-controller"),
			},
		},
	}, pulumi.Provider(provider))
	if err != nil {
		return err
	}
	sa, err := corev1.NewServiceAccount(ctx, "aws-lb-controller-sa", &corev1.ServiceAccountArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("aws-lb-controller-serviceaccount"),
			Namespace: ns.Metadata.Name(),
			Annotations: pulumi.StringMap{
				"eks.amazonaws.com/role-arn": albRole.Arn,
			},
		},
	}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{ns}))
	if err != nil {
		return err
	}

	_, err = helm.NewRelease(ctx, "aws-load-balancer-controller", &helm.ReleaseArgs{
		Chart:     pulumi.String("aws-load-balancer-controller"),
		Version:   pulumi.String("1.4.8"),
		Namespace: ns.Metadata.Name(),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://aws.github.io/eks-charts"),
		},
		Values: pulumi.Map{
			"clusterName": cluster.EksCluster.ToClusterOutput().Name(),
			"region":      pulumi.String("eu-central-1"),
			"serviceAccount": pulumi.Map{
				"create": pulumi.Bool(false),
				"name":   sa.Metadata.Name(),
			},
			"vpcId": cluster.EksCluster.VpcConfig().VpcId(),
			"podLabels": pulumi.Map{
				"stack": pulumi.String("eks"),
				"app":   pulumi.String("aws-lb-controller"),
			},
		},
	}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{ns, sa}))
	if err != nil {
		return err
	}
	externalDNS, err := helm.NewRelease(ctx, "external-dns", &helm.ReleaseArgs{
		Chart:           pulumi.String("external-dns"),
		Version:         pulumi.String("1.12.2"),
		Namespace:       pulumi.String("external-dns"),
		CreateNamespace: pulumi.Bool(true),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://kubernetes-sigs.github.io/external-dns/"),
		},
		Values: pulumi.Map{
			"provider": pulumi.String("digitalocean"),
			"sources": pulumi.Array{
				pulumi.String("ingress"),
			},
			"env": pulumi.Array{
				pulumi.Map{
					"name":  pulumi.String("DO_TOKEN"),
					"value": config.GetSecret(ctx, "do"),
				},
			},
		},
	}, pulumi.Provider(provider))
}
```

The `kubernetes.NewProvider()` function is creating a new provider that will manage Kubernetes resources, we are using the server-side apply feature.

We then create a new namespace for the ALB Controller using the `corev1.NewNamespace` function. Next, we create a new service account for the ALB Controller using the `corev1.NewServiceAccount` function. The `Annotations` field specifies the ARN of the role that the ALB Controller will assume.

Finally, we create the ALB Controller using the `helm.NewRelease` function. Important to note here is that we are passing the already created service account as a value to the `serviceAccount.name` field. This is important because the ALB Controller will use the service account to assume the role that we created earlier.

The `helm.NewRelease` function is also used to create the External DNS controller. This is really because of me, as I want that the address of the ingress will be added as an A record in my DigitalOcean DNS. You can skip this part if you do use for example AWS Route53.

```go
package main

// omitted for brevity

func main() {
	// omitted for brevity
	_, err = helm.NewRelease(ctx, "argocd", &helm.ReleaseArgs{
		Chart:           pulumi.String("argo-cd"),
		Version:         pulumi.String("5.28.0"),
		Namespace:       pulumi.String("argocd"),
		CreateNamespace: pulumi.Bool(true),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://argoproj.github.io/argo-helm"),
		},
		Values: pulumi.Map{
			"dex": pulumi.Map{
				"enabled": pulumi.Bool(false),
			},
			"server": pulumi.Map{
				"extraArgs": pulumi.Array{
					pulumi.String("--insecure"),
					pulumi.String("--enable-gzip"),
				},
				"ingress": pulumi.Map{
					"enabled":          pulumi.Bool(true),
					"hosts":            pulumi.Array{pulumi.String("argocd.ediri.online")},
					"ingressClassName": pulumi.String("alb"),
					"annotations": pulumi.Map{
						"alb.ingress.kubernetes.io/target-type":        pulumi.String("ip"),
						"alb.ingress.kubernetes.io/scheme":             pulumi.String("internet-facing"),
						"alb.ingress.kubernetes.io/load-balancer-name": pulumi.String("argocd"),
						"alb.ingress.kubernetes.io/certificate-arn":    config.GetSecret(ctx, "cert_arn"),
					},
				},
			},
			"configs": pulumi.Map{
				"rbac": pulumi.Map{
					"policy.default": pulumi.String("role:readonly"),
					"policy.csv":     pulumi.Sprintf(`g, %s, role:admin`, adminUserEmail),
					"scopes":         pulumi.String(`[email]`),
				},
				"cm": pulumi.Map{
					"admin.enabled": pulumi.Bool(false),
					"url":           pulumi.String("https://argocd.ediri.online"),
					"oidc.config": pulumi.Sprintf(`name: Cognito
issuer: https://cognito-idp.eu-central-1.amazonaws.com/%s
clientID: %s
clientSecret: %s
requestedScopes: ["openid", "profile", "email"]
requestedIDTokenClaims: {"email": {"essential": true}}`, userPool.ID(), userPoolClient.ID(), userPoolClient.ClientSecret),
				},
			},
		},
	}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{ns, externalDNS}))
	if err != nil {
		return err
	}
}
```

The last part of the code is creating the ArgoCD deployment. We are using the `helm.NewRelease` function again to create the deployment. The `Values` field contains the configuration for the deployment. The `server.ingress` field contains the now some interesting parts. We need to set the `ingressClassName` to `alb` so that the ingress will be managed by the ALB Controller. We also need to set the `alb.ingress.kubernetes.io/certificate-arn` annotation to the ARN of the certificate in our AWS Certificate Manager. We also need to set the `alb.ingress.kubernetes.io/scheme` and the `alb.ingress.kubernetes.io/target-type` annotations to `internet-facing` and `ip` respectively.

The `configs.cm.oidc.config` field contains the configuration for the OIDC provider. We are using Cognito as the OIDC provider set the `issuer` field to the URL of the Cognito user pool. The `clientID` and `clientSecret` fields are the ID and secret of the Cognito user pool client. The `requestedScopes` and `requestedIDTokenClaims` fields are the scopes and claims that we want to request from the OIDC provider.

Don't forget to set the `url` field to your domain name, in my case `argocd.ediri.online`.

## Deploying the stack

Finally, we can deploy the stack. We can do this by running the following command:

```bash
pulumi up
```

This will create the stack and deploy the resources. This will take a few minutes and after that, you should be able to access the ArgoCD dashboard at the URL that you specified! In my case, this is `https://argocd.ediri.online`.

![](https://cdn.hashnode.com/res/hashnode/image/upload/v1680725205287/3e91be85-18a2-4a2b-b616-ff5604db5926.png align="center")

![](https://cdn.hashnode.com/res/hashnode/image/upload/v1680725231629/07692133-d09d-40b1-b7a4-7a9f66ca16f8.png align="center")

![](https://cdn.hashnode.com/res/hashnode/image/upload/v1680725266127/099a53d9-493e-41f3-abf3-3d18b5c62eb6.png align="center")

![](https://cdn.hashnode.com/res/hashnode/image/upload/v1680725287097/6d2b5402-0e66-4fa6-848a-233773f42f36.png align="center")



## Housekeeping

After you are done with the stack, you can destroy it by running the following command:

```bash
pulumi destroy
```

This will destroy all the resources that were created by the stack.

## Conclusion

Using AWS Cognito as an OIDC provider for ArgoCD is a great way to secure your ArgoCD deployment. If you already run your infrastructure on AWS it is very easy to set Cognito up and configure it to use it as an OAuth2 provider for ArgoCD.

## Resources

* [ArgoCD](https://argo-cd.readthedocs.io/en/stable/)

* [AWS Load Balancer Controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller)

* [AWS Cognito](https://aws.amazon.com/cognito/)

* [AWS Certificate Manager](https://aws.amazon.com/certificate-manager/)

* [AWS IAM](https://aws.amazon.com/iam/)
