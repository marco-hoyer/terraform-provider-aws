package fsx

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/fsx"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceOntapVolume() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceOntapVolumeCreate,
		ReadWithoutTimeout:   resourceOntapVolumeRead,
		UpdateWithoutTimeout: resourceOntapVolumeUpdate,
		DeleteWithoutTimeout: resourceOntapVolumeDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"file_system_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"flexcache_endpoint_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"junction_path": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(1, 255),
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 203),
			},
			"ontap_volume_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"security_style": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "UNIX",
				ValidateFunc: validation.StringInSlice(fsx.StorageVirtualMachineRootVolumeSecurityStyle_Values(), false),
			},
			"size_in_megabytes": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(0, 2147483647),
			},
			"storage_efficiency_enabled": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"storage_virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(21, 21),
			},
			"tiering_policy": {
				Type:             schema.TypeList,
				Optional:         true,
				DiffSuppressFunc: verify.SuppressMissingOptionalConfigurationBlock,
				MaxItems:         1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cooling_period": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(2, 183),
						},
						"name": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.StringInSlice(fsx.TieringPolicyName_Values(), false),
						},
					},
				},
			},
			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"volume_type": {
				Type:         schema.TypeString,
				Default:      fsx.VolumeTypeOntap,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(fsx.VolumeType_Values(), false),
			},
		},
		CustomizeDiff: verify.SetTagsDiff,
	}
}

func resourceOntapVolumeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).FSxConn()
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(ctx, d.Get("tags").(map[string]interface{})))

	input := &fsx.CreateVolumeInput{
		Name:       aws.String(d.Get("name").(string)),
		VolumeType: aws.String(d.Get("volume_type").(string)),
		OntapConfiguration: &fsx.CreateOntapVolumeConfiguration{
			JunctionPath:             aws.String(d.Get("junction_path").(string)),
			SizeInMegabytes:          aws.Int64(int64(d.Get("size_in_megabytes").(int))),
			StorageEfficiencyEnabled: aws.Bool(d.Get("storage_efficiency_enabled").(bool)),
			StorageVirtualMachineId:  aws.String(d.Get("storage_virtual_machine_id").(string)),
		},
	}

	if v, ok := d.GetOk("security_style"); ok {
		input.OntapConfiguration.SecurityStyle = aws.String(v.(string))
	}

	if v, ok := d.GetOk("tiering_policy"); ok {
		input.OntapConfiguration.TieringPolicy = expandOntapVolumeTieringPolicy(v.([]interface{}))
	}

	if len(tags) > 0 {
		input.Tags = Tags(tags.IgnoreAWS())
	}

	log.Printf("[DEBUG] Creating FSx ONTAP Volume: %s", input)
	result, err := conn.CreateVolumeWithContext(ctx, input)

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "creating FSx Volume: %s", err)
	}

	d.SetId(aws.StringValue(result.Volume.VolumeId))

	if _, err := waitVolumeCreated(ctx, conn, d.Id(), d.Timeout(schema.TimeoutCreate)); err != nil {
		return sdkdiag.AppendErrorf(diags, "waiting for FSx Volume(%s) create: %s", d.Id(), err)
	}

	return append(diags, resourceOntapVolumeRead(ctx, d, meta)...)
}

func resourceOntapVolumeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).FSxConn()
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	volume, err := FindVolumeByID(ctx, conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] FSx ONTAP Volume (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading FSx ONTAP Volume (%s): %s", d.Id(), err)
	}

	ontapConfig := volume.OntapConfiguration
	if ontapConfig == nil {
		return sdkdiag.AppendErrorf(diags, "describing FSx ONTAP Volume (%s): empty ONTAP configuration", d.Id())
	}

	d.Set("arn", volume.ResourceARN)
	d.Set("name", volume.Name)
	d.Set("file_system_id", volume.FileSystemId)
	d.Set("junction_path", ontapConfig.JunctionPath)
	d.Set("ontap_volume_type", ontapConfig.OntapVolumeType)
	d.Set("security_style", ontapConfig.SecurityStyle)
	d.Set("size_in_megabytes", ontapConfig.SizeInMegabytes)
	d.Set("storage_efficiency_enabled", ontapConfig.StorageEfficiencyEnabled)
	d.Set("storage_virtual_machine_id", ontapConfig.StorageVirtualMachineId)
	d.Set("uuid", ontapConfig.UUID)
	d.Set("volume_type", volume.VolumeType)

	if err := d.Set("tiering_policy", flattenOntapVolumeTieringPolicy(ontapConfig.TieringPolicy)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting tiering_policy: %s", err)
	}

	//Volume tags do not get returned with describe call so need to make a separate list tags call
	tags, tagserr := ListTags(ctx, conn, *volume.ResourceARN)

	if tagserr != nil {
		return sdkdiag.AppendErrorf(diags, "reading Tags for FSx ONTAP Volume (%s): %s", d.Id(), err)
	} else {
		tags = tags.IgnoreAWS().IgnoreConfig(ignoreTagsConfig)
	}

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting tags: %s", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting tags_all: %s", err)
	}

	return diags
}

func resourceOntapVolumeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).FSxConn()

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := UpdateTags(ctx, conn, d.Get("arn").(string), o, n); err != nil {
			return sdkdiag.AppendErrorf(diags, "updating FSx ONTAP Volume (%s) tags: %s", d.Get("arn").(string), err)
		}
	}

	if d.HasChangesExcept("tags_all", "tags") {
		input := &fsx.UpdateVolumeInput{
			ClientRequestToken: aws.String(resource.UniqueId()),
			VolumeId:           aws.String(d.Id()),
			OntapConfiguration: &fsx.UpdateOntapVolumeConfiguration{},
		}

		if d.HasChange("junction_path") {
			input.OntapConfiguration.JunctionPath = aws.String(d.Get("junction_path").(string))
		}

		if d.HasChange("security_style") {
			input.OntapConfiguration.SecurityStyle = aws.String(d.Get("security_style").(string))
		}

		if d.HasChange("size_in_megabytes") {
			input.OntapConfiguration.SizeInMegabytes = aws.Int64(int64(d.Get("size_in_megabytes").(int)))
		}

		if d.HasChange("storage_efficiency_enabled") {
			input.OntapConfiguration.StorageEfficiencyEnabled = aws.Bool(d.Get("storage_efficiency_enabled").(bool))
		}

		if d.HasChange("tiering_policy") {
			input.OntapConfiguration.TieringPolicy = expandOntapVolumeTieringPolicy(d.Get("tiering_policy").([]interface{}))
		}

		_, err := conn.UpdateVolumeWithContext(ctx, input)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating FSx ONTAP Volume (%s): %s", d.Id(), err)
		}

		if _, err := waitVolumeUpdated(ctx, conn, d.Id(), d.Timeout(schema.TimeoutUpdate)); err != nil {
			return sdkdiag.AppendErrorf(diags, "waiting for FSx ONTAP Volume (%s) update: %s", d.Id(), err)
		}
	}

	return append(diags, resourceOntapVolumeRead(ctx, d, meta)...)
}

func resourceOntapVolumeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).FSxConn()

	log.Printf("[DEBUG] Deleting FSx ONTAP Volume: %s", d.Id())
	_, err := conn.DeleteVolumeWithContext(ctx, &fsx.DeleteVolumeInput{
		VolumeId: aws.String(d.Id()),
	})

	if tfawserr.ErrCodeEquals(err, fsx.ErrCodeVolumeNotFound) {
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "deleting FSx ONTAP Volume (%s): %s", d.Id(), err)
	}

	if _, err := waitVolumeDeleted(ctx, conn, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		return sdkdiag.AppendErrorf(diags, "waiting for FSx ONTAP Volume (%s) delete: %s", d.Id(), err)
	}

	return diags
}

func expandOntapVolumeTieringPolicy(cfg []interface{}) *fsx.TieringPolicy {
	if len(cfg) < 1 {
		return nil
	}

	conf := cfg[0].(map[string]interface{})

	out := fsx.TieringPolicy{}

	//Cooling period only accepts a minimum of 2 but int will return 0 not nil if unset
	//Therefore we only set it if it is 2 or more
	if v, ok := conf["cooling_period"].(int); ok && v >= 2 {
		out.CoolingPeriod = aws.Int64(int64(v))
	}

	if v, ok := conf["name"].(string); ok {
		out.Name = aws.String(v)
	}

	return &out
}

func flattenOntapVolumeTieringPolicy(rs *fsx.TieringPolicy) []interface{} {
	if rs == nil {
		return []interface{}{}
	}

	minCoolingPeriod := 2

	m := make(map[string]interface{})
	if aws.Int64Value(rs.CoolingPeriod) >= int64(minCoolingPeriod) {
		m["cooling_period"] = aws.Int64Value(rs.CoolingPeriod)
	}

	if rs.Name != nil {
		m["name"] = aws.StringValue(rs.Name)
	}

	return []interface{}{m}
}
