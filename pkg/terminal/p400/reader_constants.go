package p400

var (
	readerURL                          = "https://%v.%v:4443/protojsonservice/JackRabbitService"
	stripeTerminalReadersPath          = "/v1/terminal/readers?device_type=verifone_P400&limit=100&compatible_sdk_type=js&compatible_sdk_version=1.3.2"
	rpcSessionPath                     = "/v1/terminal/connection_tokens/generate_pos_rpc_session"
	stripeTerminalConnectionTokensPath = "/v1/terminal/connection_tokens"
	stripeTerminalRegisterPath         = "/v1/terminal/readers"
	stripeCreatePaymentIntentPath      = "/v1/payment_intents"
	stripeCapturePaymentIntentPath     = "/v1/payment_intents/%v/capture"
)
