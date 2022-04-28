package main

import (
	"encoding/base64"
	"github.com/pulumi/pulumi-oci/sdk/go/oci/core"
	"github.com/pulumi/pulumi-oci/sdk/go/oci/identity"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"io/ioutil"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		compartment, err := identity.NewCompartment(ctx, "compartment", &identity.CompartmentArgs{
			Name:        pulumi.Sprintf("%s-minecraft-compartment", ctx.Stack()),
			Description: pulumi.String("Compartment for minecraft"),
		})
		if err != nil {
			return err
		}

		vcn, err := core.NewVcn(ctx, "minecraft-vcn", &core.VcnArgs{
			CidrBlock:     pulumi.String("10.0.0.0/16"),
			DisplayName:   pulumi.Sprintf("%s-minecraft-vcn", ctx.Stack()),
			DnsLabel:      pulumi.String("vcnminecraft"),
			CompartmentId: compartment.ID(),
		})
		if err != nil {
			return err
		}

		securityList, err := core.NewSecurityList(ctx, "minecraft-security-list", &core.SecurityListArgs{
			VcnId:         vcn.ID(),
			CompartmentId: compartment.ID(),
			DisplayName:   pulumi.Sprintf("%s-minecraft-sl", ctx.Stack()),
			EgressSecurityRules: core.SecurityListEgressSecurityRuleArray{
				core.SecurityListEgressSecurityRuleArgs{
					Protocol:    pulumi.String("all"),
					Destination: pulumi.String("0.0.0.0/0"),
				},
			},
			IngressSecurityRules: core.SecurityListIngressSecurityRuleArray{
				core.SecurityListIngressSecurityRuleArgs{
					Protocol:    pulumi.String("6"),
					Source:      pulumi.String("0.0.0.0/0"),
					Description: pulumi.String("Non Standard SSH Port"),
					TcpOptions: core.SecurityListIngressSecurityRuleTcpOptionsArgs{
						Max: pulumi.Int(22),
						Min: pulumi.Int(22),
					},
				},
				core.SecurityListIngressSecurityRuleArgs{
					Protocol:    pulumi.String("6"),
					Source:      pulumi.String("0.0.0.0/0"),
					Description: pulumi.String("Minecraft Server Port"),
					TcpOptions: core.SecurityListIngressSecurityRuleTcpOptionsArgs{
						Max: pulumi.Int(25565),
						Min: pulumi.Int(25565),
					},
				},
			},
		})
		if err != nil {
			return err
		}
		subnet, err := core.NewSubnet(ctx, "minecraft-subnet", &core.SubnetArgs{
			CompartmentId: compartment.ID(),
			VcnId:         vcn.ID(),
			CidrBlock:     pulumi.String("10.0.0.0/24"),
			SecurityListIds: pulumi.StringArray{
				vcn.DefaultSecurityListId,
				securityList.ID(),
			},
			ProhibitPublicIpOnVnic: pulumi.Bool(false),
			RouteTableId:           vcn.DefaultRouteTableId,
			DhcpOptionsId:          vcn.DefaultDhcpOptionsId,
			DisplayName:            pulumi.Sprintf("%s-minecraft-subnet", ctx.Stack()),
			DnsLabel:               pulumi.String("subnetminecraft"),
		})
		if err != nil {
			return err
		}

		internetGateway, err := core.NewInternetGateway(ctx, "minecraft-internet-gateway", &core.InternetGatewayArgs{
			CompartmentId: compartment.ID(),
			VcnId:         vcn.ID(),
			DisplayName:   pulumi.Sprintf("%s-minecraft-rg", ctx.Stack()),
			Enabled:       pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		_, err = core.NewDefaultRouteTable(ctx, "minecraft-route-table", &core.DefaultRouteTableArgs{
			ManageDefaultResourceId: vcn.DefaultRouteTableId,
			CompartmentId:           compartment.ID(),
			DisplayName:             pulumi.Sprintf("%s-minecraft-rt", ctx.Stack()),
			RouteRules: core.DefaultRouteTableRouteRuleArray{
				core.DefaultRouteTableRouteRuleArgs{
					NetworkEntityId: internetGateway.ID(),
					Destination:     pulumi.String("0.0.0.0/0"),
					DestinationType: pulumi.String("CIDR_BLOCK"),
				},
			},
		})
		if err != nil {
			return err
		}

		userData, err := ioutil.ReadFile("config/cloud-init.yaml")
		if err != nil {
			return err
		}
		pubKeyFile, err := ioutil.ReadFile("ssh/oci.pub")
		if err != nil {
			return err
		}

		imageId := compartment.CompartmentId.ApplyT(func(id string) string {
			images, _ := core.GetImages(ctx, &core.GetImagesArgs{
				CompartmentId:          id,
				OperatingSystem:        pulumi.StringRef("Canonical Ubuntu"),
				OperatingSystemVersion: pulumi.StringRef("20.04"),
				SortBy:                 pulumi.StringRef("TIMECREATED"),
				SortOrder:              pulumi.StringRef("DESC"),
				Shape:                  pulumi.StringRef("VM.Standard.A1.Flex"),
			})
			return images.Images[0].Id
		}).(pulumi.StringOutput)

		availabilityDomainName := compartment.CompartmentId.ApplyT(func(id string) string {
			availabilityDomains, _ := identity.GetAvailabilityDomains(ctx, &identity.GetAvailabilityDomainsArgs{
				CompartmentId: id,
			})
			return availabilityDomains.AvailabilityDomains[0].Name
		}).(pulumi.StringOutput)

		minecraft, err := core.NewInstance(ctx, "minecraft-arm", &core.InstanceArgs{
			CompartmentId:      compartment.ID(),
			DisplayName:        pulumi.Sprintf("%s-minecraft-instance", ctx.Stack()),
			AvailabilityDomain: availabilityDomainName,
			Shape:              pulumi.String("VM.Standard.A1.Flex"),
			ShapeConfig: core.InstanceShapeConfigArgs{
				Ocpus:       pulumi.Float64(1),
				MemoryInGbs: pulumi.Float64(6),
			},
			SourceDetails: core.InstanceSourceDetailsArgs{
				SourceType: pulumi.String("image"),
				SourceId:   imageId,
			},
			CreateVnicDetails: core.InstanceCreateVnicDetailsArgs{
				AssignPublicIp: pulumi.String("true"),
				SubnetId:       subnet.ID(),
				DisplayName:    pulumi.Sprintf("%s-minecraft", ctx.Stack()),
			},
			Metadata: pulumi.Map{
				"user_data":           pulumi.String(base64.StdEncoding.EncodeToString(userData)),
				"ssh_authorized_keys": pulumi.String(pubKeyFile),
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("minecraft-ip", minecraft.PublicIp)

		return nil
	})
}
