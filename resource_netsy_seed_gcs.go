// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*netsySeedGCSResource)(nil)

type netsySeedGCSResource struct{}

type netsySeedGCSModel struct {
	ID                types.String `tfsdk:"id"`
	ClusterConfigPath types.String `tfsdk:"cluster_config_path"`
	SeedPath          types.String `tfsdk:"seed_path"`
	ValuesPath        types.String `tfsdk:"values_path"`
	Bucket            types.String `tfsdk:"bucket"`
	Prefix            types.String `tfsdk:"prefix"`
	Project           types.String `tfsdk:"project"`
}

// NewNetsySeedGCSResource returns the GCS Netsy seed resource implementation.
func NewNetsySeedGCSResource() resource.Resource {
	return &netsySeedGCSResource{}
}

// Metadata returns the resource type name.
func (r *netsySeedGCSResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_netsy_seed_gcs"
}

// Schema returns the resource schema.
func (r *netsySeedGCSResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{Computed: true},
			"cluster_config_path": resourceschema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"seed_path": resourceschema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"values_path": resourceschema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bucket": resourceschema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"prefix": resourceschema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project": resourceschema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create generates the Netsy snapshot, uploads it to GCS, and records state.
func (r *netsySeedGCSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan netsySeedGCSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	opts := seedOptionsFromGCSModel(plan)
	doSeed, err := shouldSeed(opts)
	if err != nil {
		resp.Diagnostics.AddError("Read Netsy seed config", err.Error())
		return
	}
	if doSeed {
		if err := checkSnapshotTargetGCS(ctx, opts); err != nil {
			resp.Diagnostics.AddError("Check Netsy seed target", err.Error())
			return
		}
		seedPath, cleanup, err := resolveSeedPath(ctx, opts)
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			resp.Diagnostics.AddError("Resolve Netsy seed", err.Error())
			return
		}
		path, snapshotCleanup, err := generateNetsySeedSnapshot(ctx, opts, seedPath)
		if snapshotCleanup != nil {
			defer snapshotCleanup()
		}
		if err != nil {
			resp.Diagnostics.AddError("Generate Netsy snapshot", err.Error())
			return
		}
		if err := uploadSnapshotGCS(ctx, opts, path); err != nil {
			resp.Diagnostics.AddError("Upload Netsy snapshot to GCS", err.Error())
			return
		}
	}
	plan.ID = types.StringValue("gs://" + opts.Bucket + "/" + opts.objectKey())
	plan.Prefix = types.StringValue(opts.Prefix)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read keeps the stored seed resource state.
func (r *netsySeedGCSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state netsySeedGCSModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update updates Terraform state; seed-affecting fields require replacement.
func (r *netsySeedGCSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan netsySeedGCSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	opts := seedOptionsFromGCSModel(plan)
	plan.ID = types.StringValue("gs://" + opts.Bucket + "/" + opts.objectKey())
	plan.Prefix = types.StringValue(opts.Prefix)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete leaves remote Netsy state intact and removes Terraform state.
func (r *netsySeedGCSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

// seedOptionsFromGCSModel converts Terraform GCS resource state to seed options.
func seedOptionsFromGCSModel(model netsySeedGCSModel) SeedOptions {
	return SeedOptions{
		ClusterConfigPath: stringOr(model.ClusterConfigPath, ""),
		SeedPath:          stringOr(model.SeedPath, ""),
		ValuesPath:        stringOr(model.ValuesPath, ""),
		Bucket:            stringOr(model.Bucket, ""),
		Prefix:            stringOr(model.Prefix, ""),
		Project:           stringOr(model.Project, ""),
	}
}
