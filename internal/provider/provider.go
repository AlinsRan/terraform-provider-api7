package provider

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/api7/terraform-provider-api7/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &Api7Provider{}

type Api7Provider struct {
	version string
}

type Api7ProviderModel struct {
	Endpoint       types.String `tfsdk:"endpoint"`
	ApiKey         types.String `tfsdk:"api_key"`
	GatewayGroupID types.String `tfsdk:"gateway_group_id"`
	Insecure       types.Bool   `tfsdk:"insecure"`
}

// ProviderData is passed to resources via ConfigureRequest.ProviderData
type ProviderData struct {
	Client         *client.ClientWithResponses
	GatewayGroupID string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &Api7Provider{version: version}
	}
}

func (p *Api7Provider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "api7"
	resp.Version = p.version
}

func (p *Api7Provider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "API7 Enterprise Terraform Provider",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Required:    true,
				Description: "API7 Dashboard endpoint, e.g. https://127.0.0.1:7443",
			},
			"api_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "API7 API token (X-API-KEY header)",
			},
			"gateway_group_id": schema.StringAttribute{
				Required:    true,
				Description: "Default gateway group ID for all resources",
			},
			"insecure": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS verification (for local testing)",
			},
		},
	}
}

func (p *Api7Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config Api7ProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpClient := &http.Client{}
	if config.Insecure.ValueBool() {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	apiKey := config.ApiKey.ValueString()
	addAuth := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-API-KEY", apiKey)
		return nil
	}

	c, err := client.NewClientWithResponses(
		config.Endpoint.ValueString(),
		client.WithHTTPClient(httpClient),
		client.WithRequestEditorFn(addAuth),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API7 client", err.Error())
		return
	}

	providerData := &ProviderData{
		Client:         c,
		GatewayGroupID: config.GatewayGroupID.ValueString(),
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *Api7Provider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewConsumerResource,
		NewServiceResource,
		NewRouteResource,
	}
}

func (p *Api7Provider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
