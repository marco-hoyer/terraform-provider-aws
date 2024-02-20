// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configservice

import (
	"time"
)

const (
	propagationTimeout = 2 * time.Minute
)

const (
	defaultConfigurationRecorderName = "default"
	defaultDeliveryChannelName       = "default"
)

const (
	ResNameAggregateAuthorization      = "Aggregate Authorization"
	ResNameConfigurationAggregator     = "Configuration Aggregator"
	ResNameConfigurationRecorderStatus = "Configuration Recorder Status"
	ResNameConfigurationRecorder       = "Configuration Recorder"
	ResNameDeliveryChannel             = "Delivery Channel"
	ResNameOrganizationManagedRule     = "Organization Managed Rule"
	ResNameOrganizationCustomRule      = "Organization Custom Rule"
	ResNameRemediationConfiguration    = "Remediation Configuration"
)
