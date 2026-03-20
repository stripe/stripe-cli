// This file is generated; DO NOT EDIT.

package resources

import "github.com/stripe/stripe-cli/pkg/cmd/resource"

var V2PreviewCoreAccountPersonsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/core/accounts/{account_id}/persons/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve a Person",
}

var V2PreviewCoreAccountPersonsUpdate = resource.OperationSpec{
	Name:      "update",
	Path:      "/v2/core/accounts/{account_id}/persons/{id}",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Update a person",
	Params: map[string]*resource.ParamSpec{
		"relationship.authorizer": {
			Type:        "boolean",
			Description: "Whether the individual is an authorizer of the Account's identity.",
		},
		"nationalities": {
			Type:        "array",
			Description: "The nationalities (countries) this person is associated with.",
		},
		"script_addresses.kana.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"surname": {
			Type:        "string",
			Description: "The person's last name.",
		},
		"script_addresses.kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"relationship.percent_ownership": {
			Type:        "string",
			Description: "The percentage of ownership the person has in the associated legal entity.",
			Format:      "decimal",
		},
		"additional_terms_of_service.account.ip": {
			Type:        "string",
			Description: "The IP address from which the Account's representative accepted the terms of service.",
		},
		"documents.visa.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"relationship.director": {
			Type:        "boolean",
			Description: "Indicates whether the person is a director of the associated legal entity.",
		},
		"script_addresses.kanji.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"script_addresses.kanji.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"script_addresses.kana.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"documents.company_authorization.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"documents.primary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"script_names.kanji.given_name": {
			Type:        "string",
			Description: "The person's first or given name.",
		},
		"script_addresses.kanji.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"script_addresses.kana.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"documents.primary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"address.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"relationship.representative": {
			Type:        "boolean",
			Description: "Indicates whether the person is a representative of the associated legal entity.",
		},
		"script_addresses.kanji.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"script_addresses.kana.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"date_of_birth.day": {
			Type:        "integer",
			Description: "The day of the birth.",
			Required:    true,
		},
		"documents.primary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"documents.visa.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"script_names.kana.surname": {
			Type:        "string",
			Description: "The person's last or family name.",
		},
		"script_names.kanji.surname": {
			Type:        "string",
			Description: "The person's last or family name.",
		},
		"script_addresses.kanji.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"script_addresses.kana.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"relationship.title": {
			Type:        "string",
			Description: "The title or position the person holds in the associated legal entity.",
		},
		"script_addresses.kana.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"political_exposure": {
			Type:        "string",
			Description: "The person's political exposure.",
			Enum: []resource.EnumSpec{
				{Value: "existing"},
				{Value: "none"},
			},
		},
		"relationship.executive": {
			Type:        "boolean",
			Description: "Indicates whether the person is an executive of the associated legal entity.",
		},
		"relationship.legal_guardian": {
			Type:        "boolean",
			Description: "Indicates whether the person is a legal guardian of the associated legal entity.",
		},
		"date_of_birth.year": {
			Type:        "integer",
			Description: "The year of birth.",
			Required:    true,
		},
		"relationship.owner": {
			Type:        "boolean",
			Description: "Indicates whether the person is an owner of the associated legal entity.",
		},
		"date_of_birth.month": {
			Type:        "integer",
			Description: "The month of birth.",
			Required:    true,
		},
		"documents.passport.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"script_names.kana.given_name": {
			Type:        "string",
			Description: "The person's first or given name.",
		},
		"given_name": {
			Type:        "string",
			Description: "The person's first name.",
		},
		"script_addresses.kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"phone": {
			Type:        "string",
			Description: "The phone number for this person.",
		},
		"documents.company_authorization.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"legal_gender": {
			Type:        "string",
			Description: "The person's gender (International regulations require either \"male\" or \"female\").",
			Enum: []resource.EnumSpec{
				{Value: "female"},
				{Value: "male"},
			},
		},
		"person_token": {
			Type:        "string",
			Description: "The person token generated by the person token api.",
		},
		"documents.secondary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"documents.passport.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"script_addresses.kanji.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"additional_terms_of_service.account.date": {
			Type:        "string",
			Description: "The time when the Account's representative accepted the terms of service. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Format:      "date-time",
		},
		"documents.secondary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"documents.secondary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"email": {
			Type:        "string",
			Description: "Email.",
		},
		"address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"additional_terms_of_service.account.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the Account's representative accepted the terms of service.",
		},
	},
}

var V2PreviewCoreAccountPersonsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/core/accounts/{account_id}/persons",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Persons",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "The upper limit on the number of accounts returned by the List Account request.",
		},
		"page": {
			Type:        "string",
			Description: "The page token to navigate to next or previous batch of accounts given by the list request.",
		},
	},
}

var V2PreviewCoreAccountPersonsCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/core/accounts/{account_id}/persons",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create a person",
	Params: map[string]*resource.ParamSpec{
		"script_addresses.kana.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"script_names.kana.given_name": {
			Type:        "string",
			Description: "The person's first or given name.",
		},
		"documents.secondary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"surname": {
			Type:        "string",
			Description: "The person's last name.",
		},
		"political_exposure": {
			Type:        "string",
			Description: "The person's political exposure.",
			Enum: []resource.EnumSpec{
				{Value: "existing"},
				{Value: "none"},
			},
		},
		"script_addresses.kanji.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"documents.company_authorization.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"documents.visa.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"additional_terms_of_service.account.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the Account's representative accepted the terms of service.",
		},
		"script_addresses.kanji.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"documents.passport.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"script_addresses.kanji.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"address.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"documents.secondary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"relationship.percent_ownership": {
			Type:        "string",
			Description: "The percentage of ownership the person has in the associated legal entity.",
			Format:      "decimal",
		},
		"address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
			Required:    true,
		},
		"script_names.kanji.given_name": {
			Type:        "string",
			Description: "The person's first or given name.",
		},
		"additional_terms_of_service.account.date": {
			Type:        "string",
			Description: "The time when the Account's representative accepted the terms of service. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Required:    true,
			Format:      "date-time",
		},
		"relationship.owner": {
			Type:        "boolean",
			Description: "Indicates whether the person is an owner of the associated legal entity.",
		},
		"relationship.authorizer": {
			Type:        "boolean",
			Description: "Whether the individual is an authorizer of the Account's identity.",
		},
		"script_addresses.kana.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"script_addresses.kana.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"date_of_birth.month": {
			Type:        "integer",
			Description: "The month of birth.",
			Required:    true,
		},
		"documents.primary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
			Required:    true,
		},
		"relationship.director": {
			Type:        "boolean",
			Description: "Indicates whether the person is a director of the associated legal entity.",
		},
		"script_addresses.kana.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"documents.passport.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"given_name": {
			Type:        "string",
			Description: "The person's first name.",
		},
		"relationship.title": {
			Type:        "string",
			Description: "The title or position the person holds in the associated legal entity.",
		},
		"script_addresses.kana.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"script_names.kana.surname": {
			Type:        "string",
			Description: "The person's last or family name.",
		},
		"nationalities": {
			Type:        "array",
			Description: "The nationalities (countries) this person is associated with.",
		},
		"script_addresses.kanji.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"documents.visa.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"phone": {
			Type:        "string",
			Description: "The phone number for this person.",
		},
		"date_of_birth.year": {
			Type:        "integer",
			Description: "The year of birth.",
			Required:    true,
		},
		"additional_terms_of_service.account.ip": {
			Type:        "string",
			Description: "The IP address from which the Account's representative accepted the terms of service.",
			Required:    true,
		},
		"script_addresses.kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
			Required:    true,
		},
		"person_token": {
			Type:        "string",
			Description: "The person token generated by the person token api.",
		},
		"script_addresses.kanji.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"documents.primary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"script_addresses.kana.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"script_addresses.kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
			Required:    true,
		},
		"documents.primary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"documents.secondary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
			Required:    true,
		},
		"legal_gender": {
			Type:        "string",
			Description: "The person's gender (International regulations require either \"male\" or \"female\").",
			Enum: []resource.EnumSpec{
				{Value: "female"},
				{Value: "male"},
			},
		},
		"relationship.representative": {
			Type:        "boolean",
			Description: "Indicates whether the person is a representative of the associated legal entity.",
		},
		"documents.company_authorization.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"relationship.legal_guardian": {
			Type:        "boolean",
			Description: "Indicates whether the person is a legal guardian of the associated legal entity.",
		},
		"script_addresses.kanji.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"date_of_birth.day": {
			Type:        "integer",
			Description: "The day of birth.",
			Required:    true,
		},
		"address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"script_names.kanji.surname": {
			Type:        "string",
			Description: "The person's last or family name.",
		},
		"email": {
			Type:        "string",
			Description: "Email.",
		},
		"relationship.executive": {
			Type:        "boolean",
			Description: "Indicates whether the person is an executive of the associated legal entity.",
		},
	},
}

var V2PreviewCoreAccountPersonsDelete = resource.OperationSpec{
	Name:      "delete",
	Path:      "/v2/core/accounts/{account_id}/persons/{id}",
	Method:    "DELETE",
	IsPreview: true,
	Summary:   "Delete a Person",
}

var V2PreviewCoreAccountPersonTokensCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/core/accounts/{account_id}/person_tokens",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create a Person Token",
	Params: map[string]*resource.ParamSpec{
		"address.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"date_of_birth.year": {
			Type:        "integer",
			Description: "The year of birth.",
			Required:    true,
		},
		"phone": {
			Type:        "string",
			Description: "The phone number for this person.",
		},
		"documents.visa.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"script_addresses.kana.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"script_addresses.kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"documents.company_authorization.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"documents.passport.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"documents.primary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"documents.primary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"script_addresses.kana.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"script_addresses.kana.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"legal_gender": {
			Type:        "string",
			Description: "The person's gender (International regulations require either \"male\" or \"female\").",
			Enum: []resource.EnumSpec{
				{Value: "female"},
				{Value: "male"},
			},
		},
		"documents.primary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"documents.secondary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"political_exposure": {
			Type:        "string",
			Description: "The person's political exposure.",
			Enum: []resource.EnumSpec{
				{Value: "existing"},
				{Value: "none"},
			},
		},
		"script_names.kanji.given_name": {
			Type:        "string",
			Description: "The person's first or given name.",
		},
		"script_names.kanji.surname": {
			Type:        "string",
			Description: "The person's last or family name.",
		},
		"script_addresses.kana.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"script_addresses.kana.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"documents.visa.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"email": {
			Type:        "string",
			Description: "Email.",
		},
		"script_addresses.kana.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"script_addresses.kanji.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"script_addresses.kanji.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"script_addresses.kanji.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"date_of_birth.month": {
			Type:        "integer",
			Description: "The month of birth.",
			Required:    true,
		},
		"relationship.representative": {
			Type:        "boolean",
			Description: "Indicates whether the person is a representative of the associated legal entity.",
		},
		"relationship.title": {
			Type:        "string",
			Description: "The title or position the person holds in the associated legal entity.",
		},
		"relationship.authorizer": {
			Type:        "boolean",
			Description: "Whether the individual is an authorizer of the Account's identity.",
		},
		"relationship.director": {
			Type:        "boolean",
			Description: "Indicates whether the person is a director of the associated legal entity.",
		},
		"documents.secondary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"script_names.kana.given_name": {
			Type:        "string",
			Description: "The person's first or given name.",
		},
		"script_names.kana.surname": {
			Type:        "string",
			Description: "The person's last or family name.",
		},
		"nationalities": {
			Type:        "array",
			Description: "The nationalities (countries) this person is associated with.",
		},
		"surname": {
			Type:        "string",
			Description: "The person's last name.",
		},
		"script_addresses.kanji.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"script_addresses.kanji.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"date_of_birth.day": {
			Type:        "integer",
			Description: "The day of the birth.",
			Required:    true,
		},
		"given_name": {
			Type:        "string",
			Description: "The person's first name.",
		},
		"relationship.legal_guardian": {
			Type:        "boolean",
			Description: "Indicates whether the person is a legal guardian of the associated legal entity.",
		},
		"relationship.executive": {
			Type:        "boolean",
			Description: "Indicates whether the person is an executive of the associated legal entity.",
		},
		"documents.company_authorization.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"documents.secondary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"script_addresses.kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"relationship.owner": {
			Type:        "boolean",
			Description: "Indicates whether the person is an owner of the associated legal entity.",
		},
		"relationship.percent_ownership": {
			Type:        "string",
			Description: "The percentage of ownership the person has in the associated legal entity.",
			Format:      "decimal",
		},
		"additional_terms_of_service.account.shown_and_accepted": {
			Type:        "boolean",
			Description: "The boolean value indicating if the terms of service have been accepted.",
		},
		"documents.passport.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"script_addresses.kanji.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
	},
}

var V2PreviewCoreAccountPersonTokensRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/core/accounts/{account_id}/person_tokens/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve a Person Token",
}

var V2PreviewCoreVaultsGbBankAccountsCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/core/vault/gb_bank_accounts",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create a GB Bank Account",
	Params: map[string]*resource.ParamSpec{
		"account_number": {
			Type:        "string",
			Description: "The Account Number of the bank account.",
			Required:    true,
		},
		"bank_account_type": {
			Type:        "string",
			Description: "Closed Enum. The type of the bank account (checking or savings).",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"confirmation_of_payee.business_type": {
			Type:        "string",
			Description: "The business type to be checked against. Legal entity information will be used if unspecified.\nClosed enum.",
			Enum: []resource.EnumSpec{
				{Value: "business"},
				{Value: "personal"},
			},
		},
		"confirmation_of_payee.initiate": {
			Type:        "boolean",
			Description: "User specifies whether Confirmation of Payee is automatically initiated when creating the bank account.",
			Required:    true,
		},
		"confirmation_of_payee.name": {
			Type:        "string",
			Description: "The name to be checked against. Legal entity information will be used if unspecified.",
		},
		"sort_code": {
			Type:        "string",
			Description: "The Sort Code of the bank account.",
			Required:    true,
		},
	},
}

var V2PreviewCoreVaultsGbBankAccountsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/core/vault/gb_bank_accounts/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve a GB Bank Account",
}

var V2PreviewCoreVaultsGbBankAccountsAcknowledgeConfirmationOfPayee = resource.OperationSpec{
	Name:      "acknowledge_confirmation_of_payee",
	Path:      "/v2/core/vault/gb_bank_accounts/{id}/acknowledge_confirmation_of_payee",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Acknowledge Confirmation of Payee (CoP)",
}

var V2PreviewCoreVaultsGbBankAccountsArchive = resource.OperationSpec{
	Name:      "archive",
	Path:      "/v2/core/vault/gb_bank_accounts/{id}/archive",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Archive a GB Bank Account",
}

var V2PreviewCoreVaultsGbBankAccountsInitiateConfirmationOfPayee = resource.OperationSpec{
	Name:      "initiate_confirmation_of_payee",
	Path:      "/v2/core/vault/gb_bank_accounts/{id}/initiate_confirmation_of_payee",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Initiate Confirmation of Payee (CoP)",
	Params: map[string]*resource.ParamSpec{
		"business_type": {
			Type:        "string",
			Description: "The business type to be checked against. Legal entity information will be used if unspecified.",
			Enum: []resource.EnumSpec{
				{Value: "business"},
				{Value: "personal"},
			},
		},
		"name": {
			Type:        "string",
			Description: "The name of the user to be checked against. Legal entity information will be used if unspecified.",
		},
	},
}

var V2PreviewCoreVaultsGbBankAccountsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/core/vault/gb_bank_accounts",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List GB Bank Accounts",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "Optionally set the maximum number of results per page. Defaults to 10.",
		},
		"page": {
			Type:        "string",
			Description: "The pagination token.",
		},
	},
}

var V2PreviewCoreVaultsUsBankAccountsUpdate = resource.OperationSpec{
	Name:      "update",
	Path:      "/v2/core/vault/us_bank_accounts/{id}",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Update a US Bank Account",
	Params: map[string]*resource.ParamSpec{
		"fedwire_routing_number": {
			Type:        "string",
			Description: "The bank account's Fedwire routing number can be provided for update if it was empty previously.",
		},
		"routing_number": {
			Type:        "string",
			Description: "The bank account's ACH routing number can be provided for update if it was empty previously.",
		},
	},
}

var V2PreviewCoreVaultsUsBankAccountsArchive = resource.OperationSpec{
	Name:      "archive",
	Path:      "/v2/core/vault/us_bank_accounts/{id}/archive",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Archive a US Bank Account",
}

var V2PreviewCoreVaultsUsBankAccountsConfirmMicrodeposits = resource.OperationSpec{
	Name:      "confirm_microdeposits",
	Path:      "/v2/core/vault/us_bank_accounts/{id}/confirm_microdeposits",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Confirm Microdeposits for verification",
	Params: map[string]*resource.ParamSpec{
		"amounts": {
			Type:        "array",
			Description: "Two amounts received through Send Microdeposits must match the input to Confirm Microdeposits to verify US Bank Account.",
		},
		"descriptor_code": {
			Type:        "string",
			Description: "Descriptor code received through Send Microdeposits must match the input to Confirm Microdeposits to verify US Bank Account.",
		},
	},
}

var V2PreviewCoreVaultsUsBankAccountsSendMicrodeposits = resource.OperationSpec{
	Name:      "send_microdeposits",
	Path:      "/v2/core/vault/us_bank_accounts/{id}/send_microdeposits",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Send Microdeposits for verification",
}

var V2PreviewCoreVaultsUsBankAccountsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/core/vault/us_bank_accounts",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List US Bank Accounts",
	Params: map[string]*resource.ParamSpec{
		"verification_status": {
			Type:        "string",
			Description: "Optionally filter by verification status. Mutually exclusive with `unverified`, `verified`, `awaiting_verification`, and `verification_failed`.",
		},
		"limit": {
			Type:        "integer",
			Description: "Optionally set the maximum number of results per page. Defaults to 10.",
		},
		"page": {
			Type:        "string",
			Description: "The pagination token.",
		},
	},
}

var V2PreviewCoreVaultsUsBankAccountsCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/core/vault/us_bank_accounts",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create a US Bank Account",
	Params: map[string]*resource.ParamSpec{
		"bank_account_type": {
			Type:        "string",
			Description: "Closed Enum. The type of the bank account (checking or savings).",
			Enum: []resource.EnumSpec{
				{Value: "checking"},
				{Value: "savings"},
			},
		},
		"fedwire_routing_number": {
			Type:        "string",
			Description: "The fedwire routing number of the bank account. Note that certain banks have the same ACH and wire routing number.",
		},
		"routing_number": {
			Type:        "string",
			Description: "The ACH routing number of the bank account. Note that certain banks have the same ACH and wire routing number.",
		},
		"account_number": {
			Type:        "string",
			Description: "The account number of the bank account.",
			Required:    true,
		},
	},
}

var V2PreviewCoreVaultsUsBankAccountsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/core/vault/us_bank_accounts/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve a US Bank Account",
}

var V2PreviewCoreAccountLinksCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/core/account_links",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create an Account Link",
	Params: map[string]*resource.ParamSpec{
		"use_case.account_onboarding.collection_options.fields": {
			Type:        "string",
			Description: "Specifies whether the platform collects only currently_due requirements (`currently_due`) or both currently_due and eventually_due requirements (`eventually_due`). If you don’t specify collection_options, the default value is currently_due.",
			Enum: []resource.EnumSpec{
				{Value: "currently_due"},
				{Value: "eventually_due"},
			},
		},
		"use_case.account_onboarding.collection_options.future_requirements": {
			Type:        "string",
			Description: "Specifies whether the platform collects future_requirements in addition to requirements in Connect Onboarding. The default value is `omit`.",
			Enum: []resource.EnumSpec{
				{Value: "include"},
				{Value: "omit"},
			},
		},
		"use_case.account_onboarding.configurations": {
			Type:        "array",
			Description: "Open Enum. A v2/core/account can be configured to enable certain functionality. The configuration param targets the v2/core/account_link to collect information for the specified v2/core/account configuration/s.",
			Required:    true,
		},
		"use_case.account_update.collection_options.fields": {
			Type:        "string",
			Description: "Specifies whether the platform collects only currently_due requirements (`currently_due`) or both currently_due and eventually_due requirements (`eventually_due`). The default value is `currently_due`.",
			Enum: []resource.EnumSpec{
				{Value: "currently_due"},
				{Value: "eventually_due"},
			},
		},
		"use_case.account_update.collection_options.future_requirements": {
			Type:        "string",
			Description: "Specifies whether the platform collects future_requirements in addition to requirements in Connect Onboarding. The default value is `omit`.",
			Enum: []resource.EnumSpec{
				{Value: "include"},
				{Value: "omit"},
			},
		},
		"use_case.account_update.configurations": {
			Type:        "array",
			Description: "Open Enum. A v2/account can be configured to enable certain functionality. The configuration param targets the v2/account_link to collect information for the specified v2/account configuration/s.",
			Required:    true,
		},
		"use_case.account_update.return_url": {
			Type:        "string",
			Description: "The URL that the user will be redirected to upon completing the linked flow.",
		},
		"use_case.type": {
			Type:        "string",
			Description: "Open Enum. The type of Account Link the user is requesting.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "account_onboarding"},
				{Value: "account_update"},
			},
		},
		"account": {
			Type:        "string",
			Description: "The ID of the Account to create link for.",
			Required:    true,
		},
		"use_case.account_onboarding.refresh_url": {
			Type:        "string",
			Description: "The URL the user will be redirected to if the AccountLink is expired, has been used, or is otherwise invalid. The URL you specify should attempt to generate a new AccountLink with the same parameters used to create the original AccountLink, then redirect the user to the new AccountLink’s URL so they can continue the flow. If a new AccountLink cannot be generated or the redirect fails you should display a useful error to the user. Please make sure to implement authentication before redirecting the user in case this URL is leaked to a third party.",
			Required:    true,
		},
		"use_case.account_onboarding.return_url": {
			Type:        "string",
			Description: "The URL that the user will be redirected to upon completing the linked flow.",
		},
		"use_case.account_update.refresh_url": {
			Type:        "string",
			Description: "The URL the user will be redirected to if the Account Link is expired, has been used, or is otherwise invalid. The URL you specify should attempt to generate a new Account Link with the same parameters used to create the original Account Link, then redirect the user to the new Account Link URL so they can continue the flow. Make sure to authenticate the user before redirecting to the new Account Link, in case the URL leaks to a third party. If a new Account Link can't be generated, or if the redirect fails, you should display a useful error to the user.",
			Required:    true,
		},
	},
}

var V2PreviewCoreAccountsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/core/accounts/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve an Account",
	Params: map[string]*resource.ParamSpec{
		"include": {
			Type:        "array",
			Description: "Additional fields to include in the response.",
		},
	},
}

var V2PreviewCoreAccountsUpdate = resource.OperationSpec{
	Name:      "update",
	Path:      "/v2/core/accounts/{id}",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Update an Account",
	Params: map[string]*resource.ParamSpec{
		"configuration.merchant.capabilities.naver_pay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.oxxo_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"defaults.responsibilities.fees_collector": {
			Type:        "string",
			Description: "A value indicating the party responsible for collecting fees from this account.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "application"},
				{Value: "application_custom"},
				{Value: "application_express"},
				{Value: "stripe"},
			},
		},
		"identity.individual.address.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"configuration.merchant.capabilities.klarna_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.monthly_estimated_revenue.amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
		},
		"identity.business_details.registration_date.month": {
			Type:        "integer",
			Description: "The month of registration, between 1 and 12.",
			Required:    true,
		},
		"display_name": {
			Type:        "string",
			Description: "A descriptive name for the Account. This name will be surfaced in the Stripe Dashboard and on any invoices sent to the Account.",
		},
		"configuration.merchant.capabilities.multibanco_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.attestations.directorship_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the director attestation was made.",
		},
		"identity.business_details.documents.company_license.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.individual.relationship.title": {
			Type:        "string",
			Description: "The person's title (e.g., CEO, Support Engineer).",
		},
		"identity.individual.script_addresses.kanji.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"contact_phone": {
			Type:        "string",
			Description: "The default contact phone for the Account.",
		},
		"configuration.customer.billing.default_payment_method": {
			Type:        "string",
			Description: "ID of a PaymentMethod attached to the customer account to use as the default for invoices and subscriptions.",
		},
		"configuration.merchant.capabilities.boleto_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.individual.date_of_birth.month": {
			Type:        "integer",
			Description: "The month of birth.",
			Required:    true,
		},
		"identity.individual.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"configuration.merchant.branding.primary_color": {
			Type:        "string",
			Description: "A CSS hex color value representing the primary branding color for the merchant.",
		},
		"configuration.merchant.capabilities.mx_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.script_addresses.kanji.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"configuration.storer.capabilities.holds_currencies.eur.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.storer.capabilities.holds_currencies.gbp.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.smart_disputes.auto_respond.preference": {
			Type:        "string",
			Description: "The preference for automatic dispute responses.",
			Enum: []resource.EnumSpec{
				{Value: "inherit"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"configuration.recipient.applied": {
			Type:        "boolean",
			Description: "Represents the state of the configuration, and can be updated to deactivate or re-apply a configuration.",
		},
		"identity.business_details.script_addresses.kana.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.individual.documents.primary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"configuration.merchant.konbini_payments.support.phone": {
			Type:        "string",
			Description: "Support phone number for Konbini payments.",
		},
		"identity.business_details.phone": {
			Type:        "string",
			Description: "The phone number of the Business Entity.",
		},
		"identity.business_details.documents.company_registration_verification.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.individual.documents.secondary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"configuration.customer.shipping.address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"configuration.customer.shipping.phone": {
			Type:        "string",
			Description: "Customer phone (including extension).",
		},
		"configuration.merchant.capabilities.samsung_pay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.documents.proof_of_registration.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.business_details.documents.company_registration_verification.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.documents.primary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"configuration.storer.applied": {
			Type:        "boolean",
			Description: "Represents the state of the configuration, and can be updated to deactivate or re-apply a configuration.",
		},
		"configuration.merchant.support.address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.business_details.script_names.kana.registered_name": {
			Type:        "string",
			Description: "Registered name of the business.",
		},
		"identity.attestations.representative_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the representative attestation was made.",
		},
		"identity.business_details.documents.company_memorandum_of_association.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.storer.capabilities.outbound_transfers.bank_accounts.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.customer.shipping.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"configuration.merchant.capabilities.card_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.attestations.terms_of_service.account.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the Account's representative accepted the terms of service.",
		},
		"identity.attestations.terms_of_service.crypto_storer.ip": {
			Type:        "string",
			Description: "The IP address from which the Account's representative accepted the terms of service.",
		},
		"identity.business_details.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.individual.script_names.kanji.given_name": {
			Type:        "string",
			Description: "The person's first or given name.",
		},
		"contact_email": {
			Type:        "string",
			Description: "The default contact email address for the Account. Required when configuring the account as a merchant or recipient.",
		},
		"configuration.storer.capabilities.outbound_payments.cards.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.paynow_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.ideal_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.individual.nationalities": {
			Type:        "array",
			Description: "The countries where the individual is a national. Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.individual.documents.primary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"configuration.merchant.capabilities.kr_card_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.script_statement_descriptor.kana.prefix": {
			Type:        "string",
			Description: "Default text that appears on statements for card charges outside of Japan, prefixing any dynamic statement_descriptor_suffix specified on the charge. To maximize space for the dynamic part of the descriptor, keep this text short. If you don’t specify this value, statement_descriptor is used as the prefix. For more information about statement descriptors and their requirements, see the Merchant Configuration settings documentation.",
		},
		"identity.business_details.registered_name": {
			Type:        "string",
			Description: "The business legal name.",
		},
		"identity.business_details.documents.company_ministerial_decree.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.merchant.capabilities.bacs_debit_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.script_names.kanji.registered_name": {
			Type:        "string",
			Description: "Registered name of the business.",
		},
		"identity.business_details.documents.company_tax_id_verification.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.individual.documents.passport.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.attestations.terms_of_service.account.date": {
			Type:        "string",
			Description: "The time when the Account's representative accepted the terms of service. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Format:      "date-time",
		},
		"identity.business_details.script_addresses.kana.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.business_details.documents.proof_of_ultimate_beneficial_ownership.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.individual.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.individual.address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"configuration.merchant.capabilities.mobilepay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.p24_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.individual.relationship.owner": {
			Type:        "boolean",
			Description: "Whether the person is an owner of the account’s identity.",
		},
		"identity.individual.script_addresses.kanji.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.attestations.terms_of_service.account.ip": {
			Type:        "string",
			Description: "The IP address from which the Account's representative accepted the terms of service.",
		},
		"identity.business_details.estimated_worker_count": {
			Type:        "integer",
			Description: "Estimated maximum number of workers currently engaged by the business (including employees, contractors, and vendors).",
		},
		"identity.business_details.documents.primary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"identity.individual.script_addresses.kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.individual.script_names.kanji.surname": {
			Type:        "string",
			Description: "The person's last or family name.",
		},
		"identity.business_details.script_addresses.kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.business_details.script_addresses.kanji.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.business_details.address.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"configuration.customer.billing.invoice.rendering.amount_tax_display": {
			Type:        "string",
			Description: "Indicates whether displayed line item prices and amounts on invoice PDFs include inclusive tax amounts. Must be either `include_inclusive_tax` or `exclude_tax`.",
			Enum: []resource.EnumSpec{
				{Value: "exclude_tax"},
				{Value: "include_inclusive_tax"},
			},
		},
		"identity.attestations.ownership_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the beneficial owner attestation was made.",
		},
		"identity.individual.script_addresses.kana.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"configuration.merchant.branding.logo": {
			Type:        "string",
			Description: "ID of a [file upload](https://docs.stripe.com/api/persons/update#create_file): A logo for the merchant that will be used in Checkout instead of the icon and without the merchant's name next to it if provided. Must be at least 128px x 128px.",
		},
		"configuration.merchant.capabilities.konbini_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.card_payments.decline_on.avs_failure": {
			Type:        "boolean",
			Description: "Whether Stripe automatically declines charges with an incorrect ZIP or postal code. This setting only applies when a ZIP or postal code is provided and they fail bank verification.",
		},
		"defaults.profile.business_url": {
			Type:        "string",
			Description: "The business's publicly-available website.",
		},
		"configuration.customer.automatic_indirect_tax.location_source": {
			Type:        "string",
			Description: "Data source used to identify the customer account's tax location. Defaults to `identity_address`. Used for automatic indirect tax calculation.",
			Enum: []resource.EnumSpec{
				{Value: "identity_address"},
				{Value: "ip_address"},
				{Value: "payment_method"},
				{Value: "shipping_address"},
			},
		},
		"configuration.merchant.support.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.attestations.directorship_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the director attestation was made.",
		},
		"identity.business_details.script_addresses.kanji.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.individual.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.individual.documents.company_authorization.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"configuration.merchant.support.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.individual.documents.secondary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"configuration.merchant.capabilities.cartes_bancaires_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.attestations.terms_of_service.storer.ip": {
			Type:        "string",
			Description: "The IP address from which the Account's representative accepted the terms of service.",
		},
		"account_token": {
			Type:        "string",
			Description: "The account token generated by the account token api.",
		},
		"configuration.customer.billing.invoice.prefix": {
			Type:        "string",
			Description: "Prefix used to generate unique invoice numbers. Must be 3-12 uppercase letters or numbers.",
		},
		"configuration.merchant.capabilities.eps_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.attestations.terms_of_service.storer.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the Account's representative accepted the terms of service.",
		},
		"configuration.storer.capabilities.inbound_transfers.bank_accounts.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.entity_type": {
			Type:        "string",
			Description: "The entity type.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "government_entity"},
				{Value: "individual"},
				{Value: "non_profit"},
			},
		},
		"identity.individual.documents.primary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"identity.attestations.terms_of_service.crypto_storer.date": {
			Type:        "string",
			Description: "The time when the Account's representative accepted the terms of service. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Format:      "date-time",
		},
		"identity.business_details.annual_revenue.amount.value": {
			Type:        "integer",
			Description: "A non-negative integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#minor-units).",
			Required:    true,
		},
		"configuration.customer.billing.invoice.rendering.template": {
			Type:        "string",
			Description: "ID of the invoice rendering template to use for future invoices.",
		},
		"configuration.customer.shipping.address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"configuration.merchant.capabilities.us_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.recipient.default_outbound_destination": {
			Type:        "string",
			Description: "The payout method id to be used as a default outbound destination. This will allow the PayoutMethod to be omitted on OutboundPayments made through API or sending payouts via dashboard. Can also be explicitly set to `null` to clear the existing default outbound destination. For further details about creating an Outbound Destination, see [Collect recipient's payment details](https://docs.stripe.com/global-payouts-private-preview/quickstart?dashboard-or-api=api#collect-bank-account-details).",
		},
		"defaults.profile.doing_business_as": {
			Type:        "string",
			Description: "The name which is used by the business.",
		},
		"identity.business_details.address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.individual.surname": {
			Type:        "string",
			Description: "The individual's last name.",
		},
		"configuration.customer.shipping.name": {
			Type:        "string",
			Description: "Customer name.",
		},
		"configuration.merchant.capabilities.sepa_debit_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.individual.script_addresses.kana.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"configuration.merchant.mcc": {
			Type:        "string",
			Description: "The Merchant Category Code (MCC) for the merchant. MCCs classify businesses based on the goods or services they provide.",
		},
		"configuration.merchant.capabilities.ach_debit_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.individual.given_name": {
			Type:        "string",
			Description: "The individual's first name.",
		},
		"dashboard": {
			Type:        "string",
			Description: "A value indicating the Stripe dashboard this Account has access to. This will depend on which configurations are enabled for this account.",
			Enum: []resource.EnumSpec{
				{Value: "express"},
				{Value: "full"},
				{Value: "none"},
			},
		},
		"configuration.storer.capabilities.outbound_payments.bank_accounts.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.alma_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"defaults.responsibilities.losses_collector": {
			Type:        "string",
			Description: "A value indicating who is responsible for losses when this Account can’t pay back negative balances from payments.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "application"},
				{Value: "stripe"},
			},
		},
		"identity.individual.political_exposure": {
			Type:        "string",
			Description: "The individual's political exposure.",
			Enum: []resource.EnumSpec{
				{Value: "existing"},
				{Value: "none"},
			},
		},
		"configuration.merchant.capabilities.link_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.twint_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.individual.script_addresses.kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"configuration.customer.shipping.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"configuration.merchant.support.address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"configuration.merchant.branding.secondary_color": {
			Type:        "string",
			Description: "A CSS hex color value representing the secondary branding color for the merchant.",
		},
		"configuration.merchant.capabilities.bancontact_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.pay_by_bank_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.swish_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.recipient.capabilities.bank_accounts.wire.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.script_addresses.kana.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.attestations.representative_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the representative attestation was made.",
		},
		"identity.attestations.directorship_declaration.date": {
			Type:        "string",
			Description: "The time marking when the director attestation was made. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Format:      "date-time",
		},
		"identity.business_details.script_addresses.kanji.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.business_details.structure": {
			Type:        "string",
			Description: "The category identifying the legal structure of the business.",
			Enum: []resource.EnumSpec{
				{Value: "cooperative"},
				{Value: "free_zone_establishment"},
				{Value: "free_zone_llc"},
				{Value: "governmental_unit"},
				{Value: "government_instrumentality"},
				{Value: "incorporated_association"},
				{Value: "incorporated_non_profit"},
				{Value: "incorporated_partnership"},
				{Value: "limited_liability_partnership"},
				{Value: "llc"},
				{Value: "multi_member_llc"},
				{Value: "private_company"},
				{Value: "private_corporation"},
				{Value: "private_partnership"},
				{Value: "public_company"},
				{Value: "public_corporation"},
				{Value: "public_listed_corporation"},
				{Value: "public_partnership"},
				{Value: "registered_charity"},
				{Value: "single_member_llc"},
				{Value: "sole_establishment"},
				{Value: "sole_proprietorship"},
				{Value: "tax_exempt_government_instrumentality"},
				{Value: "trust"},
				{Value: "unincorporated_association"},
				{Value: "unincorporated_non_profit"},
				{Value: "unincorporated_partnership"},
			},
		},
		"identity.business_details.documents.company_tax_id_verification.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.individual.script_addresses.kana.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.individual.phone": {
			Type:        "string",
			Description: "The individual's phone number.",
		},
		"configuration.customer.automatic_indirect_tax.exempt": {
			Type:        "string",
			Description: "The customer account's tax exemption status: `none`, `exempt`, or `reverse`. When `reverse`, invoice and receipt PDFs include \"Reverse charge\".",
			Enum: []resource.EnumSpec{
				{Value: "exempt"},
				{Value: "none"},
				{Value: "reverse"},
			},
		},
		"configuration.merchant.capabilities.affirm_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.kakao_pay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.script_statement_descriptor.kanji.descriptor": {
			Type:        "string",
			Description: "The default text that appears on statements for non-card charges outside of Japan. For card charges, if you don’t set a statement_descriptor_prefix, this text is also used as the statement descriptor prefix. In that case, if concatenating the statement descriptor suffix causes the combined statement descriptor to exceed 22 characters, we truncate the statement_descriptor text to limit the full descriptor to 22 characters. For more information about statement descriptors and their requirements, see the Merchant Configuration settings documentation.",
		},
		"identity.individual.relationship.director": {
			Type:        "boolean",
			Description: "Whether the person is a director of the account's identity. Directors are typically members of the governing board of the company, or responsible for ensuring the company meets its regulatory obligations.",
		},
		"configuration.storer.capabilities.holds_currencies.usd.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.customer.capabilities.automatic_indirect_tax.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.customer.test_clock": {
			Type:        "string",
			Description: "ID of the test clock to attach to the customer. Can only be set on testmode Accounts, and when the Customer Configuration is first set on an Account.",
		},
		"configuration.merchant.capabilities.zip_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.script_statement_descriptor.kana.descriptor": {
			Type:        "string",
			Description: "The default text that appears on statements for non-card charges outside of Japan. For card charges, if you don’t set a statement_descriptor_prefix, this text is also used as the statement descriptor prefix. In that case, if concatenating the statement descriptor suffix causes the combined statement descriptor to exceed 22 characters, we truncate the statement_descriptor text to limit the full descriptor to 22 characters. For more information about statement descriptors and their requirements, see the Merchant Configuration settings documentation.",
		},
		"defaults.locales": {
			Type:        "array",
			Description: "The Account's preferred locales (languages), ordered by preference.",
		},
		"identity.attestations.persons_provided.directors": {
			Type:        "boolean",
			Description: "Whether the company’s directors have been provided. Set this Boolean to true after creating all the company’s directors with the [Persons API](https://docs.stripe.com/api/v2/core/accounts/createperson).",
		},
		"identity.attestations.terms_of_service.storer.date": {
			Type:        "string",
			Description: "The time when the Account's representative accepted the terms of service. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Format:      "date-time",
		},
		"configuration.merchant.support.address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"configuration.merchant.support.email": {
			Type:        "string",
			Description: "A publicly available email address for sending support issues to.",
		},
		"identity.individual.script_addresses.kanji.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.individual.script_addresses.kanji.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"configuration.merchant.capabilities.revolut_pay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.applied": {
			Type:        "boolean",
			Description: "Represents the state of the configuration, and can be updated to deactivate or re-apply a configuration.",
		},
		"identity.business_details.address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.individual.documents.visa.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.individual.documents.company_authorization.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.individual.legal_gender": {
			Type:        "string",
			Description: "The individual's gender (International regulations require either \"male\" or \"female\").",
			Enum: []resource.EnumSpec{
				{Value: "female"},
				{Value: "male"},
			},
		},
		"configuration.customer.billing.invoice.next_sequence": {
			Type:        "integer",
			Description: "Sequence number to use on the customer account's next invoice. Defaults to 1.",
		},
		"configuration.merchant.statement_descriptor.descriptor": {
			Type:        "string",
			Description: "The default text that appears on statements for non-card charges outside of Japan. For card charges, if you don’t set a statement_descriptor_prefix, this text is also used as the statement descriptor prefix. In that case, if concatenating the statement descriptor suffix causes the combined statement descriptor to exceed 22 characters, we truncate the statement_descriptor text to limit the full descriptor to 22 characters. For more information about statement descriptors and their requirements, see the Merchant Configuration settings documentation.",
		},
		"configuration.merchant.bacs_debit_payments.display_name": {
			Type:        "string",
			Description: "Display name for Bacs Direct Debit payments.",
		},
		"identity.individual.email": {
			Type:        "string",
			Description: "The individual's email address.",
		},
		"configuration.storer.capabilities.outbound_transfers.financial_accounts.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.storer.capabilities.financial_addresses.bank_accounts.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.customer.shipping.address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"configuration.merchant.capabilities.au_becs_debit_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.monthly_estimated_revenue.amount.value": {
			Type:        "integer",
			Description: "A non-negative integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#minor-units).",
			Required:    true,
		},
		"identity.business_details.registration_date.year": {
			Type:        "integer",
			Description: "The four-digit year of registration.",
			Required:    true,
		},
		"identity.individual.relationship.executive": {
			Type:        "boolean",
			Description: "Whether the person has significant responsibility to control, manage, or direct the organization.",
		},
		"configuration.customer.applied": {
			Type:        "boolean",
			Description: "Represents the state of the configuration, and can be updated to deactivate or re-apply a configuration.",
		},
		"configuration.merchant.statement_descriptor.prefix": {
			Type:        "string",
			Description: "Default text that appears on statements for card charges outside of Japan, prefixing any dynamic statement_descriptor_suffix specified on the charge. To maximize space for the dynamic part of the descriptor, keep this text short. If you don’t specify this value, statement_descriptor is used as the prefix. For more information about statement descriptors and their requirements, see the Merchant Configuration settings documentation.",
		},
		"configuration.merchant.capabilities.amazon_pay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.gb_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.konbini_payments.support.hours.start_time": {
			Type:        "string",
			Description: "Support hours start time (JST time of day) for in `HH:MM` format.",
		},
		"identity.individual.script_addresses.kana.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.individual.script_names.kana.given_name": {
			Type:        "string",
			Description: "The person's first or given name.",
		},
		"configuration.customer.billing.invoice.footer": {
			Type:        "string",
			Description: "Default invoice footer.",
		},
		"identity.individual.script_addresses.kana.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.individual.script_addresses.kanji.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.attestations.ownership_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the beneficial owner attestation was made.",
		},
		"identity.business_details.documents.proof_of_address.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.individual.documents.secondary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"configuration.storer.capabilities.outbound_payments.financial_accounts.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.support.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"configuration.merchant.konbini_payments.support.email": {
			Type:        "string",
			Description: "Support email address for Konbini payments.",
		},
		"configuration.recipient.capabilities.cards.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.attestations.ownership_declaration.date": {
			Type:        "string",
			Description: "The time marking when the beneficial owner attestation was made. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Format:      "date-time",
		},
		"identity.business_details.script_addresses.kanji.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"configuration.customer.automatic_indirect_tax.validate_location": {
			Type:        "string",
			Description: "A per-request flag that indicates when Stripe should validate the customer tax location - defaults to `auto`.",
			Enum: []resource.EnumSpec{
				{Value: "auto"},
				{Value: "deferred"},
				{Value: "immediately"},
			},
		},
		"configuration.merchant.branding.icon": {
			Type:        "string",
			Description: "ID of a [file upload](https://docs.stripe.com/api/persons/update#create_file): An icon for the merchant. Must be square and at least 128px x 128px.",
		},
		"defaults.profile.product_description": {
			Type:        "string",
			Description: "Internal-only description of the product sold or service provided by the business. It's used by Stripe for risk and underwriting purposes.",
		},
		"identity.business_details.script_addresses.kana.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"configuration.merchant.card_payments.decline_on.cvc_failure": {
			Type:        "boolean",
			Description: "Whether Stripe automatically declines charges with an incorrect CVC. This setting only applies when a CVC is provided and it fails bank verification.",
		},
		"identity.attestations.persons_provided.owners": {
			Type:        "boolean",
			Description: "Whether the company’s owners have been provided. Set this Boolean to true after creating all the company’s owners with the [Persons API](https://docs.stripe.com/api/v2/core/accounts/createperson).",
		},
		"identity.business_details.documents.primary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"defaults.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
		},
		"identity.business_details.script_addresses.kana.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.business_details.documents.bank_account_ownership_verification.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.individual.date_of_birth.day": {
			Type:        "integer",
			Description: "The day of the birth.",
			Required:    true,
		},
		"identity.individual.script_addresses.kana.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"configuration.merchant.capabilities.grabpay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.documents.proof_of_address.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.merchant.capabilities.fpx_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.annual_revenue.fiscal_year_end": {
			Type:        "string",
			Description: "The close-out date of the preceding fiscal year in ISO 8601 format. E.g. 2023-12-31 for the 31st of December, 2023.",
		},
		"identity.country": {
			Type:        "string",
			Description: "The country in which the account holder resides, or in which the business is legally established. This should be an [ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2) country code.",
		},
		"identity.individual.script_names.kana.surname": {
			Type:        "string",
			Description: "The person's last or family name.",
		},
		"identity.attestations.terms_of_service.crypto_storer.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the Account's representative accepted the terms of service.",
		},
		"identity.business_details.documents.company_ministerial_decree.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"configuration.merchant.capabilities.jp_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.script_statement_descriptor.kanji.prefix": {
			Type:        "string",
			Description: "Default text that appears on statements for card charges outside of Japan, prefixing any dynamic statement_descriptor_suffix specified on the charge. To maximize space for the dynamic part of the descriptor, keep this text short. If you don’t specify this value, statement_descriptor is used as the prefix. For more information about statement descriptors and their requirements, see the Merchant Configuration settings documentation.",
		},
		"identity.business_details.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.individual.date_of_birth.year": {
			Type:        "integer",
			Description: "The year of birth.",
			Required:    true,
		},
		"identity.individual.script_addresses.kanji.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"configuration.customer.automatic_indirect_tax.ip_address": {
			Type:        "string",
			Description: "A recent IP address of the customer used for tax reporting and tax location inference.",
		},
		"configuration.recipient.capabilities.stripe_balance.stripe_transfers.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.individual.address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.individual.documents.passport.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.merchant.support.address.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"configuration.merchant.konbini_payments.support.hours.end_time": {
			Type:        "string",
			Description: "Support hours end time (JST time of day) for in `HH:MM` format.",
		},
		"identity.attestations.persons_provided.ownership_exemption_reason": {
			Type:        "string",
			Description: "Reason for why the company is exempt from providing ownership information.",
			Enum: []resource.EnumSpec{
				{Value: "qualified_entity_exceeds_ownership_threshold"},
				{Value: "qualifies_as_financial_institution"},
			},
		},
		"identity.attestations.representative_declaration.date": {
			Type:        "string",
			Description: "The time marking when the representative attestation was made. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Format:      "date-time",
		},
		"identity.business_details.registration_date.day": {
			Type:        "integer",
			Description: "The day of registration, between 1 and 31.",
			Required:    true,
		},
		"identity.business_details.documents.bank_account_ownership_verification.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.merchant.capabilities.cashapp_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.documents.proof_of_registration.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.documents.company_memorandum_of_association.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"configuration.merchant.capabilities.acss_debit_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.promptpay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.script_addresses.kana.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.business_details.script_addresses.kanji.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.business_details.documents.company_license.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.customer.shipping.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"configuration.merchant.support.phone": {
			Type:        "string",
			Description: "A publicly available phone number to call with support issues.",
		},
		"identity.individual.documents.visa.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"configuration.merchant.support.url": {
			Type:        "string",
			Description: "A publicly available website for handling support issues.",
		},
		"configuration.merchant.capabilities.blik_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.afterpay_clearpay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"include": {
			Type:        "array",
			Description: "Additional fields to include in the response.",
		},
		"configuration.merchant.capabilities.sepa_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.merchant.capabilities.payco_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"configuration.recipient.capabilities.bank_accounts.local.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.business_details.annual_revenue.amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
		},
		"identity.individual.relationship.percent_ownership": {
			Type:        "string",
			Description: "The percent owned by the person of the account's legal entity.",
			Format:      "decimal",
		},
		"configuration.merchant.capabilities.jcb_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
		},
		"identity.attestations.persons_provided.executives": {
			Type:        "boolean",
			Description: "Whether the company’s executives have been provided. Set this Boolean to true after creating all the company’s executives with the [Persons API](https://docs.stripe.com/api/v2/core/accounts/createperson).",
		},
		"identity.business_details.script_addresses.kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.business_details.documents.proof_of_ultimate_beneficial_ownership.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.individual.address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
	},
}

var V2PreviewCoreAccountsClose = resource.OperationSpec{
	Name:      "close",
	Path:      "/v2/core/accounts/{id}/close",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Close an Account",
	Params: map[string]*resource.ParamSpec{
		"applied_configurations": {
			Type:        "array",
			Description: "Configurations on the Account to be closed. All configurations on the Account must be passed in for this request to succeed.",
		},
	},
}

var V2PreviewCoreAccountsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/core/accounts",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Accounts",
	Params: map[string]*resource.ParamSpec{
		"applied_configurations": {
			Type:        "array",
			Description: "Filter only accounts that have all of the configurations specified. If omitted, returns all accounts regardless of which configurations they have.",
		},
		"closed": {
			Type:        "boolean",
			Description: "Filter by whether the account is closed. If omitted, returns only Accounts that are not closed.",
		},
		"limit": {
			Type:        "integer",
			Description: "The upper limit on the number of accounts returned by the List Account request.",
		},
		"page": {
			Type:        "string",
			Description: "The page token to navigate to next or previous batch of accounts given by the list request.",
		},
	},
}

var V2PreviewCoreAccountsCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/core/accounts",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create an Account",
	Params: map[string]*resource.ParamSpec{
		"configuration.merchant.capabilities.payco_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.alma_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.konbini_payments.support.hours.end_time": {
			Type:        "string",
			Description: "Support hours end time (JST time of day) for in `HH:MM` format.",
		},
		"defaults.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
		},
		"identity.individual.script_addresses.kana.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"configuration.merchant.capabilities.samsung_pay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.ideal_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"defaults.profile.product_description": {
			Type:        "string",
			Description: "Internal-only description of the product sold or service provided by the business. It's used by Stripe for risk and underwriting purposes.",
		},
		"identity.individual.given_name": {
			Type:        "string",
			Description: "The individual's first name.",
		},
		"identity.attestations.persons_provided.owners": {
			Type:        "boolean",
			Description: "Whether the company’s owners have been provided. Set this Boolean to true after creating all the company’s owners with the [Persons API](https://docs.stripe.com/api/v2/core/accounts/createperson).",
		},
		"identity.business_details.script_addresses.kanji.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.individual.script_addresses.kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
			Required:    true,
		},
		"identity.business_details.documents.company_license.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.estimated_worker_count": {
			Type:        "integer",
			Description: "Estimated maximum number of workers currently engaged by the business (including employees, contractors, and vendors).",
		},
		"configuration.merchant.capabilities.swish_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.relationship.executive": {
			Type:        "boolean",
			Description: "Whether the person has significant responsibility to control, manage, or direct the organization.",
		},
		"identity.individual.address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.attestations.persons_provided.ownership_exemption_reason": {
			Type:        "string",
			Description: "Reason for why the company is exempt from providing ownership information.",
			Enum: []resource.EnumSpec{
				{Value: "qualified_entity_exceeds_ownership_threshold"},
				{Value: "qualifies_as_financial_institution"},
			},
		},
		"identity.business_details.registration_date.month": {
			Type:        "integer",
			Description: "The month of registration, between 1 and 12.",
			Required:    true,
		},
		"configuration.storer.capabilities.inbound_transfers.bank_accounts.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.date_of_birth.year": {
			Type:        "integer",
			Description: "The year of birth.",
			Required:    true,
		},
		"identity.individual.legal_gender": {
			Type:        "string",
			Description: "The individual's gender (International regulations require either \"male\" or \"female\").",
			Enum: []resource.EnumSpec{
				{Value: "female"},
				{Value: "male"},
			},
		},
		"configuration.customer.billing.invoice.prefix": {
			Type:        "string",
			Description: "Prefix used to generate unique invoice numbers. Must be 3-12 uppercase letters or numbers.",
		},
		"display_name": {
			Type:        "string",
			Description: "A descriptive name for the Account. This name will be surfaced in the Stripe Dashboard and on any invoices sent to the Account.",
		},
		"identity.individual.nationalities": {
			Type:        "array",
			Description: "The countries where the individual is a national. Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"configuration.merchant.capabilities.klarna_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.relationship.percent_ownership": {
			Type:        "string",
			Description: "The percent owned by the person of the account's legal entity.",
			Format:      "decimal",
		},
		"identity.business_details.documents.primary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"configuration.customer.capabilities.automatic_indirect_tax.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.documents.primary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"identity.attestations.persons_provided.directors": {
			Type:        "boolean",
			Description: "Whether the company’s directors have been provided. Set this Boolean to true after creating all the company’s directors with the [Persons API](https://docs.stripe.com/api/v2/core/accounts/createperson).",
		},
		"configuration.merchant.capabilities.pay_by_bank_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.script_addresses.kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
			Required:    true,
		},
		"configuration.merchant.capabilities.kakao_pay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.cashapp_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.cartes_bancaires_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.branding.icon": {
			Type:        "string",
			Description: "ID of a [file upload](https://docs.stripe.com/api/persons/update#create_file): An icon for the merchant. Must be square and at least 128px x 128px.",
		},
		"identity.individual.relationship.title": {
			Type:        "string",
			Description: "The person's title (e.g., CEO, Support Engineer).",
		},
		"identity.business_details.script_addresses.kana.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"account_token": {
			Type:        "string",
			Description: "The account token generated by the account token api.",
		},
		"configuration.merchant.capabilities.fpx_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.affirm_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.script_addresses.kanji.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.individual.script_addresses.kanji.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"configuration.merchant.capabilities.zip_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.amazon_pay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.recipient.capabilities.bank_accounts.wire.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"dashboard": {
			Type:        "string",
			Description: "A value indicating the Stripe dashboard this Account has access to. This will depend on which configurations are enabled for this account.",
			Enum: []resource.EnumSpec{
				{Value: "express"},
				{Value: "full"},
				{Value: "none"},
			},
		},
		"identity.individual.script_addresses.kana.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.business_details.script_addresses.kanji.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.business_details.documents.proof_of_address.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.phone": {
			Type:        "string",
			Description: "The phone number of the Business Entity.",
		},
		"identity.individual.date_of_birth.day": {
			Type:        "integer",
			Description: "The day of birth.",
			Required:    true,
		},
		"identity.individual.script_addresses.kana.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.individual.documents.passport.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.merchant.capabilities.eps_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.branding.logo": {
			Type:        "string",
			Description: "ID of a [file upload](https://docs.stripe.com/api/persons/update#create_file): A logo for the merchant that will be used in Checkout instead of the icon and without the merchant's name next to it if provided. Must be at least 128px x 128px.",
		},
		"identity.business_details.documents.proof_of_address.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.customer.shipping.address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"configuration.merchant.capabilities.revolut_pay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.support.address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"configuration.merchant.capabilities.card_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"defaults.responsibilities.fees_collector": {
			Type:        "string",
			Description: "A value indicating the party responsible for collecting fees from this account.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "application"},
				{Value: "application_custom"},
				{Value: "application_express"},
				{Value: "stripe"},
			},
		},
		"identity.individual.phone": {
			Type:        "string",
			Description: "The individual's phone number.",
		},
		"identity.individual.documents.primary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
			Required:    true,
		},
		"identity.attestations.representative_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the representative attestation was made.",
		},
		"identity.business_details.script_addresses.kana.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.business_details.annual_revenue.amount.value": {
			Type:        "integer",
			Description: "A non-negative integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#minor-units).",
			Required:    true,
		},
		"configuration.merchant.branding.primary_color": {
			Type:        "string",
			Description: "A CSS hex color value representing the primary branding color for the merchant.",
		},
		"identity.individual.script_names.kana.surname": {
			Type:        "string",
			Description: "The person's last or family name.",
		},
		"identity.individual.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
			Required:    true,
		},
		"configuration.merchant.capabilities.paynow_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.attestations.terms_of_service.storer.date": {
			Type:        "string",
			Description: "The time when the Account's representative accepted the terms of service. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Required:    true,
			Format:      "date-time",
		},
		"configuration.merchant.capabilities.bacs_debit_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.mobilepay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.documents.visa.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.storer.capabilities.holds_currencies.gbp.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.script_addresses.kana.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"configuration.customer.shipping.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"configuration.merchant.support.phone": {
			Type:        "string",
			Description: "A publicly available phone number to call with support issues.",
		},
		"configuration.merchant.capabilities.link_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.script_addresses.kana.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.individual.documents.secondary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"identity.attestations.terms_of_service.account.date": {
			Type:        "string",
			Description: "The time when the Account's representative accepted the terms of service. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Required:    true,
			Format:      "date-time",
		},
		"identity.business_details.script_addresses.kana.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"configuration.customer.automatic_indirect_tax.location_source": {
			Type:        "string",
			Description: "The data source used to identify the customer's tax location - defaults to `identity_address`. Will only be used for automatic tax calculation on the customer's Invoices and Subscriptions. This behavior is now deprecated for new users.",
			Enum: []resource.EnumSpec{
				{Value: "identity_address"},
				{Value: "ip_address"},
				{Value: "payment_method"},
				{Value: "shipping_address"},
			},
		},
		"configuration.merchant.bacs_debit_payments.display_name": {
			Type:        "string",
			Description: "Display name for Bacs Direct Debit payments.",
		},
		"configuration.merchant.statement_descriptor.descriptor": {
			Type:        "string",
			Description: "The default text that appears on statements for non-card charges outside of Japan. For card charges, if you don’t set a statement_descriptor_prefix, this text is also used as the statement descriptor prefix. In that case, if concatenating the statement descriptor suffix causes the combined statement descriptor to exceed 22 characters, we truncate the statement_descriptor text to limit the full descriptor to 22 characters. For more information about statement descriptors and their requirements, see the Merchant Configuration settings documentation.",
		},
		"identity.business_details.script_addresses.kana.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.business_details.structure": {
			Type:        "string",
			Description: "The category identifying the legal structure of the business.",
			Enum: []resource.EnumSpec{
				{Value: "cooperative"},
				{Value: "free_zone_establishment"},
				{Value: "free_zone_llc"},
				{Value: "governmental_unit"},
				{Value: "government_instrumentality"},
				{Value: "incorporated_association"},
				{Value: "incorporated_non_profit"},
				{Value: "incorporated_partnership"},
				{Value: "limited_liability_partnership"},
				{Value: "llc"},
				{Value: "multi_member_llc"},
				{Value: "private_company"},
				{Value: "private_corporation"},
				{Value: "private_partnership"},
				{Value: "public_company"},
				{Value: "public_corporation"},
				{Value: "public_listed_corporation"},
				{Value: "public_partnership"},
				{Value: "registered_charity"},
				{Value: "single_member_llc"},
				{Value: "sole_establishment"},
				{Value: "sole_proprietorship"},
				{Value: "tax_exempt_government_instrumentality"},
				{Value: "trust"},
				{Value: "unincorporated_association"},
				{Value: "unincorporated_non_profit"},
				{Value: "unincorporated_partnership"},
			},
		},
		"configuration.merchant.capabilities.kr_card_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"defaults.responsibilities.losses_collector": {
			Type:        "string",
			Description: "A value indicating who is responsible for losses when this Account can’t pay back negative balances from payments.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "application"},
				{Value: "stripe"},
			},
		},
		"identity.attestations.representative_declaration.date": {
			Type:        "string",
			Description: "The time marking when the representative attestation was made. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Format:      "date-time",
		},
		"identity.business_details.documents.proof_of_registration.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"contact_email": {
			Type:        "string",
			Description: "The default contact email address for the Account. Required when configuring the account as a merchant or recipient.",
		},
		"configuration.merchant.capabilities.oxxo_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.card_payments.decline_on.cvc_failure": {
			Type:        "boolean",
			Description: "Whether Stripe automatically declines charges with an incorrect CVC. This setting only applies when a CVC is provided and it fails bank verification.",
		},
		"identity.individual.script_addresses.kanji.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.business_details.script_addresses.kanji.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.business_details.documents.proof_of_registration.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.business_details.documents.company_registration_verification.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.merchant.capabilities.twint_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.konbini_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.sepa_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.documents.secondary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
			Required:    true,
		},
		"configuration.merchant.script_statement_descriptor.kanji.prefix": {
			Type:        "string",
			Description: "Default text that appears on statements for card charges outside of Japan, prefixing any dynamic statement_descriptor_suffix specified on the charge. To maximize space for the dynamic part of the descriptor, keep this text short. If you don’t specify this value, statement_descriptor is used as the prefix. For more information about statement descriptors and their requirements, see the Merchant Configuration settings documentation.",
		},
		"configuration.merchant.capabilities.sepa_debit_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.recipient.capabilities.bank_accounts.local.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.script_addresses.kanji.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.individual.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.attestations.terms_of_service.storer.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the Account's representative accepted the terms of service.",
		},
		"identity.business_details.script_addresses.kana.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.business_details.documents.company_license.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.storer.capabilities.outbound_payments.financial_accounts.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.business_details.script_addresses.kanji.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.business_details.documents.company_registration_verification.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.entity_type": {
			Type:        "string",
			Description: "The entity type.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "government_entity"},
				{Value: "individual"},
				{Value: "non_profit"},
			},
		},
		"configuration.merchant.script_statement_descriptor.kana.prefix": {
			Type:        "string",
			Description: "Default text that appears on statements for card charges outside of Japan, prefixing any dynamic statement_descriptor_suffix specified on the charge. To maximize space for the dynamic part of the descriptor, keep this text short. If you don’t specify this value, statement_descriptor is used as the prefix. For more information about statement descriptors and their requirements, see the Merchant Configuration settings documentation.",
		},
		"configuration.merchant.support.url": {
			Type:        "string",
			Description: "A publicly available website for handling support issues.",
		},
		"configuration.merchant.capabilities.gb_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.statement_descriptor.prefix": {
			Type:        "string",
			Description: "Default text that appears on statements for card charges outside of Japan, prefixing any dynamic statement_descriptor_suffix specified on the charge. To maximize space for the dynamic part of the descriptor, keep this text short. If you don’t specify this value, statement_descriptor is used as the prefix. For more information about statement descriptors and their requirements, see the Merchant Configuration settings documentation.",
		},
		"identity.individual.script_names.kana.given_name": {
			Type:        "string",
			Description: "The person's first or given name.",
		},
		"identity.business_details.script_names.kana.registered_name": {
			Type:        "string",
			Description: "Registered name of the business.",
		},
		"identity.business_details.script_names.kanji.registered_name": {
			Type:        "string",
			Description: "Registered name of the business.",
		},
		"configuration.customer.shipping.name": {
			Type:        "string",
			Description: "Customer name.",
		},
		"identity.individual.relationship.director": {
			Type:        "boolean",
			Description: "Whether the person is a director of the account's identity. Directors are typically members of the governing board of the company, or responsible for ensuring the company meets its regulatory obligations.",
		},
		"identity.attestations.directorship_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the director attestation was made.",
		},
		"identity.business_details.registration_date.day": {
			Type:        "integer",
			Description: "The day of registration, between 1 and 31.",
			Required:    true,
		},
		"identity.business_details.registration_date.year": {
			Type:        "integer",
			Description: "The four-digit year of registration.",
			Required:    true,
		},
		"configuration.merchant.konbini_payments.support.email": {
			Type:        "string",
			Description: "Support email address for Konbini payments.",
		},
		"identity.individual.script_names.kanji.given_name": {
			Type:        "string",
			Description: "The person's first or given name.",
		},
		"identity.business_details.documents.primary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"configuration.customer.shipping.phone": {
			Type:        "string",
			Description: "Customer phone (including extension).",
		},
		"configuration.merchant.mcc": {
			Type:        "string",
			Description: "The Merchant Category Code (MCC) for the Merchant Configuration. MCCs classify businesses based on the goods or services they provide.",
		},
		"defaults.profile.business_url": {
			Type:        "string",
			Description: "The business's publicly-available website.",
		},
		"configuration.recipient.capabilities.stripe_balance.stripe_transfers.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.date_of_birth.month": {
			Type:        "integer",
			Description: "The month of birth.",
			Required:    true,
		},
		"configuration.merchant.capabilities.naver_pay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.jcb_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.branding.secondary_color": {
			Type:        "string",
			Description: "A CSS hex color value representing the secondary branding color for the merchant.",
		},
		"include": {
			Type:        "array",
			Description: "Additional fields to include in the response.",
		},
		"identity.business_details.script_addresses.kanji.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"configuration.merchant.capabilities.p24_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.surname": {
			Type:        "string",
			Description: "The individual's last name.",
		},
		"identity.attestations.ownership_declaration.date": {
			Type:        "string",
			Description: "The time marking when the beneficial owner attestation was made. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Format:      "date-time",
		},
		"configuration.merchant.capabilities.jp_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.script_names.kanji.surname": {
			Type:        "string",
			Description: "The person's last or family name.",
		},
		"identity.business_details.script_addresses.kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
			Required:    true,
		},
		"configuration.merchant.script_statement_descriptor.kana.descriptor": {
			Type:        "string",
			Description: "The default text that appears on statements for non-card charges outside of Japan. For card charges, if you don’t set a statement_descriptor_prefix, this text is also used as the statement descriptor prefix. In that case, if concatenating the statement descriptor suffix causes the combined statement descriptor to exceed 22 characters, we truncate the statement_descriptor text to limit the full descriptor to 22 characters. For more information about statement descriptors and their requirements, see the Merchant Configuration settings documentation.",
		},
		"configuration.merchant.capabilities.promptpay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.multibanco_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.business_details.address.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"configuration.merchant.smart_disputes.auto_respond.preference": {
			Type:        "string",
			Description: "The preference for Smart Disputes auto-respond.",
			Enum: []resource.EnumSpec{
				{Value: "inherit"},
				{Value: "off"},
				{Value: "on"},
			},
		},
		"configuration.merchant.support.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"configuration.merchant.card_payments.decline_on.avs_failure": {
			Type:        "boolean",
			Description: "Whether Stripe automatically declines charges with an incorrect ZIP or postal code. This setting only applies when a ZIP or postal code is provided and they fail bank verification.",
		},
		"identity.business_details.documents.proof_of_ultimate_beneficial_ownership.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.merchant.support.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
			Required:    true,
		},
		"configuration.merchant.support.address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.business_details.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
			Required:    true,
		},
		"identity.business_details.documents.company_memorandum_of_association.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.business_details.documents.primary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
			Required:    true,
		},
		"configuration.merchant.support.address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"configuration.merchant.capabilities.afterpay_clearpay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"defaults.profile.doing_business_as": {
			Type:        "string",
			Description: "The name which is used by the business.",
		},
		"identity.individual.documents.company_authorization.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.attestations.directorship_declaration.date": {
			Type:        "string",
			Description: "The time marking when the director attestation was made. Represented as a RFC 3339 date & time UTC value in millisecond precision, for example: 2022-09-18T13:22:18.123Z.",
			Format:      "date-time",
		},
		"identity.business_details.documents.proof_of_ultimate_beneficial_ownership.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.individual.script_addresses.kanji.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.business_details.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.business_details.documents.company_memorandum_of_association.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"contact_phone": {
			Type:        "string",
			Description: "The default contact phone for the Account.",
		},
		"identity.business_details.annual_revenue.amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
		},
		"identity.business_details.registered_name": {
			Type:        "string",
			Description: "The business legal name.",
		},
		"configuration.merchant.capabilities.blik_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.customer.automatic_indirect_tax.ip_address": {
			Type:        "string",
			Description: "A recent IP address of the customer used for tax reporting and tax location inference.",
		},
		"configuration.storer.capabilities.outbound_payments.cards.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.individual.documents.secondary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"identity.individual.email": {
			Type:        "string",
			Description: "The individual's email address.",
		},
		"identity.attestations.persons_provided.executives": {
			Type:        "boolean",
			Description: "Whether the company’s executives have been provided. Set this Boolean to true after creating all the company’s executives with the [Persons API](https://docs.stripe.com/api/v2/core/accounts/createperson).",
		},
		"configuration.merchant.capabilities.bancontact_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.mx_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.ach_debit_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.storer.capabilities.holds_currencies.usd.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.script_addresses.kana.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.business_details.script_addresses.kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
			Required:    true,
		},
		"configuration.customer.billing.invoice.rendering.amount_tax_display": {
			Type:        "string",
			Description: "Indicates whether displayed line item prices and amounts on invoice PDFs include inclusive tax amounts. Must be either `include_inclusive_tax` or `exclude_tax`.",
			Enum: []resource.EnumSpec{
				{Value: "exclude_tax"},
				{Value: "include_inclusive_tax"},
			},
		},
		"configuration.merchant.capabilities.us_bank_transfer_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.relationship.owner": {
			Type:        "boolean",
			Description: "Whether the person is an owner of the account’s identity.",
		},
		"identity.attestations.terms_of_service.account.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the Account's representative accepted the terms of service.",
		},
		"identity.business_details.address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"configuration.merchant.support.address.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.individual.documents.passport.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.script_addresses.kanji.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.individual.address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.individual.documents.primary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"identity.attestations.terms_of_service.storer.ip": {
			Type:        "string",
			Description: "The IP address from which the Account's representative accepted the terms of service.",
			Required:    true,
		},
		"configuration.merchant.capabilities.acss_debit_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.documents.company_authorization.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.business_details.annual_revenue.fiscal_year_end": {
			Type:        "string",
			Description: "The close-out date of the preceding fiscal year in ISO 8601 format. E.g. 2023-12-31 for the 31st of December, 2023.",
		},
		"identity.business_details.monthly_estimated_revenue.amount.value": {
			Type:        "integer",
			Description: "A non-negative integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#minor-units).",
			Required:    true,
		},
		"configuration.merchant.capabilities.grabpay_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.merchant.capabilities.au_becs_debit_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"defaults.locales": {
			Type:        "array",
			Description: "The Account's preferred locales (languages), ordered by preference.",
		},
		"identity.attestations.ownership_declaration.user_agent": {
			Type:        "string",
			Description: "The user agent of the browser from which the beneficial owner attestation was made.",
		},
		"identity.business_details.address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.business_details.monthly_estimated_revenue.amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
		},
		"identity.business_details.documents.company_ministerial_decree.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"configuration.merchant.support.email": {
			Type:        "string",
			Description: "A publicly available email address for sending support issues to.",
		},
		"configuration.storer.capabilities.outbound_transfers.bank_accounts.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.address.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.attestations.directorship_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the director attestation was made.",
		},
		"identity.business_details.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"configuration.storer.capabilities.holds_currencies.eur.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.business_details.documents.bank_account_ownership_verification.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"configuration.customer.shipping.address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"configuration.customer.shipping.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"configuration.merchant.script_statement_descriptor.kanji.descriptor": {
			Type:        "string",
			Description: "The default text that appears on statements for non-card charges outside of Japan. For card charges, if you don’t set a statement_descriptor_prefix, this text is also used as the statement descriptor prefix. In that case, if concatenating the statement descriptor suffix causes the combined statement descriptor to exceed 22 characters, we truncate the statement_descriptor text to limit the full descriptor to 22 characters. For more information about statement descriptors and their requirements, see the Merchant Configuration settings documentation.",
		},
		"configuration.merchant.konbini_payments.support.hours.start_time": {
			Type:        "string",
			Description: "Support hours start time (JST time of day) for in `HH:MM` format.",
		},
		"configuration.merchant.konbini_payments.support.phone": {
			Type:        "string",
			Description: "Support phone number for Konbini payments.",
		},
		"configuration.storer.capabilities.outbound_payments.bank_accounts.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.attestations.representative_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the representative attestation was made.",
		},
		"configuration.customer.shipping.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.individual.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.business_details.documents.bank_account_ownership_verification.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"configuration.customer.automatic_indirect_tax.exempt": {
			Type:        "string",
			Description: "Describes the customer's tax exemption status, which is `none`, `exempt`, or `reverse`. When set to reverse, invoice and receipt PDFs include the following text: “Reverse charge”.",
			Enum: []resource.EnumSpec{
				{Value: "exempt"},
				{Value: "none"},
				{Value: "reverse"},
			},
		},
		"configuration.customer.billing.invoice.footer": {
			Type:        "string",
			Description: "Default invoice footer.",
		},
		"configuration.merchant.support.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"configuration.merchant.capabilities.boleto_payments.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.individual.script_addresses.kanji.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.attestations.ownership_declaration.ip": {
			Type:        "string",
			Description: "The IP address from which the beneficial owner attestation was made.",
		},
		"identity.business_details.script_addresses.kana.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"configuration.customer.billing.invoice.next_sequence": {
			Type:        "integer",
			Description: "Sequence number to use on the customer account's next invoice. Defaults to 1.",
		},
		"configuration.customer.billing.invoice.rendering.template": {
			Type:        "string",
			Description: "ID of the invoice rendering template to use for future invoices.",
		},
		"configuration.customer.shipping.address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.business_details.documents.company_ministerial_decree.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.country": {
			Type:        "string",
			Description: "The country in which the account holder resides, or in which the business is legally established. This should be an [ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2) country code.",
		},
		"identity.individual.political_exposure": {
			Type:        "string",
			Description: "The individual's political exposure.",
			Enum: []resource.EnumSpec{
				{Value: "existing"},
				{Value: "none"},
			},
		},
		"identity.individual.documents.visa.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.attestations.terms_of_service.account.ip": {
			Type:        "string",
			Description: "The IP address from which the Account's representative accepted the terms of service.",
			Required:    true,
		},
		"identity.business_details.address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"configuration.customer.test_clock": {
			Type:        "string",
			Description: "ID of the test clock to attach to the customer. Can only be set on testmode Accounts, and when the Customer Configuration is first set on an Account.",
		},
		"configuration.recipient.capabilities.cards.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.storer.capabilities.outbound_transfers.financial_accounts.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"configuration.storer.capabilities.financial_addresses.bank_accounts.requested": {
			Type:        "boolean",
			Description: "To request a new Capability for an account, pass true. There can be a delay before the requested Capability becomes active.",
			Required:    true,
		},
		"identity.business_details.documents.company_tax_id_verification.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.business_details.documents.company_tax_id_verification.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
	},
}

var V2PreviewCoreAccountTokensRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/core/account_tokens/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve an Account Token",
}

var V2PreviewCoreAccountTokensCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/core/account_tokens",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create an Account Token",
	Params: map[string]*resource.ParamSpec{
		"identity.individual.relationship.title": {
			Type:        "string",
			Description: "The person's title (e.g., CEO, Support Engineer).",
		},
		"identity.individual.address.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.business_details.documents.company_license.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.script_names.kana.registered_name": {
			Type:        "string",
			Description: "Registered name of the business.",
		},
		"identity.business_details.annual_revenue.amount.value": {
			Type:        "integer",
			Description: "A non-negative integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#minor-units).",
			Required:    true,
		},
		"identity.business_details.script_addresses.kanji.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.business_details.documents.primary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"identity.business_details.address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.individual.legal_gender": {
			Type:        "string",
			Description: "The individual's gender (International regulations require either \"male\" or \"female\").",
			Enum: []resource.EnumSpec{
				{Value: "female"},
				{Value: "male"},
			},
		},
		"identity.individual.script_addresses.kanji.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.attestations.terms_of_service.storer.shown_and_accepted": {
			Type:        "boolean",
			Description: "The boolean value indicating if the terms of service have been accepted.",
		},
		"identity.individual.script_names.kana.surname": {
			Type:        "string",
			Description: "The person's last or family name.",
		},
		"identity.individual.script_addresses.kana.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.individual.script_addresses.kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.individual.documents.primary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"identity.business_details.documents.proof_of_registration.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.documents.company_registration_verification.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.business_details.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.business_details.documents.company_memorandum_of_association.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.individual.relationship.percent_ownership": {
			Type:        "string",
			Description: "The percent owned by the person of the account's legal entity.",
			Format:      "decimal",
		},
		"identity.individual.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.individual.documents.primary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"identity.business_details.documents.company_registration_verification.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.script_addresses.kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.individual.address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.individual.email": {
			Type:        "string",
			Description: "The individual's email address.",
		},
		"identity.business_details.address.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.individual.date_of_birth.year": {
			Type:        "integer",
			Description: "The year of birth.",
			Required:    true,
		},
		"identity.individual.relationship.director": {
			Type:        "boolean",
			Description: "Whether the person is a director of the account's identity. Directors are typically members of the governing board of the company, or responsible for ensuring the company meets its regulatory obligations.",
		},
		"identity.business_details.script_addresses.kana.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.business_details.script_addresses.kana.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.business_details.script_addresses.kanji.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.business_details.documents.proof_of_address.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.documents.company_tax_id_verification.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.business_details.address.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.business_details.address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.business_details.documents.company_ministerial_decree.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.individual.script_addresses.kana.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.individual.surname": {
			Type:        "string",
			Description: "The individual's last name.",
		},
		"identity.individual.relationship.executive": {
			Type:        "boolean",
			Description: "Whether the person has significant responsibility to control, manage, or direct the organization.",
		},
		"identity.business_details.structure": {
			Type:        "string",
			Description: "The category identifying the legal structure of the business.",
			Enum: []resource.EnumSpec{
				{Value: "cooperative"},
				{Value: "free_zone_establishment"},
				{Value: "free_zone_llc"},
				{Value: "governmental_unit"},
				{Value: "government_instrumentality"},
				{Value: "incorporated_association"},
				{Value: "incorporated_non_profit"},
				{Value: "incorporated_partnership"},
				{Value: "limited_liability_partnership"},
				{Value: "llc"},
				{Value: "multi_member_llc"},
				{Value: "private_company"},
				{Value: "private_corporation"},
				{Value: "private_partnership"},
				{Value: "public_company"},
				{Value: "public_corporation"},
				{Value: "public_listed_corporation"},
				{Value: "public_partnership"},
				{Value: "registered_charity"},
				{Value: "single_member_llc"},
				{Value: "sole_establishment"},
				{Value: "sole_proprietorship"},
				{Value: "tax_exempt_government_instrumentality"},
				{Value: "trust"},
				{Value: "unincorporated_association"},
				{Value: "unincorporated_non_profit"},
				{Value: "unincorporated_partnership"},
			},
		},
		"identity.business_details.documents.company_memorandum_of_association.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.business_details.documents.primary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"identity.business_details.monthly_estimated_revenue.amount.value": {
			Type:        "integer",
			Description: "A non-negative integer representing how much to charge in the [smallest currency unit](https://docs.stripe.com/currencies#minor-units).",
			Required:    true,
		},
		"identity.business_details.phone": {
			Type:        "string",
			Description: "The phone number of the Business Entity.",
		},
		"identity.attestations.persons_provided.executives": {
			Type:        "boolean",
			Description: "Whether the company’s executives have been provided. Set this Boolean to true after creating all the company’s executives with the [Persons API](https://docs.stripe.com/api/v2/core/accounts/createperson).",
		},
		"identity.attestations.directorship_declaration.attested": {
			Type:        "boolean",
			Description: "A boolean indicating if the directors information has been attested.",
		},
		"identity.business_details.registration_date.year": {
			Type:        "integer",
			Description: "The four-digit year of registration.",
			Required:    true,
		},
		"identity.individual.script_names.kanji.surname": {
			Type:        "string",
			Description: "The person's last or family name.",
		},
		"identity.individual.script_addresses.kanji.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.individual.documents.visa.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.individual.script_addresses.kana.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.individual.political_exposure": {
			Type:        "string",
			Description: "The individual's political exposure.",
			Enum: []resource.EnumSpec{
				{Value: "existing"},
				{Value: "none"},
			},
		},
		"identity.business_details.documents.bank_account_ownership_verification.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.documents.primary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"identity.business_details.script_addresses.kanji.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.individual.documents.secondary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"identity.attestations.persons_provided.ownership_exemption_reason": {
			Type:        "string",
			Description: "Reason for why the company is exempt from providing ownership information.",
			Enum: []resource.EnumSpec{
				{Value: "qualified_entity_exceeds_ownership_threshold"},
				{Value: "qualifies_as_financial_institution"},
			},
		},
		"identity.business_details.documents.proof_of_ultimate_beneficial_ownership.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.individual.relationship.owner": {
			Type:        "boolean",
			Description: "Whether the person is an owner of the account’s identity.",
		},
		"identity.attestations.ownership_declaration.attested": {
			Type:        "boolean",
			Description: "A boolean indicating if the beneficial owner information has been attested.",
		},
		"identity.business_details.documents.proof_of_registration.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.business_details.script_addresses.kana.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.business_details.script_addresses.kanji.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.individual.script_names.kana.given_name": {
			Type:        "string",
			Description: "The person's first or given name.",
		},
		"identity.individual.script_addresses.kana.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.individual.address.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.individual.address.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"display_name": {
			Type:        "string",
			Description: "A descriptive name for the Account. This name will be surfaced in the Stripe Dashboard and on any invoices sent to the Account.",
		},
		"identity.business_details.annual_revenue.fiscal_year_end": {
			Type:        "string",
			Description: "The close-out date of the preceding fiscal year in ISO 8601 format. E.g. 2023-12-31 for the 31st of December, 2023.",
		},
		"identity.business_details.documents.company_license.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.business_details.monthly_estimated_revenue.amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
		},
		"identity.individual.script_addresses.kana.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.individual.documents.passport.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.individual.documents.secondary_verification.front_back.front": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the front of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"contact_phone": {
			Type:        "string",
			Description: "The default contact phone for the Account.",
		},
		"identity.business_details.script_addresses.kanji.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.individual.given_name": {
			Type:        "string",
			Description: "The individual's first name.",
		},
		"identity.individual.script_addresses.kanji.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.individual.address.postal_code": {
			Type:        "string",
			Description: "ZIP or postal code.",
		},
		"identity.individual.documents.company_authorization.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.individual.documents.passport.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.attestations.persons_provided.owners": {
			Type:        "boolean",
			Description: "Whether the company’s owners have been provided. Set this Boolean to true after creating all the company’s owners with the [Persons API](https://docs.stripe.com/api/v2/core/accounts/createperson).",
		},
		"identity.individual.script_addresses.kana.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.individual.documents.company_authorization.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.individual.script_addresses.kanji.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.individual.script_addresses.kanji.line1": {
			Type:        "string",
			Description: "Address line 1 (e.g., street, PO Box, or company name).",
		},
		"identity.individual.date_of_birth.month": {
			Type:        "integer",
			Description: "The month of birth.",
			Required:    true,
		},
		"identity.individual.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.business_details.estimated_worker_count": {
			Type:        "integer",
			Description: "Estimated maximum number of workers currently engaged by the business (including employees, contractors, and vendors).",
		},
		"identity.business_details.registered_name": {
			Type:        "string",
			Description: "The business legal name.",
		},
		"identity.attestations.representative_declaration.attested": {
			Type:        "boolean",
			Description: "A boolean indicating if the representative is authorized to act as the representative of their legal entity.",
		},
		"identity.business_details.documents.proof_of_ultimate_beneficial_ownership.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.registration_date.month": {
			Type:        "integer",
			Description: "The month of registration, between 1 and 12.",
			Required:    true,
		},
		"identity.individual.documents.primary_verification.front_back.back": {
			Type:        "string",
			Description: "A [file upload](https://docs.stripe.com/api/persons/update#create_file) token representing the back of the verification document. The purpose of the uploaded file should be 'identity_document'. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.",
		},
		"contact_email": {
			Type:        "string",
			Description: "The default contact email address for the Account. Required when configuring the account as a merchant or recipient.",
		},
		"identity.attestations.persons_provided.directors": {
			Type:        "boolean",
			Description: "Whether the company’s directors have been provided. Set this Boolean to true after creating all the company’s directors with the [Persons API](https://docs.stripe.com/api/v2/core/accounts/createperson).",
		},
		"identity.business_details.documents.proof_of_address.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.business_details.documents.company_ministerial_decree.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.script_addresses.kana.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.entity_type": {
			Type:        "string",
			Description: "The entity type.",
			Enum: []resource.EnumSpec{
				{Value: "company"},
				{Value: "government_entity"},
				{Value: "individual"},
				{Value: "non_profit"},
			},
		},
		"identity.individual.documents.secondary_verification.type": {
			Type:        "string",
			Description: "The format of the verification document. Currently supports `front_back` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "front_back"},
			},
		},
		"identity.individual.documents.visa.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.attestations.terms_of_service.account.shown_and_accepted": {
			Type:        "boolean",
			Description: "The boolean value indicating if the terms of service have been accepted.",
		},
		"identity.business_details.script_addresses.kana.line2": {
			Type:        "string",
			Description: "Address line 2 (e.g., apartment, suite, unit, or building).",
		},
		"identity.business_details.script_addresses.kanji.state": {
			Type:        "string",
			Description: "State, county, province, or region.",
		},
		"identity.individual.script_names.kanji.given_name": {
			Type:        "string",
			Description: "The person's first or given name.",
		},
		"identity.individual.nationalities": {
			Type:        "array",
			Description: "The countries where the individual is a national. Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.individual.phone": {
			Type:        "string",
			Description: "The individual's phone number.",
		},
		"identity.individual.script_addresses.kana.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
		"identity.business_details.script_names.kanji.registered_name": {
			Type:        "string",
			Description: "Registered name of the business.",
		},
		"identity.business_details.script_addresses.kana.town": {
			Type:        "string",
			Description: "Town or district.",
		},
		"identity.business_details.documents.bank_account_ownership_verification.files": {
			Type:        "array",
			Description: "One or more document IDs returned by a [file upload](https://docs.stripe.com/api/persons/update#create_file) with a purpose value of `account_requirement`.",
			Required:    true,
		},
		"identity.business_details.documents.company_tax_id_verification.type": {
			Type:        "string",
			Description: "The format of the document. Currently supports `files` only.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "files"},
			},
		},
		"identity.business_details.registration_date.day": {
			Type:        "integer",
			Description: "The day of registration, between 1 and 31.",
			Required:    true,
		},
		"identity.business_details.address.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.business_details.script_addresses.kanji.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.individual.script_addresses.kanji.city": {
			Type:        "string",
			Description: "City, district, suburb, town, or village.",
		},
		"identity.individual.date_of_birth.day": {
			Type:        "integer",
			Description: "The day of the birth.",
			Required:    true,
		},
		"identity.business_details.annual_revenue.amount.currency": {
			Type:        "string",
			Description: "Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).",
			Required:    true,
		},
		"identity.business_details.address.country": {
			Type:        "string",
			Description: "Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).",
		},
	},
}

var V2PreviewCoreEventDestinationsUpdate = resource.OperationSpec{
	Name:      "update",
	Path:      "/v2/core/event_destinations/{id}",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Update an Event Destination",
	Params: map[string]*resource.ParamSpec{
		"description": {
			Type:        "string",
			Description: "An optional description of what the event destination is used for.",
		},
		"enabled_events": {
			Type:        "array",
			Description: "The list of events to enable for this endpoint.",
		},
		"include": {
			Type:        "array",
			Description: "Additional fields to include in the response. Currently supports `webhook_endpoint.url`.",
		},
		"name": {
			Type:        "string",
			Description: "Event destination name.",
		},
		"webhook_endpoint.url": {
			Type:        "string",
			Description: "The URL of the webhook endpoint.",
			Required:    true,
		},
	},
}

var V2PreviewCoreEventDestinationsDisable = resource.OperationSpec{
	Name:      "disable",
	Path:      "/v2/core/event_destinations/{id}/disable",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Disable an Event Destination",
}

var V2PreviewCoreEventDestinationsEnable = resource.OperationSpec{
	Name:      "enable",
	Path:      "/v2/core/event_destinations/{id}/enable",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Enable an Event Destination",
}

var V2PreviewCoreEventDestinationsPing = resource.OperationSpec{
	Name:      "ping",
	Path:      "/v2/core/event_destinations/{id}/ping",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Ping an Event Destination",
}

var V2PreviewCoreEventDestinationsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/core/event_destinations",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Event Destinations",
	Params: map[string]*resource.ParamSpec{
		"include": {
			Type:        "array",
			Description: "Additional fields to include in the response. Currently supports `webhook_endpoint.url`.",
		},
		"limit": {
			Type:        "integer",
			Description: "The page size.",
		},
		"page": {
			Type:        "string",
			Description: "The requested page.",
		},
	},
}

var V2PreviewCoreEventDestinationsCreate = resource.OperationSpec{
	Name:      "create",
	Path:      "/v2/core/event_destinations",
	Method:    "POST",
	IsPreview: true,
	Summary:   "Create an Event Destination",
	Params: map[string]*resource.ParamSpec{
		"amazon_eventbridge.aws_account_id": {
			Type:        "string",
			Description: "The AWS account ID.",
			Required:    true,
		},
		"amazon_eventbridge.aws_region": {
			Type:        "string",
			Description: "The region of the AWS event source.",
			Required:    true,
		},
		"enabled_events": {
			Type:        "array",
			Description: "The list of events to enable for this endpoint.",
			Required:    true,
		},
		"description": {
			Type:        "string",
			Description: "An optional description of what the event destination is used for.",
		},
		"events_from": {
			Type:        "array",
			Description: "Where events should be routed from.",
		},
		"include": {
			Type:        "array",
			Description: "Additional fields to include in the response.",
		},
		"name": {
			Type:        "string",
			Description: "Event destination name.",
			Required:    true,
		},
		"event_payload": {
			Type:        "string",
			Description: "Payload type of events being subscribed to.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "snapshot"},
				{Value: "thin"},
			},
		},
		"snapshot_api_version": {
			Type:        "string",
			Description: "If using the snapshot event payload, the API version events are rendered as.",
		},
		"type": {
			Type:        "string",
			Description: "Event destination type.",
			Required:    true,
			Enum: []resource.EnumSpec{
				{Value: "amazon_eventbridge"},
				{Value: "webhook_endpoint"},
			},
		},
		"webhook_endpoint.url": {
			Type:        "string",
			Description: "The URL of the webhook endpoint.",
			Required:    true,
		},
	},
}

var V2PreviewCoreEventDestinationsDelete = resource.OperationSpec{
	Name:      "delete",
	Path:      "/v2/core/event_destinations/{id}",
	Method:    "DELETE",
	IsPreview: true,
	Summary:   "Delete an Event Destination",
}

var V2PreviewCoreEventDestinationsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/core/event_destinations/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve an Event Destination",
	Params: map[string]*resource.ParamSpec{
		"include": {
			Type:        "array",
			Description: "Additional fields to include in the response.",
		},
	},
}

var V2PreviewCoreEventsList = resource.OperationSpec{
	Name:      "list",
	Path:      "/v2/core/events",
	Method:    "GET",
	IsPreview: true,
	Summary:   "List Events",
	Params: map[string]*resource.ParamSpec{
		"limit": {
			Type:        "integer",
			Description: "The page size.",
		},
		"object_id": {
			Type:        "string",
			Description: "Primary object ID used to retrieve related events.",
		},
		"page": {
			Type:        "string",
			Description: "The requested page.",
		},
		"types": {
			Type:        "array",
			Description: "An array of up to 20 strings containing specific event names.",
		},
	},
}

var V2PreviewCoreEventsRetrieve = resource.OperationSpec{
	Name:      "retrieve",
	Path:      "/v2/core/events/{id}",
	Method:    "GET",
	IsPreview: true,
	Summary:   "Retrieve an Event",
}
