package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/api7/terraform-provider-api7/internal/client"
	gen "github.com/api7/terraform-provider-api7/internal/provider/generated/resource_route"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &RouteResource{}
var _ resource.ResourceWithImportState = &RouteResource{}

type RouteResource struct {
	client         *client.ClientWithResponses
	gatewayGroupID string
}

// RouteResourceModel mirrors the schema fields used for CRUD mapping.
// plugins is jsontypes.Normalized (free-form JSON) instead of the generated empty SingleNested.
// timeout uses the generated TimeoutValue which has connect/send/read int64 fields.
type RouteResourceModel struct {
	Desc            types.String         `tfsdk:"desc"`
	EnableWebsocket types.Bool           `tfsdk:"enable_websocket"`
	Id              types.String         `tfsdk:"id"`
	Labels          types.Map            `tfsdk:"labels"`
	Methods         types.List           `tfsdk:"methods"`
	Name            types.String         `tfsdk:"name"`
	Paths           types.List           `tfsdk:"paths"`
	Plugins         jsontypes.Normalized `tfsdk:"plugins"`
	Priority        types.Int64          `tfsdk:"priority"`
	ServiceId       types.String         `tfsdk:"service_id"`
	Timeout         gen.TimeoutValue     `tfsdk:"timeout"`
}

func NewRouteResource() resource.Resource {
	return &RouteResource{}
}

func (r *RouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route"
}

// Schema uses the generated schema (with auto-derived validators) and overrides
// the plugins attribute to jsontypes.Normalized for free-form JSON support.
// Internal query-param fields (gateway_group_id, with_publish_info) are removed.
func (r *RouteResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := gen.RouteResourceSchema(ctx)
	s.Description = "Manages an API7 Route."

	// Override plugins: generated schema has empty SingleNested, we need free-form JSON.
	s.Attributes["plugins"] = schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		CustomType:  jsontypes.NormalizedType{},
		Description: "Plugin configuration as JSON string, e.g. jsonencode({\"key-auth\"={}}).",
	}

	// id should not be set by the user; use state for unknown to avoid needless diffs.
	s.Attributes["id"] = schema.StringAttribute{
		Computed:    true,
		Description: "Route ID (assigned by API7).",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}

	// Remove internal query-param fields that are not resource attributes.
	delete(s.Attributes, "gateway_group_id")
	delete(s.Attributes, "with_publish_info")

	resp.Schema = s
}

func (r *RouteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := routeModelToCreateRequest(ctx, plan)
	params := &client.CreatePublishedServiceRouteParams{GatewayGroupId: r.gatewayGroupID}

	apiResp, err := r.client.CreatePublishedServiceRouteWithResponse(ctx, params, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create route", err.Error())
		return
	}
	if apiResp.StatusCode() != 200 {
		resp.Diagnostics.AddError("API error creating route",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
		return
	}

	state, diags := buildRouteModelFromResponse(ctx, apiResp.JSON200.Value)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &client.GetPublishedServiceRouteParams{GatewayGroupId: r.gatewayGroupID}
	apiResp, err := r.client.GetPublishedServiceRouteWithResponse(ctx, state.Id.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read route", err.Error())
		return
	}
	if apiResp.StatusCode() == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != 200 {
		resp.Diagnostics.AddError("API error reading route",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
		return
	}

	newState, diags := buildRouteModelFromResponse(ctx, apiResp.JSON200.Value)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *RouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := routeModelToPutRequest(ctx, plan)
	params := &client.PutPublishedServiceRouteParams{GatewayGroupId: r.gatewayGroupID}

	apiResp, err := r.client.PutPublishedServiceRouteWithResponse(ctx, plan.Id.ValueString(), params, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update route", err.Error())
		return
	}
	if apiResp.StatusCode() != 200 {
		resp.Diagnostics.AddError("API error updating route",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
		return
	}

	newState, diags := buildRouteModelFromResponse(ctx, apiResp.JSON200.Value)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *RouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &client.DeletePublishedServiceRouteParams{GatewayGroupId: r.gatewayGroupID}
	apiResp, err := r.client.DeletePublishedServiceRouteWithResponse(ctx, state.Id.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete route", err.Error())
		return
	}
	if apiResp.StatusCode() != 200 && apiResp.StatusCode() != 204 {
		resp.Diagnostics.AddError("API error deleting route",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
	}
}

func (r *RouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// --- helpers ---

func routeModelToCreateRequest(ctx context.Context, m RouteResourceModel) client.CreatePublishedServiceRouteJSONRequestBody {
	body := client.CreatePublishedServiceRouteJSONRequestBody{
		ServiceId: m.ServiceId.ValueString(),
	}
	name := m.Name.ValueString()
	body.Name = &name

	var paths []string
	m.Paths.ElementsAs(ctx, &paths, false)
	body.Paths = &paths

	if !m.Methods.IsNull() && !m.Methods.IsUnknown() {
		var methodStrs []string
		m.Methods.ElementsAs(ctx, &methodStrs, false)
		methods := make([]client.CreatePublishedServiceRouteJSONBodyMethods, len(methodStrs))
		for i, s := range methodStrs {
			methods[i] = client.CreatePublishedServiceRouteJSONBodyMethods(s)
		}
		body.Methods = &methods
	}

	if !m.Plugins.IsNull() && !m.Plugins.IsUnknown() {
		var plugins map[string]interface{}
		_ = json.Unmarshal([]byte(m.Plugins.ValueString()), &plugins)
		body.Plugins = &plugins
	}

	if !m.Labels.IsNull() && !m.Labels.IsUnknown() {
		var labels map[string]string
		m.Labels.ElementsAs(ctx, &labels, false)
		body.Labels = &labels
	}

	return body
}

func routeModelToPutRequest(ctx context.Context, m RouteResourceModel) client.PutPublishedServiceRouteJSONRequestBody {
	body := client.PutPublishedServiceRouteJSONRequestBody{
		ServiceId: m.ServiceId.ValueString(),
	}
	name := m.Name.ValueString()
	body.Name = &name

	var paths []string
	m.Paths.ElementsAs(ctx, &paths, false)
	body.Paths = &paths

	if !m.Methods.IsNull() && !m.Methods.IsUnknown() {
		var methodStrs []string
		m.Methods.ElementsAs(ctx, &methodStrs, false)
		methods := make([]client.PutPublishedServiceRouteJSONBodyMethods, len(methodStrs))
		for i, s := range methodStrs {
			methods[i] = client.PutPublishedServiceRouteJSONBodyMethods(s)
		}
		body.Methods = &methods
	}

	if !m.Plugins.IsNull() && !m.Plugins.IsUnknown() {
		var plugins map[string]interface{}
		_ = json.Unmarshal([]byte(m.Plugins.ValueString()), &plugins)
		body.Plugins = &plugins
	}

	if !m.Labels.IsNull() && !m.Labels.IsUnknown() {
		var labels map[string]string
		m.Labels.ElementsAs(ctx, &labels, false)
		body.Labels = &labels
	}

	return body
}

func buildRouteModelFromResponse(ctx context.Context, val interface{}) (RouteResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	// Use JSON round-trip to avoid dealing with anonymous struct types from oapi-codegen.
	raw, _ := json.Marshal(val)
	var m map[string]interface{}
	_ = json.Unmarshal(raw, &m)

	state := RouteResourceModel{
		Plugins: jsontypes.NewNormalizedNull(),
		Labels:  types.MapValueMust(types.StringType, map[string]attr.Value{}),
		Methods: types.ListValueMust(types.StringType, []attr.Value{}),
		Paths:   types.ListValueMust(types.StringType, []attr.Value{}),
		Timeout: gen.NewTimeoutValueNull(),
	}

	if id, ok := m["id"].(string); ok {
		state.Id = types.StringValue(id)
	}
	if name, ok := m["name"].(string); ok {
		state.Name = types.StringValue(name)
	}
	if serviceID, ok := m["service_id"].(string); ok {
		state.ServiceId = types.StringValue(serviceID)
	}
	if desc, ok := m["desc"].(string); ok {
		state.Desc = types.StringValue(desc)
	}
	if ws, ok := m["enable_websocket"].(bool); ok {
		state.EnableWebsocket = types.BoolValue(ws)
	}
	if p, ok := m["priority"].(float64); ok {
		state.Priority = types.Int64Value(int64(p))
	}

	// paths
	if pathsRaw, ok := m["paths"].([]interface{}); ok {
		elems := make([]attr.Value, 0, len(pathsRaw))
		for _, p := range pathsRaw {
			if s, ok := p.(string); ok {
				elems = append(elems, types.StringValue(s))
			}
		}
		state.Paths = types.ListValueMust(types.StringType, elems)
	}

	// methods
	if methodsRaw, ok := m["methods"].([]interface{}); ok {
		elems := make([]attr.Value, 0, len(methodsRaw))
		for _, method := range methodsRaw {
			if s, ok := method.(string); ok {
				elems = append(elems, types.StringValue(s))
			}
		}
		state.Methods = types.ListValueMust(types.StringType, elems)
	}

	// plugins
	if pluginsRaw, ok := m["plugins"]; ok && pluginsRaw != nil {
		b, _ := json.Marshal(pluginsRaw)
		state.Plugins = jsontypes.NewNormalizedValue(string(b))
	}

	// labels
	if labelsRaw, ok := m["labels"].(map[string]interface{}); ok && len(labelsRaw) > 0 {
		elems := make(map[string]attr.Value, len(labelsRaw))
		for k, v := range labelsRaw {
			if s, ok := v.(string); ok {
				elems[k] = types.StringValue(s)
			}
		}
		state.Labels = types.MapValueMust(types.StringType, elems)
	}

	// timeout
	if timeoutRaw, ok := m["timeout"].(map[string]interface{}); ok {
		connect := types.Int64Value(int64(timeoutRaw["connect"].(float64)))
		send := types.Int64Value(int64(timeoutRaw["send"].(float64)))
		read := types.Int64Value(int64(timeoutRaw["read"].(float64)))
		tv, d := gen.NewTimeoutValue(
			gen.TimeoutValue{}.AttributeTypes(ctx),
			map[string]attr.Value{
				"connect": connect,
				"send":    send,
				"read":    read,
			},
		)
		diags.Append(d...)
		state.Timeout = tv
	}

	return state, diags
}
