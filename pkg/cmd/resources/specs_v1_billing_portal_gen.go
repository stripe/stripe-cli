// This file is generated; DO NOT EDIT.

package resources

import "github.com/stripe/stripe-cli/pkg/cmd/resource"

var V1BillingPortalConfigurationsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/billing_portal/configurations/{configuration}",
	Method:  "POST",
	Summary: "Update a portal configuration",
	Params: map[string]*resource.ParamSpec{
		"features.subscription_update.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled.",
		},
		"features.subscription_update.proration_behavior": {
			Type:        "string",
			Description: "Determines how to handle prorations resulting from subscription updates. Valid values are `none`, `create_prorations`, and `always_invoice`.",
			Enum: []resource.EnumSpec{
				{Value: "always_invoice"},
				{Value: "create_prorations"},
				{Value: "none"},
			},
		},
		"features.customer_update.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled.",
		},
		"features.payment_method_update.payment_method_configuration": {
			Type:        "string",
			Description: "The [Payment Method Configuration](/api/payment_method_configurations) to use for this portal session. When specified, customers will be able to update their payment method to one of the options specified by the payment method configuration. If not set or set to an empty string, the default payment method configuration is used.",
		},
		"name": {
			Type:        "string",
			Description: "The name of the configuration.",
		},
		"business_profile.terms_of_service_url": {
			Type:        "string",
			Description: "A link to the business’s publicly available terms of service.",
		},
		"features.subscription_cancel.proration_behavior": {
			Type:        "string",
			Description: "Whether to create prorations when canceling subscriptions. Possible values are `none` and `create_prorations`, which is only compatible with `mode=immediately`. Passing `always_invoice` will result in an error. No prorations are generated when canceling a subscription at the end of its natural billing period.",
			Enum: []resource.EnumSpec{
				{Value: "always_invoice"},
				{Value: "create_prorations"},
				{Value: "none"},
			},
		},
		"features.invoice_history.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled.",
			Required:    true,
		},
		"active": {
			Type:        "boolean",
			Description: "Whether the configuration is active and can be used to create portal sessions.",
		},
		"business_profile.privacy_policy_url": {
			Type:        "string",
			Description: "A link to the business’s publicly available privacy policy.",
		},
		"default_return_url": {
			Type:        "string",
			Description: "The default URL to redirect customers to when they click on the portal's link to return to your website. This can be [overriden](https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-return_url) when creating the session.",
		},
		"features.subscription_cancel.cancellation_reason.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled.",
			Required:    true,
		},
		"features.subscription_cancel.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled.",
		},
		"features.subscription_cancel.mode": {
			Type:        "string",
			Description: "Whether to cancel subscriptions immediately or at the end of the billing period.",
			Enum: []resource.EnumSpec{
				{Value: "at_period_end"},
				{Value: "immediately"},
			},
		},
		"features.subscription_update.billing_cycle_anchor": {
			Type:        "string",
			Description: "Determines the value to use for the billing cycle anchor on subscription updates. Valid values are `now` or `unchanged`, and the default value is `unchanged`. Setting the value to `now` resets the subscription's billing cycle anchor to the current time (in UTC). For more information, see the billing cycle [documentation](https://docs.stripe.com/billing/subscriptions/billing-cycle).",
			Enum: []resource.EnumSpec{
				{Value: "now"},
				{Value: "unchanged"},
			},
		},
		"login_page.enabled": {
			Type:        "boolean",
			Description: "Set to `true` to generate a shareable URL [`login_page.url`](https://docs.stripe.com/api/customer_portal/configuration#portal_configuration_object-login_page-url) that will take your customers to a hosted login page for the customer portal.\n\nSet to `false` to deactivate the `login_page.url`.",
			Required:    true,
		},
		"business_profile.headline": {
			Type:        "string",
			Description: "The messaging shown to customers in the portal.",
		},
		"features.subscription_cancel.cancellation_reason.options": {
			Type:        "array",
			Description: "Which cancellation reasons will be given as options to the customer.",
		},
		"features.subscription_update.trial_update_behavior": {
			Type:        "string",
			Description: "The behavior when updating a subscription that is trialing.",
			Enum: []resource.EnumSpec{
				{Value: "continue_trial"},
				{Value: "end_trial"},
			},
		},
		"features.customer_update.allowed_updates": {
			Type:        "array",
			Description: "The types of customer updates that are supported. When empty, customers are not updateable.",
		},
		"features.payment_method_update.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled.",
			Required:    true,
		},
		"features.subscription_update.default_allowed_updates": {
			Type:        "array",
			Description: "The types of subscription updates that are supported. When empty, subscriptions are not updateable.",
		},
	},
}

var V1BillingPortalConfigurationsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/billing_portal/configurations",
	Method:  "GET",
	Summary: "List portal configurations",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"active": {
			Type:        "boolean",
			Description: "Only return configurations that are active or inactive (e.g., pass `true` to only list active configurations).",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"is_default": {
			Type:        "boolean",
			Description: "Only return the default or non-default configurations (e.g., pass `true` to only list the default configuration).",
		},
	},
}

var V1BillingPortalConfigurationsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/billing_portal/configurations/{configuration}",
	Method:  "GET",
	Summary: "Retrieve a portal configuration",
}

var V1BillingPortalConfigurationsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/billing_portal/configurations",
	Method:  "POST",
	Summary: "Create a portal configuration",
	Params: map[string]*resource.ParamSpec{
		"features.subscription_cancel.cancellation_reason.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled.",
			Required:    true,
		},
		"features.subscription_cancel.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled.",
			Required:    true,
		},
		"features.customer_update.allowed_updates": {
			Type:        "array",
			Description: "The types of customer updates that are supported. When empty, customers are not updateable.",
		},
		"features.customer_update.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled.",
			Required:    true,
		},
		"features.invoice_history.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled.",
			Required:    true,
		},
		"name": {
			Type:        "string",
			Description: "The name of the configuration.",
		},
		"business_profile.terms_of_service_url": {
			Type:        "string",
			Description: "A link to the business’s publicly available terms of service.",
		},
		"features.subscription_update.proration_behavior": {
			Type:        "string",
			Description: "Determines how to handle prorations resulting from subscription updates. Valid values are `none`, `create_prorations`, and `always_invoice`.",
			Enum: []resource.EnumSpec{
				{Value: "always_invoice"},
				{Value: "create_prorations"},
				{Value: "none"},
			},
		},
		"features.subscription_cancel.cancellation_reason.options": {
			Type:        "array",
			Description: "Which cancellation reasons will be given as options to the customer.",
			Required:    true,
		},
		"features.subscription_cancel.mode": {
			Type:        "string",
			Description: "Whether to cancel subscriptions immediately or at the end of the billing period.",
			Enum: []resource.EnumSpec{
				{Value: "at_period_end"},
				{Value: "immediately"},
			},
		},
		"business_profile.privacy_policy_url": {
			Type:        "string",
			Description: "A link to the business’s publicly available privacy policy.",
		},
		"business_profile.headline": {
			Type:        "string",
			Description: "The messaging shown to customers in the portal.",
		},
		"features.payment_method_update.payment_method_configuration": {
			Type:        "string",
			Description: "The [Payment Method Configuration](/api/payment_method_configurations) to use for this portal session. When specified, customers will be able to update their payment method to one of the options specified by the payment method configuration. If not set or set to an empty string, the default payment method configuration is used.",
		},
		"features.subscription_cancel.proration_behavior": {
			Type:        "string",
			Description: "Whether to create prorations when canceling subscriptions. Possible values are `none` and `create_prorations`, which is only compatible with `mode=immediately`. Passing `always_invoice` will result in an error. No prorations are generated when canceling a subscription at the end of its natural billing period.",
			Enum: []resource.EnumSpec{
				{Value: "always_invoice"},
				{Value: "create_prorations"},
				{Value: "none"},
			},
		},
		"features.subscription_update.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled.",
			Required:    true,
		},
		"login_page.enabled": {
			Type:        "boolean",
			Description: "Set to `true` to generate a shareable URL [`login_page.url`](https://docs.stripe.com/api/customer_portal/configuration#portal_configuration_object-login_page-url) that will take your customers to a hosted login page for the customer portal.",
			Required:    true,
		},
		"default_return_url": {
			Type:        "string",
			Description: "The default URL to redirect customers to when they click on the portal's link to return to your website. This can be [overriden](https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-return_url) when creating the session.",
		},
		"features.subscription_update.trial_update_behavior": {
			Type:        "string",
			Description: "The behavior when updating a subscription that is trialing.",
			Enum: []resource.EnumSpec{
				{Value: "continue_trial"},
				{Value: "end_trial"},
			},
		},
		"features.subscription_update.billing_cycle_anchor": {
			Type:        "string",
			Description: "Determines the value to use for the billing cycle anchor on subscription updates. Valid values are `now` or `unchanged`, and the default value is `unchanged`. Setting the value to `now` resets the subscription's billing cycle anchor to the current time (in UTC). For more information, see the billing cycle [documentation](https://docs.stripe.com/billing/subscriptions/billing-cycle).",
			Enum: []resource.EnumSpec{
				{Value: "now"},
				{Value: "unchanged"},
			},
		},
		"features.subscription_update.default_allowed_updates": {
			Type:        "array",
			Description: "The types of subscription updates that are supported. When empty, subscriptions are not updateable.",
		},
		"features.payment_method_update.enabled": {
			Type:        "boolean",
			Description: "Whether the feature is enabled.",
			Required:    true,
		},
	},
}

var V1BillingPortalSessionsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/billing_portal/sessions",
	Method:  "POST",
	Summary: "Create a portal session",
	Params: map[string]*resource.ParamSpec{
		"on_behalf_of": {
			Type:        "string",
			Description: "The `on_behalf_of` account to use for this session. When specified, only subscriptions and invoices with this `on_behalf_of` account appear in the portal. For more information, see the [docs](https://docs.stripe.com/connect/separate-charges-and-transfers#settlement-merchant). Use the [Accounts API](https://docs.stripe.com/api/accounts/object#account_object-settings-branding) to modify the `on_behalf_of` account's branding settings, which the portal displays.",
		},
		"return_url": {
			Type:        "string",
			Description: "The default URL to redirect customers to when they click on the portal's link to return to your website.",
		},
		"flow_data.subscription_update_confirm.subscription": {
			Type:        "string",
			Description: "The ID of the subscription to be updated.",
			Required:    true,
		},
		"configuration": {
			Type:        "string",
			Description: "The ID of an existing [configuration](https://docs.stripe.com/api/customer_portal/configurations) to use for this session, describing its functionality and features. If not specified, the session uses the default configuration.",
		},
		"customer": {
			Type:        "string",
			Description: "The ID of an existing customer.",
		},
		"flow_data.after_completion.type": {
			Type:        "string",
			Description: "The specified behavior after the flow is completed.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "hosted_confirmation"},
				{Value: "portal_homepage"},
				{Value: "redirect"},
			},
		},
		"flow_data.after_completion.hosted_confirmation.custom_message": {
			Type:        "string",
			Description: "A custom message to display to the customer after the flow is completed.",
		},
		"flow_data.subscription_cancel.subscription": {
			Type:        "string",
			Description: "The ID of the subscription to be canceled.",
			Required:    true,
		},
		"flow_data.type": {
			Type:        "string",
			Description: "Type of flow that the customer will go through.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "payment_method_update"},
				{Value: "subscription_cancel"},
				{Value: "subscription_update"},
				{Value: "subscription_update_confirm"},
			},
		},
		"locale": {
			Type:        "string",
			Description: "The IETF language tag of the locale customer portal is displayed in. If blank or auto, the customer’s `preferred_locales` or browser’s locale is used.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "bg"},
				{Value: "cs"},
				{Value: "da"},
				{Value: "de"},
				{Value: "el"},
				{Value: "en"},
				{Value: "en-AU"},
				{Value: "en-CA"},
				{Value: "en-GB"},
				{Value: "en-IE"},
				{Value: "en-IN"},
				{Value: "en-NZ"},
				{Value: "en-SG"},
				{Value: "es"},
				{Value: "es-419"},
				{Value: "et"},
				{Value: "fi"},
				{Value: "fil"},
				{Value: "fr"},
				{Value: "fr-CA"},
				{Value: "hr"},
				{Value: "hu"},
				{Value: "id"},
				{Value: "it"},
				{Value: "ja"},
				{Value: "ko"},
				{Value: "lt"},
				{Value: "lv"},
				{Value: "ms"},
				{Value: "mt"},
				{Value: "nb"},
				{Value: "nl"},
				{Value: "pl"},
				{Value: "pt"},
				{Value: "pt-BR"},
				{Value: "ro"},
				{Value: "ru"},
				{Value: "sk"},
				{Value: "sl"},
				{Value: "sv"},
				{Value: "th"},
				{Value: "tr"},
				{Value: "vi"},
				{Value: "zh"},
				{Value: "zh-HK"},
				{Value: "zh-TW"},
			},
		},
		"customer_account": {
			Type:        "string",
			Description: "The ID of an existing account.",
		},
		"flow_data.after_completion.redirect.return_url": {
			Type:        "string",
			Description: "The URL the customer will be redirected to after the flow is completed.",
			Required:    true,
		},
		"flow_data.subscription_cancel.retention.coupon_offer.coupon": {
			Type:        "string",
			Description: "The ID of the coupon to be offered.",
			Required:    true,
		},
		"flow_data.subscription_cancel.retention.type": {
			Type:        "string",
			Description: "Type of retention strategy to use with the customer.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "coupon_offer"},
			},
		},
		"flow_data.subscription_update.subscription": {
			Type:        "string",
			Description: "The ID of the subscription to be updated.",
			Required:    true,
		},
	},
}
