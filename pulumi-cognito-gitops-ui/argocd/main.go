package main

import (
	"fmt"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/cognito"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"os"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi-eks/sdk/go/eks"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	clusterName       = "cognito-argocd-eks-cluster"
	clusterTag        = "kubernetes.io/cluster/" + clusterName
	albNamespace      = "aws-lb-controller"
	albServiceAccount = "system:serviceaccount:" + albNamespace + ":aws-lb-controller-serviceaccount"
	ebsServiceAccount = "system:serviceaccount:kube-system:ebs-csi-controller-sa"

	adminUserEmail = "info@ediri.de"
)

var (
	publicSubnetCidrs = []string{
		"172.31.0.0/20",
		"172.31.48.0/20",
	}
	availabilityZones = []string{
		"eu-central-1a",
		"eu-central-1b",
	}
)

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

		ebsRole, err := iam.NewRole(ctx, "cognito-argocd-ebs-role", &iam.RoleArgs{
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
			}`, arn, url, ebsServiceAccount), nil
			}).(pulumi.StringOutput),
		})
		if err != nil {
			return err
		}
		ebsPolicyFile, _ := os.ReadFile("../policies/ebs-iam-policy.json")
		ebsIAMPolicy, err := iam.NewPolicy(ctx, "cognito-argocd-ebs-policy", &iam.PolicyArgs{
			Policy: pulumi.String(string(ebsPolicyFile)),
		}, pulumi.DependsOn([]pulumi.Resource{ebsRole}))
		if err != nil {
			return err
		}

		ebsA, _ := iam.NewRolePolicyAttachment(ctx, "cognito-argocd-esb-role-attachment", &iam.RolePolicyAttachmentArgs{
			PolicyArn: ebsIAMPolicy.Arn,
			Role:      ebsRole.Name,
		}, pulumi.DependsOn([]pulumi.Resource{ebsIAMPolicy}))

		provider, err := kubernetes.NewProvider(ctx, "my-provider", &kubernetes.ProviderArgs{
			Kubeconfig:            cluster.KubeconfigJson,
			EnableServerSideApply: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		_, err = helm.NewRelease(ctx, "aws-ebs-csi-driver", &helm.ReleaseArgs{
			Chart:       pulumi.String("aws-ebs-csi-driver"),
			Version:     pulumi.String("2.17.1"),
			Namespace:   pulumi.String("kube-system"),
			ForceUpdate: pulumi.Bool(true),
			RepositoryOpts: helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://kubernetes-sigs.github.io/aws-ebs-csi-driver"),
			},
			Values: pulumi.Map{
				"controller": pulumi.Map{
					"serviceAccount": pulumi.Map{
						"annotations": pulumi.Map{
							"eks.amazonaws.com/role-arn": ebsRole.Arn,
						},
					},
				},
			},
		}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{ebsA}))
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
