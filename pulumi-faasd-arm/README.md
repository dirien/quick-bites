# Running faasd on Azure Arm-based Virtual Machines

Since April 2022, Azure offers virtual machines with Ampere Altra Arm-based processors. To try them out, you need to
request access to the preview by filling out [this form](https://aka.ms/arm64vmspreview).

I applied, and got accepted for the preview. Since then, I play with different use-cases to discover the potential from
Arm-based virtual machines.

While thinking about all the possible use-cases, instantly OpenFaaS come into my mind. Because with `faasd`, the little
brother of OpenFaaS, we have an option to run OpenFaaS just like on Kubernetes, the same API, same UI, CLI and
ecosystem, but without the complexity and stress of running Kubernetes.

You can use `faasd` to deploy containers, which conform to the OpenFaaS serverless workload definition. Or you can:

- Deploy a microservice
- Get a HTTP API for any binary or CLI through the use of the Classic Watchdog
- Deploy a function written in any language from our templates or the template store

Question would be: What sort of use-cases might work well for `faasd`?
`faasd` works well for the same kinds of use-cases as OpenFaaS on Kubernetes, but is much simpler to manage.

Some ideas out of my head:

- Static site built with Hugo, Jekyll, etc
- An API
- A single-page app
- A bot
- Batch jobs, overnight processing via cron

So let's roll:

## Infrastructure

To provision our infrastructure, we're going to use Pulumi. This time too keep thins really simple, we use the new Pulumi
YAML language support.

Pulumi YAML is a great option for smaller scale cloud infrastructure.

Like all other Pulumi languages, Pulumi YAML programs have access to all the core features of Pulumi’s
infrastructure-as-code tooling, including native providers, secrets management, stack references, Pulumi Packages, and
all the features of the Pulumi Service. Critically, Pulumi YAML programs can interoperate seamlessly with the rest of
the Pulumi ecosystem, consuming the outputs of other Pulumi programs and using Pulumi components built in existing
Pulumi languages.

For more details, I highly recommend the [official documentation](https://www.pulumi.com/blog/pulumi-yaml/)

So with the command

```bash
pulumi new azure-yaml (--force)
```

I created my Pulumi program and stack for Azure.

To connect to my virtual machine, I need to generate a SSH keypair. I do this via:

```bash
ssh-keygen -t rsa -f faasd
```

And then I can start to create my variables block in my yaml file:

```yaml
variables:
  sshPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDIBq1WoJOr81nYCdrbkmcGtdVtkshDU36IKpNMg3MBi4dk408ITluFCYykZcVCqbJWCRvwr9iOjKajtMJpErHevUdpUD/ViOyW68KgwZrjVQLfp6VpAGzbdyFcBzM1jqOjSBhdPRCJfA5jKZJPncVWDsL/c0IarI1+lYds3Mf5OARd+46evm4aPOPcSHRnIDm4ylY2Wo/Lsd+EHCt9Ya7XpB3u15uaagnI/5VM5Oy4vSoDl6tU8cONrT+ofEdCojVR79SJFDBr+GdM5dQxgz4CngQLrX+QcTAAlyvvlthCwLIH44+/orbvAgyA0Q0Jcw56sWI1M59F2adKhiJNCwx++u1GGfVKGvrFH7CjiPVTFUSAmUF+GdCwzoy9GpWBP/eXiKudi5OcbVA4Ze4Isy8gAwUAINrjbK52HPh54Euk1JvxkTYUx2zBKaw3YlSulCu7xsRpVULneiOjUWR/Sp4CQK30RtFtWA0drUlO/OtRm23rvxfsVb3Qhcw604bztBM= dirien@SIT-SMBP1766"
  adminUsername: ubuntu

  cloudConfig:
    Fn::ToBase64: |
      ...
```

The variable called `sshPublicKey` is the content of my `faasd.pub` file. The feature to read directly from a file is
already [merged](https://github.com/pulumi/pulumi-yaml/pull/217). And hopefully, with the next release of the yaml
provider, we don't need to copy and paste the content of the public key anymore.

Next big block is my `cloud-config`. But what is `cloud-init`? `Cloud-init` is a service used for customizing
Linux-based operating systems in the cloud. It allows you to customize virtual machines provided by a cloud vendor on
boot.

The service is used as an industry standard for early-stage initialization of a VM once it has been provisioned.

For more details, please head over to the [official documentation](https://cloudinit.readthedocs.io/en/latest/)

We need to use the `Fn::ToBase64` function of Pulumi YAML, as Azure is expecting the user-data of a virtual machine
base64 encoded.

The actual installation of `faasd`, is seperated into four separate pieces:

- Install containerd
- Install CNI
- Install `faasd?
- Install Caddy

The only thing we need to keep in mind is, that we need to install the Arm version of all tools.

The Caddy install is also straight forward, the only part we need to keep in mind is during the creation of
the `Caddyfile`

```yaml
    {
      acme_ca https://acme-staging-v02.api.letsencrypt.org/directory
}
      faasd-ui.ediri.online {
      reverse_proxy http://127.0.0.1:8080
      }
```

I use the staging endpoint fo Lets Encrypt, remove this whole block if you want to use the production endpoint. For the
reverse proxy part, please use your Domain.

Another detail, I would like to explain, is the usage of Spot instance here.

```yaml
...
priority: Spot
evictionPolicy: Deallocate
...
```

Setting the `priority` to `Spot` enables us another cost saving. With the `evictionPolicy` set to `Deallocate` we are
not going to lose our disk.

So that's it, the rest should be self explaining from the code.

As usual with Pulumi, we deploy the whole Stack with just calling:

```bash
pulumi up
```

Check the cli output, and wait for the deployment to be finished.

```bash
...

View Live: https://app.pulumi.com/dirien/pulumi-faasd-arm/dev/updates/16
     Type                                          Name                  Status      
 +   pulumi:pulumi:Stack                           pulumi-faasd-arm-dev  created     
 +   ├─ azure-native:resources:ResourceGroup       faasdRg               created     
 +   ├─ azure-native:network:PublicIPAddress       faasdPublicIP         created     
 +   ├─ azure-native:network:NetworkSecurityGroup  faasdSG               created     
 +   ├─ azure-native:network:VirtualNetwork        faasdVnet             created     
 +   ├─ azure-native:network:NetworkInterface      faasdNic              created     
 +   └─ azure-native:compute:VirtualMachine        faasd                 created     
 +   └─ azure-native:compute:VirtualMachine        faasd                 creating     
Outputs:
    faasdIP: "<ip>"

Resources:
    + 7 created

Duration: 1m9s
...
```

After the virtual machine is created in Azure, I could take some minutes until everything is up and running.
To get the `admin` password, you can use following command:

```bash
export PASSWORD=$(ssh -i faasd ubuntu@51.124.226.123 "sudo cat /var/lib/faasd/secrets/basic-auth-password")
```

## faasd

Now we are ready to use `faasd`, you can use either the cli or the UI to deploy your functions. In this article, I just
use the UI. As mentioned before, I am more interested in the deployment part, for testing the Arm-based virtual machines
from Azure.

But if you want, here is a quick deployment of `figlet` via the cli:

```bash
export OPENFAAS_URL=https://faasd-ui.ediri.online
echo $PASSWORD | faas-cli login --password-stdin --tls-no-verify

faas-cli store deploy figlet --env write_timeout=1s --tls-no-verify

time curl -k $OPENFAAS_URL/function/figlet -d "Azure Arm Rocks"
    _                             _                   
   / \    _____   _ _ __ ___     / \   _ __ _ __ ___  
  / _ \  |_  / | | | '__/ _ \   / _ \ | '__| '_ ` _ \ 
 / ___ \  / /| |_| | | |  __/  / ___ \| |  | | | | | |
/_/   \_\/___|\__,_|_|  \___| /_/   \_\_|  |_| |_| |_|
                                                      
 ____            _        
|  _ \ ___   ___| | _____ 
| |_) / _ \ / __| |/ / __|
|  _ < (_) | (__|   <\__ \
|_| \_\___/ \___|_|\_\___/
curl -k $OPENFAAS_URL/function/figlet -d "Azure Arm Rocks"  0.02s user 0.01s system 0% cpu 2.499 total                          
```

faasd have built-in metrics that will show us the replica count and invocations:

```bash
faas-cli list --tls-no-verify
Function                        Invocations     Replicas
figlet                          3               1    
```

Let's scale our replicas to zero with the help of the API:

```bash
curl -k https://admin:$PASSWORD@faasd-ui.ediri.online/system/scale-function/figlet -d '{"serviceName":"figlet", "replicas": 0}'

faas-cli list --tls-no-verify
Function                        Invocations     Replicas
figlet                          3               0   
```

Now I am curious, how long the cold start will take us:

```bash
time curl -k $OPENFAAS_URL/function/figlet -d "Azure Arm rocks"
    _                             _                   
   / \    _____   _ _ __ ___     / \   _ __ _ __ ___  
  / _ \  |_  / | | | '__/ _ \   / _ \ | '__| '_ ` _ \ 
 / ___ \  / /| |_| | | |  __/  / ___ \| |  | | | | | |
/_/   \_\/___|\__,_|_|  \___| /_/   \_\_|  |_| |_| |_|
                                                      
                _        
 _ __ ___   ___| | _____ 
| '__/ _ \ / __| |/ / __|
| | | (_) | (__|   <\__ \
|_|  \___/ \___|_|\_\___/
                         
curl -k $OPENFAAS_URL/function/figlet -d "Azure Arm rocks"  0.02s user 0.01s system 1% cpu 2.416 total
```

That is a very good time for a "cold" start.

Now let us use the UI, to deploy the `cows function`. Open the page UI using your `$OPENFAAS_URL/ui/`. For the basic
auth challenge, use the username `admin` and password from the $PASSWORD env variable, we just used before.

## Cleanup

Type `pulumi destroy` to clean up all the cloud resources, you just created.

Always clean up your unused cloud resources: Avoid cloud waste and save money!

# Wrap Up

Azure Arm-based Virtual Machines are really awesome, they are fairly quick to create and offer a whole new bunch of
use-cases, you could before not do.

And the new VM series include general-purpose Dpsv5 and memory-optimized Epsv5 VMs, which can deliver up to 50 percent
better price-performance than comparable x86-based VMs.

I will keep exploring Arm-based virtual machines for all my projects, when using Azure and can't wait if this is GA and
everybody can use it.
