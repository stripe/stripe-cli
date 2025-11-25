package gen

// OpenAPI spec file paths shared across code generation tools
const (
	// Legacy spec paths (separate files for v1, v2, v2 Preview)
	PathStripeSpec          = "../../api/openapi-spec/spec3.sdk.json"
	PathStripeSpecV2        = "../../api/openapi-spec/spec3.v2.sdk.json"
	PathStripeSpecV2Preview = "../../api/openapi-spec/spec3.v2.sdk.preview.json"

	// Unified spec paths (v1+v2 combined in single files)
	PathUnifiedSpec        = "../../api/openapi-spec/spec3.cli.json"
	PathUnifiedPreviewSpec = "../../api/openapi-spec/spec3.cli.preview.json"
)
