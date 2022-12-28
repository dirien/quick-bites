package main

import (
	"fmt"
	"github.com/pulumi/pulumi-azure-native-sdk/resources"
	"github.com/pulumi/pulumi-azure-native-sdk/storage"
	web "github.com/pulumi/pulumi-azure-native-sdk/web/v20220301"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"os"
)

const (
	name         = "azure-logics-apps"
	contentshare = "engindirisa-xxx"
)

// uploadWorkflows creates a AzureCliScript resource that uploads the workflow files to the storage account.
func uploadWorkflows(ctx *pulumi.Context, name string, resourceGroup *resources.ResourceGroup, account *storage.StorageAccount, key pulumi.Output, dependsOn pulumi.ResourceOption) error {
	file, err := os.ReadFile(fmt.Sprintf("%s/workflow.json", name))
	if err != nil {
		return err
	}

	_, err = resources.NewAzureCliScript(ctx, fmt.Sprintf("%s-upload-workflow", name), &resources.AzureCliScriptArgs{
		Location:          resourceGroup.Location,
		ResourceGroupName: resourceGroup.Name,
		RetentionInterval: pulumi.String("P1D"),
		AzCliVersion:      pulumi.String("2.41.0"),
		Kind:              pulumi.String("AzureCLI"),
		ForceUpdateTag:    pulumi.String("1"),
		EnvironmentVariables: resources.EnvironmentVariableArray{
			&resources.EnvironmentVariableArgs{
				Name:  pulumi.String("AZURE_STORAGE_ACCOUNT"),
				Value: account.Name,
			},
			&resources.EnvironmentVariableArgs{
				Name:        pulumi.String("AZURE_STORAGE_KEY"),
				SecureValue: pulumi.Sprintf("%s", key),
			},
			&resources.EnvironmentVariableArgs{
				Name:  pulumi.String("SHARE_NAME"),
				Value: pulumi.String(contentshare),
			},
			&resources.EnvironmentVariableArgs{
				Name:  pulumi.String("CONTENT"),
				Value: pulumi.Sprintf("%s", string(file)),
			},
		},
		ScriptContent: pulumi.Sprintf(`echo $CONTENT > workflow.json  && \
az storage directory create --share-name $SHARE_NAME --name site/wwwroot/%s && \
az storage file upload --source workflow.json --path site/wwwroot/%s/workflow.json --share-name $SHARE_NAME`, name, name),
	}, dependsOn)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		resourceGroup, err := resources.NewResourceGroup(ctx, fmt.Sprintf("%s-rg", name), &resources.ResourceGroupArgs{})
		if err != nil {
			return err
		}

		account, err := storage.NewStorageAccount(ctx, fmt.Sprintf("%s-sa", name), &storage.StorageAccountArgs{
			ResourceGroupName: resourceGroup.Name,
			AccountName:       pulumi.String("engindirisa"),
			Sku: &storage.SkuArgs{
				Name: pulumi.String("Standard_LRS"),
			},
			Kind: pulumi.String("StorageV2"),
		})
		if err != nil {
			return err
		}

		key := pulumi.All(account.Name, resourceGroup.Name).ApplyT(func(args []interface{}) string {
			storageKey, _ := storage.ListStorageAccountKeys(ctx, &storage.ListStorageAccountKeysArgs{
				AccountName:       args[0].(string),
				ResourceGroupName: args[1].(string),
			})
			return storageKey.Keys[0].Value
		})

		appServicePlan, err := web.NewAppServicePlan(ctx, "plan", &web.AppServicePlanArgs{
			ResourceGroupName: resourceGroup.Name,
			Location:          resourceGroup.Location,
			Sku: &web.SkuDescriptionArgs{
				Name: pulumi.String("WS1"),
				Tier: pulumi.String("WorkflowStandard"),
			},
			TargetWorkerCount:         pulumi.Int(1),
			MaximumElasticWorkerCount: pulumi.Int(5),
			ElasticScaleEnabled:       pulumi.Bool(true),
			ZoneRedundant:             pulumi.Bool(true),
			IsSpot:                    pulumi.Bool(false),
		})
		if err != nil {
			return err
		}

		logicApp, err := web.NewWebApp(ctx, fmt.Sprintf("%s-wa", name), &web.WebAppArgs{
			Kind:              pulumi.String("functionapp,workflowapp"),
			ResourceGroupName: resourceGroup.Name,
			Location:          resourceGroup.Location,
			Identity: &web.ManagedServiceIdentityArgs{
				Type: web.ManagedServiceIdentityTypeSystemAssigned,
			},
			HttpsOnly: pulumi.Bool(true),
			Enabled:   pulumi.Bool(true),
			SiteConfig: &web.SiteConfigArgs{
				AppSettings: web.NameValuePairArray{
					&web.NameValuePairArgs{
						Name:  pulumi.String("FUNCTIONS_EXTENSION_VERSION"),
						Value: pulumi.String("~4"),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("FUNCTIONS_WORKER_RUNTIME"),
						Value: pulumi.String("node"),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("WEBSITE_NODE_DEFAULT_VERSION"),
						Value: pulumi.String("~14"),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("AzureWebJobsStorage"),
						Value: pulumi.Sprintf("DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;EndpointSuffix=core.windows.net", account.Name, key),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("WEBSITE_CONTENTAZUREFILECONNECTIONSTRING"),
						Value: pulumi.Sprintf("DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;EndpointSuffix=core.windows.net", account.Name, key),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("WEBSITE_CONTENTSHARE"),
						Value: pulumi.String(contentshare),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("AzureFunctionsJobHost__extensionBundle__id"),
						Value: pulumi.String("Microsoft.Azure.Functions.ExtensionBundle.Workflows"),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("AzureFunctionsJobHost__extensionBundle__version"),
						Value: pulumi.String("[1.*, 2.0.0)"),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("APP_KIND"),
						Value: pulumi.String("workflowapp"),
					},
				},
				Use32BitWorkerProcess: pulumi.Bool(false),
			},
			ServerFarmId:          appServicePlan.ID(),
			ClientAffinityEnabled: pulumi.Bool(false),
		})
		if err != nil {
			return err
		}

		err = uploadWorkflows(ctx, "flow1", resourceGroup, account, key, pulumi.DependsOn([]pulumi.Resource{appServicePlan, account, logicApp}))
		if err != nil {
			return err
		}
		err = uploadWorkflows(ctx, "flow2", resourceGroup, account, key, pulumi.DependsOn([]pulumi.Resource{appServicePlan, account, logicApp}))
		if err != nil {
			return err
		}
		return nil
	})
}
