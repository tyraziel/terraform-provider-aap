package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"

	"github.com/ansible/terraform-provider-aap/internal/provider/customtypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// inventoryDataSourceModel maps the data source schema data.
type InventoryDataSourceModel struct {
	Id               types.Int64                      `tfsdk:"id"`
	Organization     types.Int64                      `tfsdk:"organization"`
	OrganizationName types.String                     `tfsdk:"organization_name"`
	Url              types.String                     `tfsdk:"url"`
	NamedUrl         types.String                     `tfsdk:"named_url"`
	Name             types.String                     `tfsdk:"name"`
	Description      types.String                     `tfsdk:"description"`
	Variables        customtypes.AAPCustomStringValue `tfsdk:"variables"`
}

// InventoryDataSource is the data source implementation.
type InventoryDataSource struct {
	client *AAPClient
}

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &InventoryDataSource{}
	_ datasource.DataSourceWithConfigure = &InventoryDataSource{}
)

// NewInventoryDataSource is a helper function to simplify the provider implementation.
func NewInventoryDataSource() datasource.DataSource {
	return &InventoryDataSource{}
}

// Metadata returns the data source type name.
func (d *InventoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inventory"
}

// Schema defines the schema for the data source.
func (d *InventoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:    true,
				Description: "Inventory id",
			},
			"organization": schema.Int64Attribute{
				Computed:    true,
				Description: "Identifier for the organization to which the inventory belongs",
			},
			"organization_name": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "The name for the organization to which the inventory belongs",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "Url of the inventory",
			},
			"named_url": schema.StringAttribute{
				Computed:    true,
				Description: "The Named Url of the inventory",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "Name of the inventory",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the inventory",
			},
			"variables": schema.StringAttribute{
				Computed:    true,
				CustomType:  customtypes.AAPCustomStringType{},
				Description: "Variables of the inventory. Will be either JSON or YAML string depending on how the variables were entered into AAP.",
			},
		},
		Description: `Get an existing inventory.`,
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *InventoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state InventoryDataSourceModel
	var diags diag.Diagnostics

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceURL, err := state.ValidateLookupParameters(d)
	if err != nil {
		resp.Diagnostics.AddError("Minimal Data Not Supplied", "Require [id] or [name and organization_name]")
		return
	}

	readResponseBody, diags := d.client.Get(resourceURL)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = state.ParseHttpResponse(readResponseBody)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Set state
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *InventoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*AAPClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *AAPClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (dm *InventoryDataSourceModel) ParseHttpResponse(body []byte) diag.Diagnostics {
	var diags diag.Diagnostics

	// Unmarshal the JSON response
	var apiInventory InventoryAPIModel
	err := json.Unmarshal(body, &apiInventory)
	if err != nil {
		diags.AddError("Error parsing JSON response from AAP", err.Error())
		return diags
	}

	// Map response to the inventory datesource schema
	dm.Id = types.Int64Value(apiInventory.Id)
	dm.Organization = types.Int64Value(apiInventory.Organization)
	dm.OrganizationName = types.StringValue(apiInventory.SummaryFields.Organization.Name)
	dm.Url = types.StringValue(apiInventory.Url)
	dm.NamedUrl = types.StringValue(apiInventory.Related.NamedUrl)
	dm.Name = ParseStringValue(apiInventory.Name)
	dm.Description = ParseStringValue(apiInventory.Description)
	dm.Variables = ParseAAPCustomStringValue(apiInventory.Variables)

	return diags
}

// ValidateLookupParameters Validate the provided lookup parameters and return the appropriate resource url
func (dm *InventoryDataSourceModel) ValidateLookupParameters(datasource *InventoryDataSource) (string, error) {
	//Here is where we can get the "named" inventory, which is "Inventory Name"++"Organization Name" to derive uniqueness
	//we will take precedence if the Id is set to use that over the named_url attempt.
	if !dm.Id.IsNull() {
		return path.Join(datasource.client.getApiEndpoint(), "inventories", dm.Id.String()), nil
	} else if dm.Name.ValueString() != "" && dm.OrganizationName.ValueString() != "" {
		namedUrl := fmt.Sprintf("%s++%s", dm.Name.ValueString(), dm.OrganizationName.ValueString())
		return path.Join(datasource.client.getApiEndpoint(), "inventories", namedUrl), nil
	} else {
		return "", errors.New("invalid inventory lookup parameters")
	}
}
