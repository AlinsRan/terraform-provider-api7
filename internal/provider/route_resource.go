package provider

import (
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

var _ resource.Resource = &RouteResource{}
var _ resource.ResourceWithImportState = &RouteResource{}

type RouteResource struct {
	client         *client.ClientWithResponses
	gatewayGroupID string
}

type RouteResourceModel struct {
	ID        types.String            `tfsdk:"id"`
	Name      types.String            `tfsdk:"name"`
	ServiceID types.String            `tfsdk:"service_id"`
	Paths     []types.String          `tfsdk:"paths"`
	Methods   []types.String          `tfsdk:"methods"`
	Labels    map[string]types.String `tfsdk:"labels"`
	Plugins   jsontypes.Normalized    `tfsdk:"plugins"`
}

func NewRouteResource() resource.Resource {
	return &RouteResource{}
}

func (r *RouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route"
}

func (r *RouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an API7 Route.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Route ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Route name.",
			},
			"service_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the parent service.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"paths": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: `URI paths to match, e.g. ["/api/v1/*"].`,
			},
			"methods": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: `HTTP methods to match, e.g. ["GET", "POST"]. Empty means all.`,
			},
			"plugins": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Plugin configuration as JSON string.",
				CustomType:  jsontypes.NormalizedType{},
			},
			"labels": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Key-value label pairs attached to the route.",
			},
		},
	}
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

	body := routeModelToCreateRequest(plan)
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

	state := buildRouteModelFromResponse(apiResp.JSON200.Value)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	routeID := state.ID.ValueString()
	params := &client.GetPublishedServiceRouteParams{GatewayGroupId: r.gatewayGroupID}

	apiResp, err := r.client.GetPublishedServiceRouteWithResponse(ctx, routeID, params)
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

	newState := buildRouteModelFromResponse(apiResp.JSON200.Value)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *RouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	routeID := plan.ID.ValueString()
	body := routeModelToPutRequest(plan)
	params := &client.PutPublishedServiceRouteParams{GatewayGroupId: r.gatewayGroupID}

	apiResp, err := r.client.PutPublishedServiceRouteWithResponse(ctx, routeID, params, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update route", err.Error())
		return
	}
	if apiResp.StatusCode() != 200 {
		resp.Diagnostics.AddError("API error updating route",
			fmt.Sprintf("status %d: %s", apiResp.StatusCode(), string(apiResp.Body)))
		return
	}

	newState := buildRouteModelFromResponse(apiResp.JSON200.Value)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *RouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	routeID := state.ID.ValueString()
	params := &client.DeletePublishedServiceRouteParams{GatewayGroupId: r.gatewayGroupID}

	apiResp, err := r.client.DeletePublishedServiceRouteWithResponse(ctx, routeID, params)
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

func routeModelToCreateRequest(m RouteResourceModel) client.CreatePublishedServiceRouteJSONRequestBody {
	body := client.CreatePublishedServiceRouteJSONRequestBody{
		ServiceId: m.ServiceID.ValueString(),
	}
	name := m.Name.ValueString()
	body.Name = &name

	paths := make([]string, 0, len(m.Paths))
	for _, p := range m.Paths {
		paths = append(paths, p.ValueString())
	}
	body.Paths = &paths

	if len(m.Methods) > 0 {
		methods := make([]client.CreatePublishedServiceRouteJSONBodyMethods, 0, len(m.Methods))
		for _, method := range m.Methods {
			methods = append(methods, client.CreatePublishedServiceRouteJSONBodyMethods(method.ValueString()))
		}
		body.Methods = &methods
	}

	if !m.Plugins.IsNull() && !m.Plugins.IsUnknown() {
		var plugins map[string]interface{}
		_ = json.Unmarshal([]byte(m.Plugins.ValueString()), &plugins)
		body.Plugins = &plugins
	}

	labels := make(map[string]string, len(m.Labels))
	for k, v := range m.Labels {
		labels[k] = v.ValueString()
	}
	body.Labels = &labels

	return body
}

func routeModelToPutRequest(m RouteResourceModel) client.PutPublishedServiceRouteJSONRequestBody {
	body := client.PutPublishedServiceRouteJSONRequestBody{
		ServiceId: m.ServiceID.ValueString(),
	}
	name := m.Name.ValueString()
	body.Name = &name

	paths := make([]string, 0, len(m.Paths))
	for _, p := range m.Paths {
		paths = append(paths, p.ValueString())
	}
	body.Paths = &paths

	if len(m.Methods) > 0 {
		methods := make([]client.PutPublishedServiceRouteJSONBodyMethods, 0, len(m.Methods))
		for _, method := range m.Methods {
			methods = append(methods, client.PutPublishedServiceRouteJSONBodyMethods(method.ValueString()))
		}
		body.Methods = &methods
	}

	if !m.Plugins.IsNull() && !m.Plugins.IsUnknown() {
		var plugins map[string]interface{}
		_ = json.Unmarshal([]byte(m.Plugins.ValueString()), &plugins)
		body.Plugins = &plugins
	}

	labels := make(map[string]string, len(m.Labels))
	for k, v := range m.Labels {
		labels[k] = v.ValueString()
	}
	body.Labels = &labels

	return body
}

// buildRouteModelFromResponse uses JSON re-encoding to avoid dealing with anonymous struct types.
func buildRouteModelFromResponse(val interface{}) RouteResourceModel {
	raw, _ := json.Marshal(val)
	var m map[string]interface{}
	_ = json.Unmarshal(raw, &m)

	state := RouteResourceModel{
		Plugins: jsontypes.NewNormalizedNull(),
	}

	if id, ok := m["id"].(string); ok {
		state.ID = types.StringValue(id)
	}
	if name, ok := m["name"].(string); ok {
		state.Name = types.StringValue(name)
	}
	if serviceID, ok := m["service_id"].(string); ok {
		state.ServiceID = types.StringValue(serviceID)
	}

	if pathsRaw, ok := m["paths"].([]interface{}); ok {
		for _, p := range pathsRaw {
			if s, ok := p.(string); ok {
				state.Paths = append(state.Paths, types.StringValue(s))
			}
		}
	}

	if methodsRaw, ok := m["methods"].([]interface{}); ok {
		for _, method := range methodsRaw {
			if s, ok := method.(string); ok {
				state.Methods = append(state.Methods, types.StringValue(s))
			}
		}
	}

	if pluginsRaw, ok := m["plugins"]; ok && pluginsRaw != nil {
		b, _ := json.Marshal(pluginsRaw)
		state.Plugins = jsontypes.NewNormalizedValue(string(b))
	}

	if labelsRaw, ok := m["labels"].(map[string]interface{}); ok && len(labelsRaw) > 0 {
		state.Labels = make(map[string]types.String, len(labelsRaw))
		for k, v := range labelsRaw {
			if s, ok := v.(string); ok {
				state.Labels[k] = types.StringValue(s)
			}
		}
	}

	return state
}
