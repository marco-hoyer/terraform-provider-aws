package ecr

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/service/ecr"
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

func ResourceRepository() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceRepositoryCreate,
		ReadWithoutTimeout:   resourceRepositoryRead,
		UpdateWithoutTimeout: resourceRepositoryUpdate,
		DeleteWithoutTimeout: resourceRepositoryDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		CustomizeDiff: verify.SetTagsDiff,

		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"encryption_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"encryption_type": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							Default:      ecr.EncryptionTypeAes256,
							ValidateFunc: validation.StringInSlice(ecr.EncryptionType_Values(), false),
						},
						"kms_key": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
					},
				},
				DiffSuppressFunc: verify.SuppressMissingOptionalConfigurationBlock,
				ForceNew:         true,
			},
			"force_delete": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"image_scanning_configuration": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"scan_on_push": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
				DiffSuppressFunc: verify.SuppressMissingOptionalConfigurationBlock,
			},
			"image_tag_mutability": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      ecr.ImageTagMutabilityMutable,
				ValidateFunc: validation.StringInSlice(ecr.ImageTagMutability_Values(), false),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"registry_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"repository_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),
		},
	}
}

func resourceRepositoryCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).ECRConn()
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(ctx, d.Get("tags").(map[string]interface{})))

	name := d.Get("name").(string)
	input := &ecr.CreateRepositoryInput{
		EncryptionConfiguration: expandRepositoryEncryptionConfiguration(d.Get("encryption_configuration").([]interface{})),
		ImageTagMutability:      aws.String(d.Get("image_tag_mutability").(string)),
		RepositoryName:          aws.String(name),
	}

	if v, ok := d.GetOk("image_scanning_configuration"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		tfMap := v.([]interface{})[0].(map[string]interface{})
		input.ImageScanningConfiguration = &ecr.ImageScanningConfiguration{
			ScanOnPush: aws.Bool(tfMap["scan_on_push"].(bool)),
		}
	}

	if len(tags) > 0 {
		input.Tags = Tags(tags.IgnoreAWS())
	}

	output, err := conn.CreateRepositoryWithContext(ctx, input)

	// Some partitions (i.e., ISO) may not support tag-on-create
	if input.Tags != nil && meta.(*conns.AWSClient).Partition != endpoints.AwsPartitionID && verify.ErrorISOUnsupported(conn.PartitionID, err) {
		log.Printf("[WARN] failed creating ECR Repository (%s) with tags: %s. Trying create without tags.", d.Get("name").(string), err)
		input.Tags = nil

		output, err = conn.CreateRepositoryWithContext(ctx, input)
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "creating ECR Repository (%s): %s", name, err)
	}

	d.SetId(aws.StringValue(output.Repository.RepositoryName))

	// Some partitions (i.e., ISO) may not support tag-on-create, attempt tag after create
	if input.Tags == nil && len(tags) > 0 && meta.(*conns.AWSClient).Partition != endpoints.AwsPartitionID {
		err := UpdateTags(ctx, conn, aws.StringValue(output.Repository.RepositoryArn), nil, tags)

		// If default tags only, log and continue. Otherwise, error.
		if v, ok := d.GetOk("tags"); (!ok || len(v.(map[string]interface{})) == 0) && verify.ErrorISOUnsupported(conn.PartitionID, err) {
			log.Printf("[WARN] failed adding tags after create for ECR Repository (%s): %s", d.Id(), err)
			return append(diags, resourceRepositoryRead(ctx, d, meta)...)
		}

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "adding tags after create for ECR Repository (%s): %s", d.Id(), err)
		}
	}

	return append(diags, resourceRepositoryRead(ctx, d, meta)...)
}

func resourceRepositoryRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).ECRConn()
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	outputRaw, err := tfresource.RetryWhenNewResourceNotFound(ctx, propagationTimeout, func() (interface{}, error) {
		return FindRepositoryByName(ctx, conn, d.Id())
	}, d.IsNewResource())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] ECR Repository (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	repository := outputRaw.(*ecr.Repository)
	arn := aws.StringValue(repository.RepositoryArn)
	d.Set("arn", arn)
	if err := d.Set("encryption_configuration", flattenRepositoryEncryptionConfiguration(repository.EncryptionConfiguration)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting encryption_configuration: %s", err)
	}
	if err := d.Set("image_scanning_configuration", flattenImageScanningConfiguration(repository.ImageScanningConfiguration)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting image_scanning_configuration: %s", err)
	}
	d.Set("image_tag_mutability", repository.ImageTagMutability)
	d.Set("name", repository.RepositoryName)
	d.Set("registry_id", repository.RegistryId)
	d.Set("repository_url", repository.RepositoryUri)

	tags, err := ListTags(ctx, conn, arn)

	// Some partitions (i.e., ISO) may not support tagging, giving error
	if meta.(*conns.AWSClient).Partition != endpoints.AwsPartitionID && verify.ErrorISOUnsupported(conn.PartitionID, err) {
		log.Printf("[WARN] failed listing tags for ECR Repository (%s): %s", d.Id(), err)
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "listing tags for ECR Repository (%s): %s", d.Id(), err)
	}

	tags = tags.IgnoreAWS().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting tags: %s", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting tags_all: %s", err)
	}

	return diags
}

func resourceRepositoryUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).ECRConn()

	if d.HasChange("image_tag_mutability") {
		input := &ecr.PutImageTagMutabilityInput{
			ImageTagMutability: aws.String(d.Get("image_tag_mutability").(string)),
			RegistryId:         aws.String(d.Get("registry_id").(string)),
			RepositoryName:     aws.String(d.Id()),
		}

		_, err := conn.PutImageTagMutabilityWithContext(ctx, input)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "setting ECR Repository (%s) image tag mutability: %s", d.Id(), err)
		}
	}

	if d.HasChange("image_scanning_configuration") {
		input := &ecr.PutImageScanningConfigurationInput{
			ImageScanningConfiguration: &ecr.ImageScanningConfiguration{},
			RegistryId:                 aws.String(d.Get("registry_id").(string)),
			RepositoryName:             aws.String(d.Id()),
		}

		if v, ok := d.GetOk("image_scanning_configuration"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
			tfMap := v.([]interface{})[0].(map[string]interface{})
			input.ImageScanningConfiguration.ScanOnPush = aws.Bool(tfMap["scan_on_push"].(bool))
		}

		_, err := conn.PutImageScanningConfigurationWithContext(ctx, input)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "setting ECR Repository (%s) image scanning configuration: %s", d.Id(), err)
		}
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		err := UpdateTags(ctx, conn, d.Get("arn").(string), o, n)

		// Some partitions may not support tagging, giving error
		if meta.(*conns.AWSClient).Partition != endpoints.AwsPartitionID && verify.ErrorISOUnsupported(conn.PartitionID, err) {
			log.Printf("[WARN] failed updating tags for ECR Repository (%s): %s", d.Id(), err)
			return append(diags, resourceRepositoryRead(ctx, d, meta)...)
		}

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating ECR Repository (%s) tags: %s", d.Id(), err)
		}
	}

	return append(diags, resourceRepositoryRead(ctx, d, meta)...)
}

func resourceRepositoryDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).ECRConn()

	log.Printf("[DEBUG] Deleting ECR Repository: %s", d.Id())
	_, err := conn.DeleteRepositoryWithContext(ctx, &ecr.DeleteRepositoryInput{
		Force:          aws.Bool(d.Get("force_delete").(bool)),
		RegistryId:     aws.String(d.Get("registry_id").(string)),
		RepositoryName: aws.String(d.Id()),
	})

	if tfawserr.ErrCodeEquals(err, ecr.ErrCodeRepositoryNotFoundException) {
		return diags
	}

	if tfawserr.ErrCodeEquals(err, ecr.ErrCodeRepositoryNotEmptyException) {
		return sdkdiag.AppendErrorf(diags, "ECR Repository (%s) not empty, consider using force_delete: %s", d.Id(), err)
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "deleting ECR Repository (%s): %s", d.Id(), err)
	}

	_, err = tfresource.RetryUntilNotFound(ctx, d.Timeout(schema.TimeoutDelete), func() (interface{}, error) {
		return FindRepositoryByName(ctx, conn, d.Id())
	})

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "waiting for ECR Repository (%s) delete: %s", d.Id(), err)
	}

	return diags
}

func FindRepositoryByName(ctx context.Context, conn *ecr.ECR, name string) (*ecr.Repository, error) {
	input := &ecr.DescribeRepositoriesInput{
		RepositoryNames: aws.StringSlice([]string{name}),
	}

	output, err := FindRepository(ctx, conn, input)

	if err != nil {
		return nil, err
	}

	// Eventual consistency check.
	if aws.StringValue(output.RepositoryName) != name {
		return nil, &resource.NotFoundError{
			LastRequest: input,
		}
	}

	return output, nil
}

func FindRepository(ctx context.Context, conn *ecr.ECR, input *ecr.DescribeRepositoriesInput) (*ecr.Repository, error) {
	output, err := conn.DescribeRepositoriesWithContext(ctx, input)

	if tfawserr.ErrCodeEquals(err, ecr.ErrCodeRepositoryNotFoundException) {
		return nil, &resource.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil || len(output.Repositories) == 0 || output.Repositories[0] == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	if count := len(output.Repositories); count > 1 {
		return nil, tfresource.NewTooManyResultsError(count, input)
	}

	return output.Repositories[0], nil
}

func flattenImageScanningConfiguration(isc *ecr.ImageScanningConfiguration) []map[string]interface{} {
	if isc == nil {
		return nil
	}

	config := make(map[string]interface{})
	config["scan_on_push"] = aws.BoolValue(isc.ScanOnPush)

	return []map[string]interface{}{
		config,
	}
}

func expandRepositoryEncryptionConfiguration(data []interface{}) *ecr.EncryptionConfiguration {
	if len(data) == 0 || data[0] == nil {
		return nil
	}

	ec := data[0].(map[string]interface{})
	config := &ecr.EncryptionConfiguration{
		EncryptionType: aws.String(ec["encryption_type"].(string)),
	}
	if v, ok := ec["kms_key"]; ok {
		if s := v.(string); s != "" {
			config.KmsKey = aws.String(v.(string))
		}
	}
	return config
}

func flattenRepositoryEncryptionConfiguration(ec *ecr.EncryptionConfiguration) []map[string]interface{} {
	if ec == nil {
		return nil
	}

	config := map[string]interface{}{
		"encryption_type": aws.StringValue(ec.EncryptionType),
		"kms_key":         aws.StringValue(ec.KmsKey),
	}

	return []map[string]interface{}{
		config,
	}
}
