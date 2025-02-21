package ram

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ram"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceResourceShare() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceResourceShareCreate,
		ReadWithoutTimeout:   resourceResourceShareRead,
		UpdateWithoutTimeout: resourceResourceShareUpdate,
		DeleteWithoutTimeout: resourceResourceShareDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"allow_external_principals": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"permission_arns": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: verify.ValidARN,
				},
			},
			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),
		},

		CustomizeDiff: verify.SetTagsDiff,
	}
}

func resourceResourceShareCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).RAMConn()
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(ctx, d.Get("tags").(map[string]interface{})))

	name := d.Get("name").(string)
	input := &ram.CreateResourceShareInput{
		AllowExternalPrincipals: aws.Bool(d.Get("allow_external_principals").(bool)),
		Name:                    aws.String(name),
	}

	if v, ok := d.GetOk("permission_arns"); ok && v.(*schema.Set).Len() > 0 {
		input.PermissionArns = flex.ExpandStringSet(v.(*schema.Set))
	}

	if len(tags) > 0 {
		input.Tags = Tags(tags.IgnoreAWS())
	}

	log.Printf("[DEBUG] Creating RAM Resource Share: %s", input)
	output, err := conn.CreateResourceShareWithContext(ctx, input)

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "creating RAM Resource Share (%s): %s", name, err)
	}

	d.SetId(aws.StringValue(output.ResourceShare.ResourceShareArn))

	if _, err = WaitResourceShareOwnedBySelfActive(ctx, conn, d.Id(), d.Timeout(schema.TimeoutCreate)); err != nil {
		return sdkdiag.AppendErrorf(diags, "waiting for RAM Resource Share (%s) to become ready: %s", d.Id(), err)
	}

	return append(diags, resourceResourceShareRead(ctx, d, meta)...)
}

func resourceResourceShareRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).RAMConn()
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	resourceShare, err := FindResourceShareOwnerSelfByARN(ctx, conn, d.Id())

	if !d.IsNewResource() && tfawserr.ErrCodeEquals(err, ram.ErrCodeUnknownResourceException) {
		log.Printf("[WARN] RAM Resource Share (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading RAM Resource Share (%s): %s", d.Id(), err)
	}

	if !d.IsNewResource() && aws.StringValue(resourceShare.Status) != ram.ResourceShareStatusActive {
		log.Printf("[WARN] RAM Resource Share (%s) not active, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	d.Set("allow_external_principals", resourceShare.AllowExternalPrincipals)
	d.Set("arn", resourceShare.ResourceShareArn)
	d.Set("name", resourceShare.Name)

	tags := KeyValueTags(ctx, resourceShare.Tags).IgnoreAWS().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting tags: %s", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting tags_all: %s", err)
	}

	perms, err := conn.ListResourceSharePermissionsWithContext(ctx, &ram.ListResourceSharePermissionsInput{
		ResourceShareArn: aws.String(d.Id()),
	})

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "listing RAM Resource Share (%s) permissions: %s", d.Id(), err)
	}

	permissionARNs := make([]*string, 0, len(perms.Permissions))

	for _, v := range perms.Permissions {
		permissionARNs = append(permissionARNs, v.Arn)
	}

	d.Set("permission_arns", aws.StringValueSlice(permissionARNs))

	return diags
}

func resourceResourceShareUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).RAMConn()

	if d.HasChanges("name", "allow_external_principals") {
		input := &ram.UpdateResourceShareInput{
			AllowExternalPrincipals: aws.Bool(d.Get("allow_external_principals").(bool)),
			Name:                    aws.String(d.Get("name").(string)),
			ResourceShareArn:        aws.String(d.Id()),
		}

		log.Printf("[DEBUG] Updating RAM Resource Share: %s", input)
		_, err := conn.UpdateResourceShareWithContext(ctx, input)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating RAM Resource Share (%s): %s", d.Id(), err)
		}
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := UpdateTags(ctx, conn, d.Id(), o, n); err != nil {
			return sdkdiag.AppendErrorf(diags, "updating RAM Resource Share (%s) tags: %s", d.Id(), err)
		}
	}

	return append(diags, resourceResourceShareRead(ctx, d, meta)...)
}

func resourceResourceShareDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).RAMConn()

	log.Printf("[DEBUG] Deleting RAM Resource Share: %s", d.Id())
	_, err := conn.DeleteResourceShareWithContext(ctx, &ram.DeleteResourceShareInput{
		ResourceShareArn: aws.String(d.Id()),
	})

	if tfawserr.ErrCodeEquals(err, ram.ErrCodeUnknownResourceException) {
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "deleting RAM Resource Share (%s): %s", d.Id(), err)
	}

	if _, err = WaitResourceShareOwnedBySelfDeleted(ctx, conn, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		return sdkdiag.AppendErrorf(diags, "waiting for RAM Resource Share (%s) delete: %s", d.Id(), err)
	}

	return diags
}
