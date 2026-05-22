package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/api7/terraform-provider-api7/internal/client"
	gen "github.com/api7/terraform-provider-api7/internal/provider/generated/resource_consumer"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ConsumerResource{}
var _ resource.ResourceWithImportState = &ConsumerResource{}

type ConsumerResource struct {
	client         *client.ClientWithResponses
	gatewayGroupID string
}

type ConsumerResourceModel struct {
	Username types.String         `tfsdk:"username"`
	Desc     types.String         `tfsdk:"desc"`
	Labels   types.Map            `tfsdk:"labels"`
	Plugins  jsontypes.Normalized `tfsdk:"plugins"`
}

func NewConsumerResource() resource.Resource {
	return &ConsumerResource{}
}

func (r *ConsumerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_consumer"
}

func (r *ConsumerResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := gen.ConsumerResourceSchema(ctx)
	s.Description = "Manages an API7 Consumer."

	// username is immutable — replacing it creates a new consumer.
	s.Attributes["username"] = schema.StringAttribute{
		Required:    true,
		Description: "Unique username. Changing this forces a new resource.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
		// Preserve validators from generated schema.
		Validators: s.Attributes["username"].(schema.StringAttribute).Validators,
	}

	// Override plugins to free-form JSON string.
	s.Attributes["plugins"] = schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		CustomType:  jsontypes.NormalizedType{},
		Description: `Plugin configuration as JSON string, e.g. jsonencode({"key-auth":{"key":"secret"}}).`,
	}

	// Remove internal / response-only fields.
	delete(s.Attributes, "gateway_group_id")
	delete(s.Attributes, "value")

	resp.Schema = s
}

func (r *ConsumerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.client = pd.Client
	r.gatewayGroupID = pd.GatewayGroupID
}

func (r *ConsumerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ConsumerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := client.CreateConsumerJSONRequestBody{Username: plan.Username.ValueString()}
	if !plan.Desc.IsNull() && !plan.Desc.IsUnknown() {
		desc := plan.Desc.ValueString()
		body.Desc = &desc
	}
	if !plan.Plugins.IsNull() && !plan.Plugins.IsUnknown() {
		var plugins map[string]interface{}
		_ = json.Unmarshal([]byte(plan.Plugins.ValueString()), &plugins)
		body.Plugins = &plugins
	}

	params := &client.CreateConsumerParams{GatewayGroupId: r.gatewayGroupID}
	apiResp, err := r.client.CreateConsumerWithResponse(ctx, params, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create consumer", err.Error())
		return
	}
	if apiResp.StatusCode() != 200 {
		resp.Diagnostics.AddError("API error creating consumer",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
		return
	}

	val := apiResp.JSON200.Value
	state := buildConsumerModel(ctx, val.Username, val.Desc, val.Plugins)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ConsumerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConsumerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &client.GetConsumerParams{GatewayGroupId: r.gatewayGroupID}
	apiResp, err := r.client.GetConsumerWithResponse(ctx, state.Username.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read consumer", err.Error())
		return
	}
	if apiResp.StatusCode() == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != 200 {
		resp.Diagnostics.AddError("API error reading consumer",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
		return
	}

	val := apiResp.JSON200.Value
	newState := buildConsumerModel(ctx, val.Username, val.Desc, val.Plugins)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ConsumerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ConsumerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := client.UpsertConsumerJSONRequestBody{Username: plan.Username.ValueString()}
	if !plan.Desc.IsNull() && !plan.Desc.IsUnknown() {
		desc := plan.Desc.ValueString()
		body.Desc = &desc
	}
	if !plan.Plugins.IsNull() && !plan.Plugins.IsUnknown() {
		var plugins map[string]interface{}
		_ = json.Unmarshal([]byte(plan.Plugins.ValueString()), &plugins)
		body.Plugins = &plugins
	}

	params := &client.UpsertConsumerParams{GatewayGroupId: r.gatewayGroupID}
	apiResp, err := r.client.UpsertConsumerWithResponse(ctx, plan.Username.ValueString(), params, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update consumer", err.Error())
		return
	}
	if apiResp.StatusCode() != 200 {
		resp.Diagnostics.AddError("API error updating consumer",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
		return
	}

	val := apiResp.JSON200.Value
	state := buildConsumerModel(ctx, val.Username, val.Desc, val.Plugins)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ConsumerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConsumerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &client.DeleteConsumerParams{GatewayGroupId: r.gatewayGroupID}
	apiResp, err := r.client.DeleteConsumerWithResponse(ctx, state.Username.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete consumer", err.Error())
		return
	}
	if apiResp.StatusCode() != 200 && apiResp.StatusCode() != 204 {
		resp.Diagnostics.AddError("API error deleting consumer",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
	}
}

func (r *ConsumerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), req.ID)...)
}

func buildConsumerModel(ctx context.Context, username string, desc *string, plugins *map[string]interface{}) ConsumerResourceModel {
	m := ConsumerResourceModel{
		Username: types.StringValue(username),
		Labels:   types.MapValueMust(types.StringType, map[string]attr.Value{}),
		Plugins:  jsontypes.NewNormalizedNull(),
	}
	if desc != nil {
		m.Desc = types.StringValue(*desc)
	} else {
		m.Desc = types.StringNull()
	}
	if plugins != nil && len(*plugins) > 0 {
		b, _ := json.Marshal(plugins)
		m.Plugins = jsontypes.NewNormalizedValue(string(b))
	}
	return m
}
