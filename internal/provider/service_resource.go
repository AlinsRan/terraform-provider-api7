package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/api7/terraform-provider-api7/internal/client"
	gen "github.com/api7/terraform-provider-api7/internal/provider/generated/resource_service"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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

// ServiceResourceModel mirrors the schema fields used for CRUD mapping.
// plugins is jsontypes.Normalized (free-form JSON) instead of the generated empty SingleNested.
// upstream uses a simple struct instead of the generated UpstreamValue to avoid
// managing deeply nested custom types while still benefiting from validators on
// the other service-level fields.
type ServiceResourceModel struct {
	Desc            types.String         `tfsdk:"desc"`
	Hosts           types.List           `tfsdk:"hosts"`
	Id              types.String         `tfsdk:"id"`
	Labels          types.Map            `tfsdk:"labels"`
	Name            types.String         `tfsdk:"name"`
	PathPrefix      types.String         `tfsdk:"path_prefix"`
	Plugins         jsontypes.Normalized `tfsdk:"plugins"`
	Status          types.Int64          `tfsdk:"status"`
	StripPathPrefix types.Bool           `tfsdk:"strip_path_prefix"`
	Type            types.String         `tfsdk:"type"`
	Upstream        *UpstreamModel       `tfsdk:"upstream"`
}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

func (r *ServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

// Schema uses the generated schema (with auto-derived validators for name, status, type, hosts,
// path_prefix, etc.) and overrides specific attributes:
//   - upstream: replaced with a simple nodes/scheme/type schema to avoid complex custom types
//   - plugins: replaced with jsontypes.Normalized for free-form JSON
//   - id: made Computed-only with UseStateForUnknown
//   - gateway_group_id: removed (managed via provider config)
func (r *ServiceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := gen.ServiceResourceSchema(ctx)
	s.Description = "Manages an API7 Service (published service)."

	// Override upstream: use simple schema rather than the generated deeply nested UpstreamValue.
	s.Attributes["upstream"] = schema.SingleNestedAttribute{
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
	}

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
		Description: "Service ID (assigned by API7).",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}

	// Override type: make it optional/computed with "http" as the only valid value.
	// The generated schema marks it Required, but since "http" is the only option,
	// we default it to avoid forcing users to specify the obvious value.
	s.Attributes["type"] = schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Service type. Currently only \"http\" is supported.",
	}

	// Remove internal query-param fields that are not resource attributes.
	delete(s.Attributes, "gateway_group_id")

	resp.Schema = s
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

	serviceID := state.Id.ValueString()
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

	serviceID := plan.Id.ValueString()
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

	serviceID := state.Id.ValueString()
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

// --- helpers ---

// buildServiceRequestBody serializes the service model to JSON for create/update.
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

	serviceType := "http"
	if !m.Type.IsNull() && !m.Type.IsUnknown() && m.Type.ValueString() != "" {
		serviceType = m.Type.ValueString()
	}

	body := map[string]interface{}{
		"name":     m.Name.ValueString(),
		"type":     serviceType,
		"upstream": upstream,
	}
	if !m.Desc.IsNull() && !m.Desc.IsUnknown() {
		body["desc"] = m.Desc.ValueString()
	}
	if !m.Hosts.IsNull() && !m.Hosts.IsUnknown() {
		var hosts []string
		m.Hosts.ElementsAs(context.Background(), &hosts, false)
		body["hosts"] = hosts
	}
	if !m.PathPrefix.IsNull() && !m.PathPrefix.IsUnknown() {
		body["path_prefix"] = m.PathPrefix.ValueString()
	}
	if !m.Labels.IsNull() && !m.Labels.IsUnknown() {
		var labels map[string]string
		m.Labels.ElementsAs(context.Background(), &labels, false)
		body["labels"] = labels
	}
	if !m.Status.IsNull() && !m.Status.IsUnknown() {
		body["status"] = m.Status.ValueInt64()
	}
	if !m.StripPathPrefix.IsNull() && !m.StripPathPrefix.IsUnknown() {
		body["strip_path_prefix"] = m.StripPathPrefix.ValueBool()
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
func buildServiceModelFromResponse(val interface{}, prevModel *ServiceResourceModel) ServiceResourceModel {
	raw, _ := json.Marshal(val)
	var m map[string]interface{}
	_ = json.Unmarshal(raw, &m)

	state := ServiceResourceModel{
		Upstream:        prevModel.Upstream,
		Plugins:         jsontypes.NewNormalizedNull(),
		Hosts:           types.ListValueMust(types.StringType, []attr.Value{}),
		Labels:          types.MapValueMust(types.StringType, map[string]attr.Value{}),
		StripPathPrefix: types.BoolNull(),
		Status:          types.Int64Null(),
	}

	if id, ok := m["id"].(string); ok {
		state.Id = types.StringValue(id)
	}
	if name, ok := m["name"].(string); ok {
		state.Name = types.StringValue(name)
	}
	if desc, ok := m["desc"].(string); ok {
		state.Desc = types.StringValue(desc)
	} else {
		state.Desc = types.StringNull()
	}
	if t, ok := m["type"].(string); ok {
		state.Type = types.StringValue(t)
	}
	if pathPrefix, ok := m["path_prefix"].(string); ok {
		state.PathPrefix = types.StringValue(pathPrefix)
	} else {
		state.PathPrefix = types.StringNull()
	}
	if status, ok := m["status"].(float64); ok {
		state.Status = types.Int64Value(int64(status))
	}
	if strip, ok := m["strip_path_prefix"].(bool); ok {
		state.StripPathPrefix = types.BoolValue(strip)
	}

	if hostsRaw, ok := m["hosts"].([]interface{}); ok && len(hostsRaw) > 0 {
		elems := make([]attr.Value, 0, len(hostsRaw))
		for _, h := range hostsRaw {
			if s, ok := h.(string); ok {
				elems = append(elems, types.StringValue(s))
			}
		}
		state.Hosts = types.ListValueMust(types.StringType, elems)
	}

	if labelsRaw, ok := m["labels"].(map[string]interface{}); ok && len(labelsRaw) > 0 {
		elems := make(map[string]attr.Value, len(labelsRaw))
		for k, v := range labelsRaw {
			if s, ok := v.(string); ok {
				elems[k] = types.StringValue(s)
			}
		}
		state.Labels = types.MapValueMust(types.StringType, elems)
	}

	if pluginsRaw, ok := m["plugins"]; ok && pluginsRaw != nil {
		b, _ := json.Marshal(pluginsRaw)
		state.Plugins = jsontypes.NewNormalizedValue(string(b))
	}

	return state
}
