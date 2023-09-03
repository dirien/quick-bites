package main

import (
	"fmt"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/s3"
	"github.com/pulumi/pulumi-eks/sdk/go/eks"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"os"
)

const (
	clusterName          = "velero-eks-cluster"
	clusterTag           = "kubernetes.io/cluster/" + clusterName
	ebsServiceAccount    = "system:serviceaccount:kube-system:ebs-csi-controller-sa"
	veleroNamespace      = "velero"
	veleroServiceAccount = "system:serviceaccount:" + veleroNamespace + ":velero"
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
		vpc, err := ec2.NewVpc(ctx, "velero-vpc", &ec2.VpcArgs{
			CidrBlock: pulumi.String("172.31.0.0/16"),
		})
		if err != nil {
			return err
		}

		igw, err := ec2.NewInternetGateway(ctx, "velero-igw", &ec2.InternetGatewayArgs{
			VpcId: vpc.ID(),
		})
		if err != nil {
			return err
		}

		rt, err := ec2.NewRouteTable(ctx, "velero-rt", &ec2.RouteTableArgs{
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
			publicSubnet, err := ec2.NewSubnet(ctx, fmt.Sprintf("velero-subnet-%d", i), &ec2.SubnetArgs{
				VpcId:                       vpc.ID(),
				CidrBlock:                   pulumi.String(publicSubnetCidrs[i]),
				MapPublicIpOnLaunch:         pulumi.Bool(true),
				AssignIpv6AddressOnCreation: pulumi.Bool(false),
				AvailabilityZone:            pulumi.String(az),
				Tags: pulumi.StringMap{
					"Name":     pulumi.Sprintf("eks-public-subnet-%d", az),
					clusterTag: pulumi.String("owned"),
				},
			})
			if err != nil {
				return err
			}
			_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("velero-rt-association-%s", az), &ec2.RouteTableAssociationArgs{
				RouteTableId: rt.ID(),
				SubnetId:     publicSubnet.ID(),
			})
			if err != nil {
				return err
			}
			publicSubnetIDs = append(publicSubnetIDs, publicSubnet.ID())
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

		ebsRole, err := iam.NewRole(ctx, "velero-ebs-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.All(cluster.Core.OidcProvider().Arn(), cluster.Core.OidcProvider().Url()).ApplyT(func(args []interface{}) string {
				arn := args[0].(string)
				url := args[1].(string)
				assumeRolePolicy, _ := iam.GetPolicyDocument(ctx, &iam.GetPolicyDocumentArgs{
					Statements: []iam.GetPolicyDocumentStatement{
						{
							Effect: pulumi.StringRef("Allow"),
							Actions: []string{
								"sts:AssumeRoleWithWebIdentity",
							},
							Principals: []iam.GetPolicyDocumentStatementPrincipal{
								{
									Type: "Federated",
									Identifiers: []string{
										arn,
									},
								},
							},
							Conditions: []iam.GetPolicyDocumentStatementCondition{
								{
									Test: "StringEquals",
									Values: []string{
										ebsServiceAccount,
									},
									Variable: fmt.Sprintf("%s:sub", url),
								},
							},
						},
					},
				})
				return assumeRolePolicy.Json
			}).(pulumi.StringOutput),
		})
		if err != nil {
			return err
		}

		ebsPolicyFile, _ := os.ReadFile("./policies/ebs-iam-policy.json")

		ebsIAMPolicy, err := iam.NewPolicy(ctx, "velero-ebs-policy", &iam.PolicyArgs{
			Policy: pulumi.String(ebsPolicyFile),
		}, pulumi.DependsOn([]pulumi.Resource{ebsRole}))
		if err != nil {
			return err
		}

		ebsRolePolicyAttachment, err := iam.NewRolePolicyAttachment(ctx, "velero-esb-role-attachment", &iam.RolePolicyAttachmentArgs{
			PolicyArn: ebsIAMPolicy.Arn,
			Role:      ebsRole.Name,
		}, pulumi.DependsOn([]pulumi.Resource{ebsIAMPolicy}))
		if err != nil {
			return err
		}

		provider, err := kubernetes.NewProvider(ctx, "kubernetes-provider", &kubernetes.ProviderArgs{
			Kubeconfig:            cluster.KubeconfigJson,
			EnableServerSideApply: pulumi.Bool(true),
		}, pulumi.DependsOn([]pulumi.Resource{cluster}))
		if err != nil {
			return err
		}

		_, err = helm.NewRelease(ctx, "aws-ebs-csi-driver", &helm.ReleaseArgs{
			Chart:       pulumi.String("aws-ebs-csi-driver"),
			Version:     pulumi.String("2.22.0"),
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
				"storageClasses": pulumi.Array{
					pulumi.Map{
						"name":              pulumi.String("ebs-sc"),
						"volumeBindingMode": pulumi.String("WaitForFirstConsumer"),
					},
				},
				"volumeSnapshotClasses": pulumi.Array{
					pulumi.Map{
						"name":           pulumi.String("ebs-vsc"),
						"deletionPolicy": pulumi.String("Delete"),
					},
				},
			},
		}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{ebsRolePolicyAttachment}))
		if err != nil {
			return err
		}

		// create the s3 bucket and the public access block for velero
		bucket, err := s3.NewBucket(ctx, "velero-bucket", &s3.BucketArgs{
			Bucket: pulumi.String("velero-eks-bucket-dirien"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("velero-eks-bucket-dirien"),
			},
		})
		if err != nil {
			return err
		}
		_, err = s3.NewBucketPublicAccessBlock(ctx, "velero-bucket-public-access-block", &s3.BucketPublicAccessBlockArgs{
			Bucket:                bucket.ID(),
			BlockPublicAcls:       pulumi.Bool(true),
			BlockPublicPolicy:     pulumi.Bool(true),
			IgnorePublicAcls:      pulumi.Bool(true),
			RestrictPublicBuckets: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// do the IRSA stuff for velero
		saRole, err := iam.NewRole(ctx, "velero-sa-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.All(cluster.Core.OidcProvider().Arn(), cluster.Core.OidcProvider().Url()).ApplyT(func(args []interface{}) string {
				arn := args[0].(string)
				url := args[1].(string)
				assumeRolePolicy, _ := iam.GetPolicyDocument(ctx, &iam.GetPolicyDocumentArgs{
					Statements: []iam.GetPolicyDocumentStatement{
						{
							Actions: []string{
								"sts:AssumeRoleWithWebIdentity",
							},
							Conditions: []iam.GetPolicyDocumentStatementCondition{
								{
									Test: "StringEquals",
									Values: []string{
										veleroServiceAccount,
									},
									Variable: fmt.Sprintf("%s:sub", url),
								},
							},
							Principals: []iam.GetPolicyDocumentStatementPrincipal{
								{
									Type: "Federated",
									Identifiers: []string{
										arn,
									},
								},
							},
							Effect: pulumi.StringRef("Allow"),
						},
					},
				})
				return assumeRolePolicy.Json
			}).(pulumi.StringOutput),
		})
		if err != nil {
			return err
		}

		veleroIAMPolicy, err := iam.NewPolicy(ctx, "velero-iam-policy", &iam.PolicyArgs{
			Policy: pulumi.All(bucket.Bucket).ApplyT(func(args []interface{}) string {
				name := args[0].(string)
				veleroIAMPolicy, _ := iam.GetPolicyDocument(ctx, &iam.GetPolicyDocumentArgs{
					Statements: []iam.GetPolicyDocumentStatement{
						{
							Effect: pulumi.StringRef("Allow"),
							Actions: []string{
								"ec2:DescribeVolumes",
								"ec2:DescribeSnapshots",
								"ec2:CreateTags",
								"ec2:CreateVolume",
								"ec2:CreateSnapshot",
								"ec2:DeleteSnapshot",
							},
							Resources: []string{
								"*",
							},
						},
						{
							Effect: pulumi.StringRef("Allow"),
							Actions: []string{
								"s3:GetObject",
								"s3:DeleteObject",
								"s3:PutObject",
								"s3:AbortMultipartUpload",
								"s3:ListMultipartUploadParts",
							},
							Resources: []string{
								fmt.Sprintf("arn:aws:s3:::%s/*", name),
							},
						},
						{
							Effect: pulumi.StringRef("Allow"),
							Actions: []string{
								"s3:ListBucket",
							},
							Resources: []string{
								fmt.Sprintf("arn:aws:s3:::%s", name),
							},
						},
					},
				})
				return veleroIAMPolicy.Json
			}).(pulumi.StringOutput),
		})
		if err != nil {
			return err
		}
		_, err = iam.NewRolePolicyAttachment(ctx, "velero-iam-role-attachment", &iam.RolePolicyAttachmentArgs{
			PolicyArn: veleroIAMPolicy.Arn,
			Role:      saRole.Name,
		})

		_, err = helm.NewRelease(ctx, "velero", &helm.ReleaseArgs{
			Chart:   pulumi.String("velero"),
			Version: pulumi.String("5.0.2"),
			RepositoryOpts: helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://vmware-tanzu.github.io/helm-charts"),
			},
			Namespace:       pulumi.String(veleroNamespace),
			CreateNamespace: pulumi.Bool(true),
			Values: pulumi.Map{
				"serviceAccount": pulumi.Map{
					"server": pulumi.Map{
						"name": pulumi.String("velero"),
						"annotations": pulumi.StringMap{
							"eks.amazonaws.com/role-arn": saRole.Arn,
						},
					},
				},
				"credentials": pulumi.Map{
					"useSecret": pulumi.Bool(false),
				},
				"podSecurityContext": pulumi.Map{
					"fsGroup": pulumi.Int(65534),
				},
				"initContainers": pulumi.Array{
					pulumi.Map{
						"name":            pulumi.String("velero-plugin-for-aws"),
						"image":           pulumi.String("velero/velero-plugin-for-aws:v1.7.1"),
						"imagePullPolicy": pulumi.String("IfNotPresent"),
						"volumeMounts": pulumi.Array{
							pulumi.Map{
								"name":      pulumi.String("plugins"),
								"mountPath": pulumi.String("/target"),
							},
						},
					},
				},
				"configuration": pulumi.Map{
					"backupStorageLocation": pulumi.Array{
						pulumi.Map{
							"name":     pulumi.String("velero-k8s"),
							"provider": pulumi.String("aws"),
							"bucket":   bucket.Bucket,
							"prefix":   pulumi.String("velero"),
							"config": pulumi.Map{
								"region": pulumi.String("eu-central-1"),
							},
							"default": pulumi.Bool(true),
						},
					},
					"volumeSnapshotLocation": pulumi.Array{
						pulumi.Map{
							"name":     pulumi.String("velero-k8s-snapshots"),
							"provider": pulumi.String("aws"),
							"config": pulumi.Map{
								"region": pulumi.String("eu-central-1"),
							},
						},
					},
				},
			},
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}
		return nil
	})
}
