// This file is generated; DO NOT EDIT.

package resources

import "github.com/stripe/stripe-cli/pkg/cmd/resource"

var V1IdentityVerificationSessionsCreate = resource.OperationSpec{
	Name:    "create",
	Path:    "/v1/identity/verification_sessions",
	Method:  "POST",
	Summary: "Create a VerificationSession",
	Params: map[string]*resource.ParamSpec{
		"related_customer_account": {
			Type:        "string",
			Description: "The ID of the Account representing a customer.",
		},
		"type": {
			Type:        "string",
			Description: "The type of [verification check](https://docs.stripe.com/identity/verification-checks) to be performed. You must provide a `type` if not passing `verification_flow`.",
			Enum: []resource.EnumSpec{
				{Value: "document"},
				{Value: "id_number"},
			},
		},
		"verification_flow": {
			Type:        "string",
			Description: "The ID of a verification flow from the Dashboard. See https://docs.stripe.com/identity/verification-flows.",
		},
		"provided_details.email": {
			Type:        "string",
			Description: "Email of user being verified",
		},
		"provided_details.phone": {
			Type:        "string",
			Description: "Phone number of user being verified",
		},
		"return_url": {
			Type:        "string",
			Description: "The URL that the user will be redirected to upon completing the verification flow.",
		},
		"related_person.account": {
			Type:        "string",
			Description: "A token representing a connected account. If provided, the person parameter is also required and must be associated with the account.",
			Required:    true,
		},
		"related_person.person": {
			Type:        "string",
			Description: "A token referencing a Person resource that this verification is being used to verify.",
			Required:    true,
		},
		"client_reference_id": {
			Type:        "string",
			Description: "A string to reference this user. This can be a customer ID, a session ID, or similar, and can be used to reconcile this verification with your internal systems.",
		},
		"related_customer": {
			Type:        "string",
			Description: "Customer ID",
		},
	},
}

var V1IdentityVerificationSessionsUpdate = resource.OperationSpec{
	Name:    "update",
	Path:    "/v1/identity/verification_sessions/{session}",
	Method:  "POST",
	Summary: "Update a VerificationSession",
	Params: map[string]*resource.ParamSpec{
		"provided_details.email": {
			Type:        "string",
			Description: "Email of user being verified",
		},
		"provided_details.phone": {
			Type:        "string",
			Description: "Phone number of user being verified",
		},
		"type": {
			Type:        "string",
			Description: "The type of [verification check](https://docs.stripe.com/identity/verification-checks) to be performed.",
			Enum: []resource.EnumSpec{
				{Value: "document"},
				{Value: "id_number"},
			},
		},
	},
}

var V1IdentityVerificationSessionsCancel = resource.OperationSpec{
	Name:    "cancel",
	Path:    "/v1/identity/verification_sessions/{session}/cancel",
	Method:  "POST",
	Summary: "Cancel a VerificationSession",
}

var V1IdentityVerificationSessionsRedact = resource.OperationSpec{
	Name:    "redact",
	Path:    "/v1/identity/verification_sessions/{session}/redact",
	Method:  "POST",
	Summary: "Redact a VerificationSession",
}

var V1IdentityVerificationSessionsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/identity/verification_sessions",
	Method:  "GET",
	Summary: "List VerificationSessions",
	Params: map[string]*resource.ParamSpec{
		"related_customer_account": {
			Type:        "string",
			Description: "The ID of the Account representing a customer.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"status": {
			Type:        "string",
			Description: "Only return VerificationSessions with this status. [Learn more about the lifecycle of sessions](https://docs.stripe.com/identity/how-sessions-work).",
			Enum: []resource.EnumSpec{
				{Value: "canceled"},
				{Value: "processing"},
				{Value: "requires_input"},
				{Value: "verified"},
			},
		},
		"client_reference_id": {
			Type:        "string",
			Description: "A string to reference this user. This can be a customer ID, a session ID, or similar, and can be used to reconcile this verification with your internal systems.",
		},
		"created": {
			Type:        "integer",
			Description: "Only return VerificationSessions that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"related_customer": {
			Type:        "string",
			Description: "Customer ID",
		},
	},
}

var V1IdentityVerificationSessionsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/identity/verification_sessions/{session}",
	Method:  "GET",
	Summary: "Retrieve a VerificationSession",
}

var V1IdentityVerificationReportsList = resource.OperationSpec{
	Name:    "list",
	Path:    "/v1/identity/verification_reports",
	Method:  "GET",
	Summary: "List VerificationReports",
	Params: map[string]*resource.ParamSpec{
		"created": {
			Type:        "integer",
			Description: "Only return VerificationReports that were created during the given date interval.",
		},
		"ending_before": {
			Type:        "string",
			Description: "A cursor for use in pagination. `ending_before` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, starting with `obj_bar`, your subsequent call can include `ending_before=obj_bar` in order to fetch the previous page of the list.",
		},
		"limit": {
			Type:        "integer",
			Description: "A limit on the number of objects to be returned. Limit can range between 1 and 100, and the default is 10.",
		},
		"starting_after": {
			Type:        "string",
			Description: "A cursor for use in pagination. `starting_after` is an object ID that defines your place in the list. For instance, if you make a list request and receive 100 objects, ending with `obj_foo`, your subsequent call can include `starting_after=obj_foo` in order to fetch the next page of the list.",
		},
		"type": {
			Type:        "string",
			Description: "Only return VerificationReports of this type",
			Enum: []resource.EnumSpec{
				{Value: "document"},
				{Value: "id_number"},
			},
		},
		"verification_session": {
			Type:        "string",
			Description: "Only return VerificationReports created by this VerificationSession ID. It is allowed to provide a VerificationIntent ID.",
		},
		"client_reference_id": {
			Type:        "string",
			Description: "A string to reference this user. This can be a customer ID, a session ID, or similar, and can be used to reconcile this verification with your internal systems.",
		},
	},
}

var V1IdentityVerificationReportsRetrieve = resource.OperationSpec{
	Name:    "retrieve",
	Path:    "/v1/identity/verification_reports/{report}",
	Method:  "GET",
	Summary: "Retrieve a VerificationReport",
}
