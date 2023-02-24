package provider

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jcmturner/gokrb5/v8/iana/etypeID"
	"github.com/jcmturner/gokrb5/v8/keytab"

	"github.com/zanecodes/terraform-provider-keytab/internal/validators/timevalidator"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &FileResource{}

func NewFileResource() resource.Resource {
	return &FileResource{}
}

// FileResource defines the resource implementation.
type FileResource struct {
}

// FileResourceModel describes the resource data model.
type FileResourceModel struct {
	Entries       []FileEntryModel `tfsdk:"entry"`
	ContentBase64 types.String     `tfsdk:"content_base64"`
	Id            types.String     `tfsdk:"id"`
}

type FileEntryModel struct {
	Principal      types.String `tfsdk:"principal"`
	Realm          types.String `tfsdk:"realm"`
	Key            types.String `tfsdk:"key"`
	KeyVersion     types.Int64  `tfsdk:"key_version"`
	EncryptionType types.String `tfsdk:"encryption_type"`
	Timestamp      types.String `tfsdk:"timestamp"`
}

func (r *FileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

func (r *FileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	supported_etypes := make([]string, 0)

	for k := range etypeID.ETypesByName {
		if etypeID.EtypeSupported(k) != 0 {
			supported_etypes = append(supported_etypes, k)
		}
	}

	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "File resource",

		Blocks: map[string]schema.Block{
			"entry": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"principal": schema.StringAttribute{
							MarkdownDescription: "The name of the Kerberos principal to which the key belongs, not including the realm.",
							Required:            true,
						},
						"realm": schema.StringAttribute{
							MarkdownDescription: "The realm to which the Kerberos principal belongs.",
							Required:            true,
						},
						"key": schema.StringAttribute{
							MarkdownDescription: "The key belonging to the Kerberos principal.",
							Required:            true,
							Sensitive:           true,
						},
						"key_version": schema.Int64Attribute{
							MarkdownDescription: "The version number of the key.",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.Between(0, math.MaxUint8),
							},
						},
						"encryption_type": schema.StringAttribute{
							MarkdownDescription: "The encryption type to use for the key. Must be one of: `aes128-cts-hmac-sha1-96`/`aes128-cts`/`aes128-sha1`, `aes256-cts-hmac-sha1-96`/`aes256-cts`/`aes256-sha1`, `aes128-cts-hmac-sha256-128`/`aes128-sha2`, `aes256-cts-hmac-sha384-192`/`aes256-sha2`, `des3-cbc-sha1-kd`, or `arcfour-hmac`/`rc4-hmac`/`arcfour-hmac-md5`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf(supported_etypes...),
							},
						},
						"timestamp": schema.StringAttribute{
							MarkdownDescription: "The creation timestamp for the Keytab entry in [RFC3339](https://datatracker.ietf.org/doc/html/rfc3339#section-5.8) format. Defaults to the current time, but may be set to a specific time for better reproducibility.",
							Optional:            true,
							Computed:            true,
							Validators: []validator.String{
								timevalidator.IsRFC3339Time(),
							},
						},
					},

					PlanModifiers: []planmodifier.Object{
						objectplanmodifier.RequiresReplaceIfConfigured(),
					},
				},
			},
		},

		Attributes: map[string]schema.Attribute{
			"content_base64": schema.StringAttribute{
				MarkdownDescription: "The base64 encoded keytab contents.",
				Computed:            true,
				Sensitive:           true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The SHA256 hash of the binary keytab contents.",
				Computed:            true,
			},
		},
	}
}

func (r *FileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
}

func (r *FileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *FileResourceModel

	now := time.Now()

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	keytab := keytab.New()

	for i, entry := range data.Entries {
		var timestamp time.Time

		if entry.Timestamp.IsUnknown() {
			timestamp = now
			data.Entries[i].Timestamp = types.StringValue(timestamp.Format(time.RFC3339))
		} else {
			var err error
			timestamp, err = time.Parse(time.RFC3339, entry.Timestamp.ValueString())

			if err != nil {
				resp.Diagnostics.AddError("Invalid timestamp", err.Error())
				return
			}
		}

		if err := keytab.AddEntry(entry.Principal.ValueString(), entry.Realm.ValueString(), entry.Key.ValueString(), timestamp, uint8(entry.KeyVersion.ValueInt64()), etypeID.EtypeSupported(entry.EncryptionType.ValueString())); err != nil {
			resp.Diagnostics.AddError("Invalid keytab entry", err.Error())
			return
		}
	}

	bytes, err := keytab.Marshal()

	if err != nil {
		resp.Diagnostics.AddError("Unable to generate keytab", err.Error())
		return
	}

	data.ContentBase64 = types.StringValue(base64.StdEncoding.EncodeToString(bytes))

	sum := sha256.Sum256(bytes)
	data.Id = types.StringValue(fmt.Sprintf("%x", sum[:]))

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *FileResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *FileResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *FileResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}
