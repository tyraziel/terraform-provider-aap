package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"slices"

	"github.com/ansible/terraform-provider-aap/internal/provider/customtypes"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// type AAPDataSource[T any] struct {
// 	client ProviderHTTPClient
// }

// type AAPDataSourceModel[T any] struct {
// 	Id               types.Int64
// 	OrganizationName types.String
// 	Name             types.String
// }

// Id               types.Int64                      `tfsdk:"id"`
// Organization     types.Int64                      `tfsdk:"organization"`
// OrganizationName types.String                     `tfsdk:"organization_name"`
// Url              types.String                     `tfsdk:"url"`
// NamedUrl         types.String                     `tfsdk:"named_url"`
// Name             types.String                     `tfsdk:"name"`
// Description      types.String                     `tfsdk:"description"`
// Variables        customtypes.AAPCustomStringValue `tfsdk:"variables"`

// func ReturnAAPNamedURL[T struct{}](model AAPDataSourceModel[T], source *AAPDataSource[T], URI string) (string, error) {
// 	if !model.Id.IsNull() {
// 		return path.Join(source.client.getApiEndpoint(), URI, model.Id.String()), nil
// 	} else if !model.Name.IsNull() && !model.OrganizationName.IsNull() {
// 		namedUrl := fmt.Sprintf("%s++%s", model.Name.ValueString(), model.OrganizationName.ValueString())
// 		return path.Join(source.client.getApiEndpoint(), URI, namedUrl), nil
// 	} else {
// 		return "", errors.New("invalid lookup parameters")
// 	}
// }

type AAPDataSourceUtils interface {
	ReturnAAPNamedURL(id types.Int64, name types.String, orgName types.String, URI string) (string, error)
}

func ReturnAAPNamedURL(id types.Int64, name types.String, orgName types.String, URI string) (string, error) {
	if !id.IsNull() {
		return path.Join(URI, id.String()), nil
	} else if !name.IsNull() && !orgName.IsNull() {
		namedUrl := fmt.Sprintf("%s++%s", name.ValueString(), orgName.ValueString())
		return path.Join(URI, namedUrl), nil
	} else {
		return "", errors.New("invalid lookup parameters")
	}
}

func IsValueProvided(value attr.Value) bool {
	return !value.IsNull() && !value.IsUnknown()
}

func ValidateResponse(resp *http.Response, body []byte, err error, expected_statuses []int) diag.Diagnostics {
	var diags diag.Diagnostics

	if err != nil {
		diags.AddError(
			"Client request error",
			err.Error(),
		)
		return diags
	}
	if resp == nil {
		diags.AddError("HTTP response error", "No HTTP response from server")
		return diags
	}
	if !slices.Contains(expected_statuses, resp.StatusCode) {
		var info map[string]interface{}
		_ = json.Unmarshal(body, &info)
		diags.AddError(
			fmt.Sprintf("Unexpected HTTP status code received for %s request to path %s", resp.Request.Method, resp.Request.URL),
			fmt.Sprintf("Expected one of (%v), got (%d). Response details: %v", expected_statuses, resp.StatusCode, info),
		)
		return diags
	}

	return diags
}

func getURL(base string, paths ...string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	u, err := url.ParseRequestURI(base)
	if err != nil {
		diags.AddError("Error parsing the URL", err.Error())
		return "", diags
	}

	u.Path = path.Join(append([]string{u.Path}, paths...)...)

	return u.String(), diags
}

type ParseValue interface {
	ParseValue(value string) any
}

type StringTyped struct {
}

func (t *StringTyped) ParseValue(value string) types.String {
	if value != "" {
		return types.StringValue(value)
	} else {
		return types.StringNull()
	}
}

func ParseStringValue(description string) types.String {
	if description != "" {
		return types.StringValue(description)
	} else {
		return types.StringNull()
	}
}

func ParseNormalizedValue(variables string) jsontypes.Normalized {
	if variables != "" {
		return jsontypes.NewNormalizedValue(variables)
	} else {
		return jsontypes.NewNormalizedNull()
	}
}

func ParseAAPCustomStringValue(variables string) customtypes.AAPCustomStringValue {
	if variables != "" {
		return customtypes.NewAAPCustomStringValue(variables)
	} else {
		return customtypes.NewAAPCustomStringNull()
	}
}
