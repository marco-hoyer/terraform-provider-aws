// Code generated by internal/generate/servicepackage/main.go; DO NOT EDIT.

package apigateway

import (
	"context"

	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/types"
	"github.com/hashicorp/terraform-provider-aws/names"
)

type servicePackage struct{}

func (p *servicePackage) FrameworkDataSources(ctx context.Context) []*types.ServicePackageFrameworkDataSource {
	return []*types.ServicePackageFrameworkDataSource{}
}

func (p *servicePackage) FrameworkResources(ctx context.Context) []*types.ServicePackageFrameworkResource {
	return []*types.ServicePackageFrameworkResource{}
}

func (p *servicePackage) SDKDataSources(ctx context.Context) []*types.ServicePackageSDKDataSource {
	return []*types.ServicePackageSDKDataSource{
		{
			Factory:  dataSourceAPIKey,
			TypeName: "aws_api_gateway_api_key",
			Name:     "API Key",
			Tags:     &types.ServicePackageResourceTags{},
		},
		{
			Factory:  dataSourceAuthorizer,
			TypeName: "aws_api_gateway_authorizer",
			Name:     "Authorizer",
		},
		{
			Factory:  dataSourceAuthorizers,
			TypeName: "aws_api_gateway_authorizers",
			Name:     "Authorizers",
		},
		{
			Factory:  dataSourceDomainName,
			TypeName: "aws_api_gateway_domain_name",
			Name:     "Domain Name",
			Tags:     &types.ServicePackageResourceTags{},
		},
		{
			Factory:  dataSourceExport,
			TypeName: "aws_api_gateway_export",
			Name:     "Export",
		},
		{
			Factory:  dataSourceResource,
			TypeName: "aws_api_gateway_resource",
			Name:     "Resource",
		},
		{
			Factory:  dataSourceRestAPI,
			TypeName: "aws_api_gateway_rest_api",
			Name:     "REST API",
			Tags:     &types.ServicePackageResourceTags{},
		},
		{
			Factory:  dataSourceSDK,
			TypeName: "aws_api_gateway_sdk",
			Name:     "SDK",
		},
		{
			Factory:  dataSourceVPCLink,
			TypeName: "aws_api_gateway_vpc_link",
			Name:     "VPC Link",
			Tags:     &types.ServicePackageResourceTags{},
		},
	}
}

func (p *servicePackage) SDKResources(ctx context.Context) []*types.ServicePackageSDKResource {
	return []*types.ServicePackageSDKResource{
		{
			Factory:  resourceAccount,
			TypeName: "aws_api_gateway_account",
			Name:     "Account",
		},
		{
			Factory:  resourceAPIKey,
			TypeName: "aws_api_gateway_api_key",
			Name:     "API Key",
			Tags: &types.ServicePackageResourceTags{
				IdentifierAttribute: names.AttrARN,
			},
		},
		{
			Factory:  resourceAuthorizer,
			TypeName: "aws_api_gateway_authorizer",
			Name:     "Authorizer",
		},
		{
			Factory:  resourceBasePathMapping,
			TypeName: "aws_api_gateway_base_path_mapping",
			Name:     "Base Path Mapping",
		},
		{
			Factory:  resourceClientCertificate,
			TypeName: "aws_api_gateway_client_certificate",
			Name:     "Client Certificate",
			Tags: &types.ServicePackageResourceTags{
				IdentifierAttribute: names.AttrARN,
			},
		},
		{
			Factory:  resourceDeployment,
			TypeName: "aws_api_gateway_deployment",
			Name:     "Deployment",
		},
		{
			Factory:  resourceDocumentationPart,
			TypeName: "aws_api_gateway_documentation_part",
			Name:     "Documentation Part",
		},
		{
			Factory:  resourceDocumentationVersion,
			TypeName: "aws_api_gateway_documentation_version",
			Name:     "Documentation Version",
		},
		{
			Factory:  resourceDomainName,
			TypeName: "aws_api_gateway_domain_name",
			Name:     "Domain Name",
			Tags: &types.ServicePackageResourceTags{
				IdentifierAttribute: names.AttrARN,
			},
		},
		{
			Factory:  resourceGatewayResponse,
			TypeName: "aws_api_gateway_gateway_response",
			Name:     "Gateway Response",
		},
		{
			Factory:  resourceIntegration,
			TypeName: "aws_api_gateway_integration",
			Name:     "Integration",
		},
		{
			Factory:  resourceIntegrationResponse,
			TypeName: "aws_api_gateway_integration_response",
			Name:     "Integration Response",
		},
		{
			Factory:  resourceMethod,
			TypeName: "aws_api_gateway_method",
			Name:     "Method",
		},
		{
			Factory:  resourceMethodResponse,
			TypeName: "aws_api_gateway_method_response",
			Name:     "Method Response",
		},
		{
			Factory:  resourceMethodSettings,
			TypeName: "aws_api_gateway_method_settings",
			Name:     "Method Settings",
		},
		{
			Factory:  resourceModel,
			TypeName: "aws_api_gateway_model",
			Name:     "Model",
		},
		{
			Factory:  resourceRequestValidator,
			TypeName: "aws_api_gateway_request_validator",
			Name:     "Request Validator",
		},
		{
			Factory:  resourceResource,
			TypeName: "aws_api_gateway_resource",
			Name:     "Resource",
		},
		{
			Factory:  resourceRestAPI,
			TypeName: "aws_api_gateway_rest_api",
			Name:     "REST API",
			Tags: &types.ServicePackageResourceTags{
				IdentifierAttribute: names.AttrARN,
			},
		},
		{
			Factory:  resourceRestAPIPolicy,
			TypeName: "aws_api_gateway_rest_api_policy",
			Name:     "REST API Policy",
		},
		{
			Factory:  resourceStage,
			TypeName: "aws_api_gateway_stage",
			Name:     "Stage",
			Tags: &types.ServicePackageResourceTags{
				IdentifierAttribute: names.AttrARN,
			},
		},
		{
			Factory:  resourceUsagePlan,
			TypeName: "aws_api_gateway_usage_plan",
			Name:     "Usage Plan",
			Tags: &types.ServicePackageResourceTags{
				IdentifierAttribute: names.AttrARN,
			},
		},
		{
			Factory:  resourceUsagePlanKey,
			TypeName: "aws_api_gateway_usage_plan_key",
			Name:     "Usage Plan Key",
		},
		{
			Factory:  resourceVPCLink,
			TypeName: "aws_api_gateway_vpc_link",
			Name:     "VPC Link",
			Tags: &types.ServicePackageResourceTags{
				IdentifierAttribute: names.AttrARN,
			},
		},
	}
}

func (p *servicePackage) ServicePackageName() string {
	return names.APIGateway
}

func ServicePackage(ctx context.Context) conns.ServicePackage {
	return &servicePackage{}
}
