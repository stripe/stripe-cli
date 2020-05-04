package p400

var (
	readerURL                          = "https://%v.device.stripe-terminal-local-reader.net:4443/protojsonservice/JackRabbitService"
	stripeTerminalReadersPath          = "/v1/terminal/readers?device_type=verifone_P400"
	rpcSessionPath                     = "/v1/terminal/connection_tokens/generate_pos_rpc_session"
	stripeTerminalConnectionTokensPath = "/v1/terminal/connection_tokens"
	stripeTerminalRegisterPath         = "/v1/terminal/readers"
	stripeCreatePaymentIntentPath      = "/v1/payment_intents"
	stripeCapturePaymentIntentPath     = "/v1/payment_intents/%v/capture"
)
