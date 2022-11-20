package mastodon

import (
	"fmt"
	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/pulumi/pulumi-docker/sdk/v3/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type MastodonStack struct {
	pulumi.ResourceState

	DefaultNetworkName pulumi.StringOutput `pulumi:"defaultNetworkName"`
}

type MastodonStackArgs struct {
	DockerHost  pulumi.StringInput `pulumi:"dockerHost"`
	LocalDomain pulumi.StringInput `pulumi:"localDomain"`
}

func NewMastodonStack(ctx *pulumi.Context, name string, args *MastodonStackArgs, opts ...pulumi.ResourceOption) (*MastodonStack, error) {
	mastodonComponent := &MastodonStack{}
	err := ctx.RegisterComponentResource("pulumi:component:MastodonStack", name, mastodonComponent, opts...)
	if err != nil {
		return nil, err
	}

	provider, err := docker.NewProvider(ctx, "docker", &docker.ProviderArgs{
		Host: args.DockerHost,
	})
	if err != nil {
		return nil, err
	}

	otpSecret, err := local.NewCommand(ctx, "otp-secret", &local.CommandArgs{
		Create: pulumi.String("openssl rand -hex 64"),
	}, pulumi.Parent(mastodonComponent))
	if err != nil {
		return nil, err
	}

	secretKeyBase, err := local.NewCommand(ctx, "secret-key-base", &local.CommandArgs{
		Create: pulumi.String("openssl rand -hex 64"),
	}, pulumi.Parent(mastodonComponent))
	if err != nil {
		return nil, err
	}

	vapidPrivateKey, err := local.NewCommand(ctx, "vapid-private-key", &local.CommandArgs{
		Create: pulumi.String(`openssl ecparam -name prime256v1 -genkey -noout -out vapid_private_key.pem &&
		openssl ec -in vapid_private_key.pem -pubout -noout  -out vapid_public_key.pem &>/dev/null &&
		cat vapid_private_key.pem | sed -e "1 d" -e "$ d" | tr -d "\n"; echo`),
		Delete: pulumi.String("rm vapid_private_key.pem"),
	}, pulumi.Parent(mastodonComponent))
	if err != nil {
		return nil, err
	}

	vapidPublicKey, err := local.NewCommand(ctx, "vapid-public-key",
		&local.CommandArgs{
			Create: pulumi.String(`cat vapid_public_key.pem | sed -e "1 d" -e "$ d" | tr -d "\n"; echo`),
			Delete: pulumi.String("rm vapid_public_key.pem"),
		},
		pulumi.DependsOn([]pulumi.Resource{vapidPrivateKey}),
		pulumi.Parent(mastodonComponent))
	if err != nil {
		return nil, err
	}
	var envVars = pulumi.StringArray{
		pulumi.Sprintf("LOCAL_DOMAIN=%s", args.LocalDomain),
		pulumi.String("SINGLE_USER_MODE=false"),
		pulumi.Sprintf("VAPID_PRIVATE_KEY=%s", vapidPrivateKey.Stdout),
		pulumi.Sprintf("VAPID_PUBLIC_KEY=%s", vapidPublicKey.Stdout),
		pulumi.Sprintf("OTP_SECRET=%s", otpSecret.Stdout),
		pulumi.Sprintf("SECRET_KEY_BASE=%s", secretKeyBase.Stdout),
		pulumi.String("DB_PORT=5432"),
		pulumi.String("DB_NAME=postgres"),
		pulumi.String("DB_USER=postgres"),
		pulumi.String("DB_PASS="),
		pulumi.String("REDIS_PORT=6379"),
		pulumi.String("REDIS_PASS="),
	}

	mastodonNetwork, err := docker.NewNetwork(ctx, "mastodonNetwork",
		&docker.NetworkArgs{
			Internal: pulumi.Bool(true),
		},
		pulumi.Provider(provider),
		pulumi.Parent(mastodonComponent),
	)
	if err != nil {
		return nil, err
	}
	externalMastodonNetwork, err := docker.NewNetwork(ctx, "externalMastodonNetwork",
		&docker.NetworkArgs{
			Internal: pulumi.Bool(false),
		},
		pulumi.Provider(provider),
		pulumi.Parent(mastodonComponent),
	)
	if err != nil {
		return nil, err
	}

	postgresRemoteImage, err := docker.NewRemoteImage(ctx, fmt.Sprintf("%s-postgres-image", name),
		&docker.RemoteImageArgs{
			Name: pulumi.String("postgres:14-alpine"),
		},
		pulumi.Provider(provider),
		pulumi.Parent(mastodonComponent))
	if err != nil {
		return nil, err
	}

	postgresVolume, err := docker.NewVolume(ctx, fmt.Sprintf("%s-postgres-volume", name),
		&docker.VolumeArgs{},
		pulumi.Provider(provider),
		pulumi.Parent(mastodonComponent))
	if err != nil {
		return nil, err
	}

	postgres, err := docker.NewContainer(ctx, fmt.Sprintf("%s-postgres-container", name),
		&docker.ContainerArgs{
			Image:   postgresRemoteImage.ImageId,
			Restart: pulumi.String("unless-stopped"),
			ShmSize: pulumi.Int(256),
			NetworksAdvanced: docker.ContainerNetworksAdvancedArray{
				&docker.ContainerNetworksAdvancedArgs{
					Name: mastodonNetwork.Name,
				},
			},
			Healthcheck: &docker.ContainerHealthcheckArgs{
				Tests: pulumi.StringArray{
					pulumi.String("CMD"),
					pulumi.String("pg_isready"),
					pulumi.String("-U"),
					pulumi.String("postgres"),
				},
			},
			Volumes: docker.ContainerVolumeArray{
				&docker.ContainerVolumeArgs{
					VolumeName:    postgresVolume.Name,
					ContainerPath: pulumi.String("/var/lib/postgresql/data"),
				},
			},
			Envs: pulumi.StringArray{
				pulumi.String("POSTGRES_HOST_AUTH_METHOD=trust"),
			},
		},
		pulumi.Provider(provider),
		pulumi.Parent(mastodonComponent))
	if err != nil {
		return nil, err
	}

	redisRemoteImage, err := docker.NewRemoteImage(ctx, fmt.Sprintf("%s-redis-image", name),
		&docker.RemoteImageArgs{
			Name: pulumi.String("redis:7-alpine"),
		},
		pulumi.Provider(provider),
		pulumi.Parent(mastodonComponent),
	)
	if err != nil {
		return nil, err
	}
	redisVolume, err := docker.NewVolume(ctx, fmt.Sprintf("%s-redis-volume", name),
		&docker.VolumeArgs{},
		pulumi.Provider(provider),
		pulumi.Parent(mastodonComponent),
	)
	if err != nil {
		return nil, err
	}
	redis, err := docker.NewContainer(ctx, fmt.Sprintf("%s-redis-container", name),
		&docker.ContainerArgs{
			Image:   redisRemoteImage.ImageId,
			Restart: pulumi.String("unless-stopped"),
			NetworksAdvanced: docker.ContainerNetworksAdvancedArray{
				&docker.ContainerNetworksAdvancedArgs{
					Name: mastodonNetwork.Name,
				},
			},
			Healthcheck: &docker.ContainerHealthcheckArgs{
				Tests: pulumi.StringArray{
					pulumi.String("CMD"),
					pulumi.String("redis-cli"),
					pulumi.String("ping"),
				},
			},
			Volumes: docker.ContainerVolumeArray{
				&docker.ContainerVolumeArgs{
					VolumeName:    redisVolume.Name,
					ContainerPath: pulumi.String("/data"),
				},
			},
		},
		pulumi.Provider(provider),
		pulumi.Parent(mastodonComponent),
	)
	if err != nil {
		return nil, err
	}

	mastodonImage, err := docker.NewRemoteImage(ctx, fmt.Sprintf("%s-mastodon-image", name),
		&docker.RemoteImageArgs{
			Name: pulumi.String("tootsuite/mastodon:v3.5.3"),
		},
		pulumi.Provider(provider),
		pulumi.Parent(mastodonComponent),
	)
	if err != nil {
		return nil, err
	}
	mastodonVolume, err := docker.NewVolume(ctx, fmt.Sprintf("%s-mastodon-volume", name),
		&docker.VolumeArgs{},
		pulumi.Provider(provider),
		pulumi.Parent(mastodonComponent),
	)
	if err != nil {
		return nil, err
	}

	envVars = append(envVars, pulumi.Sprintf("DB_HOST=%s", postgres.Name))
	envVars = append(envVars, pulumi.Sprintf("REDIS_HOST=%s", redis.Name))

	_, err = docker.NewContainer(ctx, fmt.Sprintf("%s-mastodon-container", name),
		&docker.ContainerArgs{
			Image:   mastodonImage.ImageId,
			Restart: pulumi.String("unless-stopped"),
			Envs:    envVars,
			Ports: docker.ContainerPortArray{
				&docker.ContainerPortArgs{
					Internal: pulumi.Int(3000),
					External: pulumi.Int(3000),
				},
			},
			Command: pulumi.StringArray{
				pulumi.String("bash"),
				pulumi.String("-c"),
				pulumi.String("rm -f /mastodon/tmp/pids/server.pid; bundle exec rails s -p 3000"),
			},
			Healthcheck: &docker.ContainerHealthcheckArgs{
				Tests: pulumi.StringArray{
					pulumi.String("CMD-SHELL"),
					pulumi.String("wget -q --spider --proxy=off localhost:3000/health || exit 1"),
				},
			},
			NetworksAdvanced: docker.ContainerNetworksAdvancedArray{
				&docker.ContainerNetworksAdvancedArgs{
					Name: mastodonNetwork.Name,
				},
				&docker.ContainerNetworksAdvancedArgs{
					Name: externalMastodonNetwork.Name,
					Aliases: pulumi.StringArray{
						pulumi.String("mastodon"),
					},
				},
			},
			Volumes: docker.ContainerVolumeArray{
				&docker.ContainerVolumeArgs{
					VolumeName:    mastodonVolume.Name,
					ContainerPath: pulumi.String("/mastodon/public/system"),
				},
			},
		},
		pulumi.Provider(provider),
		pulumi.DependsOn([]pulumi.Resource{postgres, redis}),
		pulumi.Parent(mastodonComponent),
	)
	if err != nil {
		return nil, err
	}

	_, err = docker.NewContainer(ctx, fmt.Sprintf("%s-streaming-container", name),
		&docker.ContainerArgs{
			Image:   mastodonImage.ImageId,
			Restart: pulumi.String("unless-stopped"),
			Envs:    envVars,
			Command: pulumi.StringArray{
				pulumi.String("node"),
				pulumi.String("./streaming"),
			},
			Ports: docker.ContainerPortArray{
				&docker.ContainerPortArgs{
					Internal: pulumi.Int(4000),
					External: pulumi.Int(4000),
				},
			},
			NetworksAdvanced: docker.ContainerNetworksAdvancedArray{
				&docker.ContainerNetworksAdvancedArgs{
					Name: mastodonNetwork.Name,
				},
				&docker.ContainerNetworksAdvancedArgs{
					Name: externalMastodonNetwork.Name,
					Aliases: pulumi.StringArray{
						pulumi.String("mastodon-streaming"),
					},
				},
			},
			Healthcheck: &docker.ContainerHealthcheckArgs{
				Tests: pulumi.StringArray{
					pulumi.String("CMD-SHELL"),
					pulumi.String("wget -q --spider --proxy=off localhost:4000/api/v1/streaming/health || exit 1"),
				},
			},
		},
		pulumi.Provider(provider),
		pulumi.DependsOn([]pulumi.Resource{postgres, redis}),
		pulumi.Parent(mastodonComponent),
	)
	if err != nil {
		return nil, err
	}

	_, err = docker.NewContainer(ctx, fmt.Sprintf("%s-sidekiq-container", name),
		&docker.ContainerArgs{
			Image:   mastodonImage.ImageId,
			Restart: pulumi.String("unless-stopped"),
			Envs:    envVars,
			Command: pulumi.StringArray{
				pulumi.String("bundle"),
				pulumi.String("exec"),
				pulumi.String("sidekiq"),
			},
			NetworksAdvanced: docker.ContainerNetworksAdvancedArray{
				&docker.ContainerNetworksAdvancedArgs{
					Name: mastodonNetwork.Name,
				},
				&docker.ContainerNetworksAdvancedArgs{
					Name: externalMastodonNetwork.Name,
				},
			},
			Volumes: docker.ContainerVolumeArray{
				&docker.ContainerVolumeArgs{
					VolumeName:    mastodonVolume.Name,
					ContainerPath: pulumi.String("/mastodon/public/system"),
				},
			},
			Healthcheck: &docker.ContainerHealthcheckArgs{
				Tests: pulumi.StringArray{
					pulumi.String("CMD-SHELL"),
					pulumi.String("ps aux | grep '[s]idekiq\\ 6' || false"),
				},
			},
		},
		pulumi.Provider(provider),
		pulumi.DependsOn([]pulumi.Resource{postgres, redis}),
		pulumi.Parent(mastodonComponent),
	)
	if err != nil {
		return nil, err
	}

	_, err = docker.NewContainer(ctx, fmt.Sprintf("%s-shell", name),
		&docker.ContainerArgs{
			Image:   mastodonImage.ImageId,
			Restart: pulumi.String("no"),
			Envs:    envVars,
			Command: pulumi.StringArray{
				pulumi.String("/bin/bash"),
				pulumi.String("-c"),
				pulumi.String("RAILS_ENV=production rails db:setup && while true; do sleep 1; done"),
			},
			NetworksAdvanced: docker.ContainerNetworksAdvancedArray{
				&docker.ContainerNetworksAdvancedArgs{
					Name: mastodonNetwork.Name,
				},
				&docker.ContainerNetworksAdvancedArgs{
					Name: externalMastodonNetwork.Name,
				},
			},
			Volumes: docker.ContainerVolumeArray{
				&docker.ContainerVolumeArgs{
					VolumeName:    mastodonVolume.Name,
					ContainerPath: pulumi.String("/mastodon/public/system"),
				},
			},
		},
		pulumi.Provider(provider),
		pulumi.DependsOn([]pulumi.Resource{postgres, redis}),
		pulumi.Parent(mastodonComponent),
	)

	if err != nil {
		return nil, err
	}

	caddyImage, err := docker.NewRemoteImage(ctx, fmt.Sprintf("%s-caddy-image", name),
		&docker.RemoteImageArgs{
			Name: pulumi.String("caddy:2.6.2-alpine"),
		},
		pulumi.Provider(provider),
		pulumi.Parent(mastodonComponent),
	)
	if err != nil {
		return nil, err
	}

	_, err = docker.NewContainer(ctx, fmt.Sprintf("%s-caddy-container", name),
		&docker.ContainerArgs{
			Image:   caddyImage.ImageId,
			Restart: pulumi.String("unless-stopped"),
			Ports: docker.ContainerPortArray{
				&docker.ContainerPortArgs{
					Internal: pulumi.Int(80),
					External: pulumi.Int(80),
				},
				&docker.ContainerPortArgs{
					Internal: pulumi.Int(443),
					External: pulumi.Int(443),
				},
			},
			NetworksAdvanced: docker.ContainerNetworksAdvancedArray{
				&docker.ContainerNetworksAdvancedArgs{
					Name: mastodonNetwork.Name,
				},
				&docker.ContainerNetworksAdvancedArgs{
					Name: externalMastodonNetwork.Name,
				},
			},
			Volumes: docker.ContainerVolumeArray{
				&docker.ContainerVolumeArgs{
					ContainerPath: pulumi.String("/etc/caddy/Caddyfile"),
					HostPath:      pulumi.String("/tmp/Caddyfile"),
				},
			},
		},
		pulumi.Provider(provider),
		pulumi.Parent(mastodonComponent),
	)
	if err != nil {
		return nil, err
	}

	//mastodonComponent.DefaultNetworkName = defaultNetwork.Name
	err = ctx.RegisterResourceOutputs(mastodonComponent, pulumi.Map{
		//"defaultNetworkName": defaultNetwork.Name,
	})
	if err != nil {
		return nil, err
	}

	return mastodonComponent, nil
}
