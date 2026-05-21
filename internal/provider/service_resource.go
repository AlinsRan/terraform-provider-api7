package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/api7/terraform-provider-api7/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithImportState = &ServiceResource{}

type ServiceResource struct {
	client         *client.ClientWithResponses
	gatewayGroupID string
}

type UpstreamNodeModel struct {
	Host   types.String `tfsdk:"host"`
	Port   types.Int64  `tfsdk:"port"`
	Weight types.Int64  `tfsdk:"weight"`
}

type UpstreamModel struct {
	Nodes  []UpstreamNodeModel `tfsdk:"nodes"`
	Scheme types.String        `tfsdk:"scheme"`
	Type   types.String        `tfsdk:"type"`
}

type ServiceResourceModel struct {
	ID       types.String         `tfsdk:"id"`
	Name     types.String         `tfsdk:"name"`
	Desc     types.String         `tfsdk:"desc"`
	Upstream *UpstreamModel       `tfsdk:"upstream"`
	Plugins  jsontypes.Normalized `tfsdk:"plugins"`
}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

func (r *ServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an API7 Service (published service).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Service ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Service name.",
			},
			"desc": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Description.",
			},
			"upstream": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Upstream configuration.",
				Attributes: map[string]schema.Attribute{
					"nodes": schema.ListNestedAttribute{
						Required:    true,
						Description: "Backend nodes.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"host":   schema.StringAttribute{Required: true},
								"port":   schema.Int64Attribute{Required: true},
								"weight": schema.Int64Attribute{Required: true},
							},
						},
					},
					"scheme": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Protocol: http, https, grpc, grpcs.",
					},
					"type": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Load balancing algorithm: roundrobin, chash, least_conn, ewma.",
					},
				},
			},
			"plugins": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Plugin configuration as JSON string.",
				CustomType:  jsontypes.NormalizedType{},
			},
		},
	}
}

func (r *ServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bodyBytes, err := buildServiceRequestBody(plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build request body", err.Error())
		return
	}

	params := &client.CreatePublishedServiceParams{GatewayGroupId: r.gatewayGroupID}
	apiResp, err := r.client.CreatePublishedServiceWithBodyWithResponse(ctx, params, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create service", err.Error())
		return
	}
	if apiResp.StatusCode() != 200 {
		resp.Diagnostics.AddError("API error creating service",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
		return
	}

	state := buildServiceModelFromResponse(apiResp.JSON200.Value, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := state.ID.ValueString()
	params := &client.GetPublishedServiceParams{GatewayGroupId: r.gatewayGroupID}

	apiResp, err := r.client.GetPublishedServiceWithResponse(ctx, serviceID, params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read service", err.Error())
		return
	}
	if apiResp.StatusCode() == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != 200 {
		resp.Diagnostics.AddError("API error reading service",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
		return
	}

	// The API response does not include upstream; preserve it from current state.
	newState := buildServiceModelFromResponse(apiResp.JSON200.Value, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bodyBytes, err := buildServiceRequestBody(plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build request body", err.Error())
		return
	}

	serviceID := plan.ID.ValueString()
	params := &client.PutPublishedServiceParams{GatewayGroupId: r.gatewayGroupID}
	apiResp, err := r.client.PutPublishedServiceWithBodyWithResponse(ctx, serviceID, params, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update service", err.Error())
		return
	}
	if apiResp.StatusCode() != 200 {
		resp.Diagnostics.AddError("API error updating service",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
		return
	}

	newState := buildServiceModelFromResponse(apiResp.JSON200.Value, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := state.ID.ValueString()
	params := &client.DeletePublishedServiceParams{GatewayGroupId: r.gatewayGroupID}

	apiResp, err := r.client.DeletePublishedServiceWithResponse(ctx, serviceID, params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete service", err.Error())
		return
	}
	if apiResp.StatusCode() != 200 && apiResp.StatusCode() != 204 {
		resp.Diagnostics.AddError("API error deleting service",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
	}
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// buildServiceRequestBody serializes the service model to JSON for create/update.
// We use map[string]interface{} to avoid dealing with the unexported union field
// in the generated CreatePublishedServiceJSONBody type.
func buildServiceRequestBody(m ServiceResourceModel) ([]byte, error) {
	nodes := make([]map[string]interface{}, 0, len(m.Upstream.Nodes))
	for _, n := range m.Upstream.Nodes {
		nodes = append(nodes, map[string]interface{}{
			"host":   n.Host.ValueString(),
			"port":   int(n.Port.ValueInt64()),
			"weight": int(n.Weight.ValueInt64()),
		})
	}

	upstream := map[string]interface{}{
		"nodes": nodes,
	}
	if !m.Upstream.Scheme.IsNull() && !m.Upstream.Scheme.IsUnknown() {
		upstream["scheme"] = m.Upstream.Scheme.ValueString()
	}
	if !m.Upstream.Type.IsNull() && !m.Upstream.Type.IsUnknown() {
		upstream["type"] = m.Upstream.Type.ValueString()
	}

	body := map[string]interface{}{
		"name":     m.Name.ValueString(),
		"type":     "http",
		"upstream": upstream,
	}
	if !m.Desc.IsNull() && !m.Desc.IsUnknown() {
		body["desc"] = m.Desc.ValueString()
	}
	if !m.Plugins.IsNull() && !m.Plugins.IsUnknown() {
		var plugins interface{}
		_ = json.Unmarshal([]byte(m.Plugins.ValueString()), &plugins)
		body["plugins"] = plugins
	}

	return json.Marshal(body)
}

// buildServiceModelFromResponse builds the Terraform state from the API response value.
// Since the API response does not include upstream, we preserve it from prevModel.
func buildServiceModelFromResponse(val interface { /* anonymous struct */
}, prevModel *ServiceResourceModel) ServiceResourceModel {
	// The val is an anonymous struct from the generated code; use type assertion on JSON re-encoding.
	// Re-encode and decode to a plain map for field access.
	raw, _ := json.Marshal(val)
	var m map[string]interface{}
	_ = json.Unmarshal(raw, &m)

	state := ServiceResourceModel{
		Upstream: prevModel.Upstream,
		Plugins:  jsontypes.NewNormalizedNull(),
	}

	if id, ok := m["id"].(string); ok {
		state.ID = types.StringValue(id)
	}
	if name, ok := m["name"].(string); ok {
		state.Name = types.StringValue(name)
	}
	if desc, ok := m["desc"].(string); ok {
		state.Desc = types.StringValue(desc)
	} else {
		state.Desc = types.StringNull()
	}

	return state
}
