package p400

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"time"

	"github.com/stripe/stripe-cli/pkg/version"
)

// TerminalSessionContext is a type that contains important context most methods need to know to complete the quickstart flow
// one copy of this is passed around a lot and is mutable for whenever a property needs to change
type TerminalSessionContext struct {
	APIKey             string
	IPAddress          string
	BaseURL            string
	LocationID         string
	PstToken           string
	SessionToken       string
	TransactionID      int
	MethodID           int
	TransactionContext TransactionContext
	DeviceInfo         DeviceInfo
	PaymentIntentID    string
	Amount             int
	Currency           string
}

// DeviceInfo belongs to the Rabbit Service RPC call payload shape
type DeviceInfo struct {
	DeviceClass   string        `json:"device_class"`
	DeviceUUID    string        `json:"device_uuid"`
	HostOSVersion string        `json:"host_os_version"`
	HardwareModel HardwareModel `json:"hardware_model"`
	AppModel      AppModel      `json:"app_model"`
}

// HardwareModel belongs to the Rabbit Service RPC call payload shape
type HardwareModel struct {
	POSInfo POSInfo `json:"pos_info"`
}

// AppModel belongs to the Rabbit Service RPC call payload shape
type AppModel struct {
	AppID      string `json:"app_id"`
	AppVersion string `json:"app_version"`
}

// POSInfo belongs to the Rabbit Service RPC call payload shape
type POSInfo struct {
	Description string `json:"description"`
}

// POSSoftwareInfo belongs to the Rabbit Service RPC call payload shape
type POSSoftwareInfo struct {
	POSType    string `json:"pos_type"`
	SdkVersion string `json:"sdk_version"`
}

// PaymentMethod exists only so that you can pass a pointer reference of it to generate an actual empty payment method object when using JSON serialization and have it not be serialized as null
type PaymentMethod struct {
}

// ReaderActivateContent represents the shape of the serialized protobuf sent to Rabbit Service for activating a terminal session
type ReaderActivateContent struct {
	POSActivationToken string          `json:"pos_activation_token"`
	StoreName          string          `json:"store_name"`
	POSDeviceID        string          `json:"pos_device_id"`
	POSSoftwareInfo    POSSoftwareInfo `json:"pos_software_info"`
}

// ReaderActivateResponse is the RPC response from calling the activateTerminal method
type ReaderActivateResponse struct {
	SessionToken string `json:"session_token"`
}

// LineItem belongs to the Cart protobuf shape for updating the reader display via Rabbit Service
type LineItem struct {
	Description string `json:"description"`
	Amount      int    `json:"amount"`
	Quantity    int    `json:"quantity"`
}

// Cart belongs to the ReaderDisplayContent protobuf sent to Rabbit Service that updates the reader display
type Cart struct {
	LineItems []LineItem `json:"line_items"`
	Tax       int        `json:"tax"`
	Total     int        `json:"total"`
	Currency  string     `json:"currency"`
}

// ReaderDisplayContent represents the shape of the serialized protobuf sent to Rabbit Service for updating the reader display
type ReaderDisplayContent struct {
	Type               string             `json:"type"`
	Cart               Cart               `json:"cart"`
	TransactionContext TransactionContext `json:"transaction_context"`
}

// ReaderDisplayClearContent represents the shape of the serialized protobuf sent to Rabbit Service for clearing the reader display
type ReaderDisplayClearContent struct {
	TransactionContext TransactionContext `json:"transaction_context"`
}

// ChargeAmount belongs to the ReaderCollectPaymentContent protobuf send to Rabbit Service for collecting a specific payment after the Payment Intent is created
type ChargeAmount struct {
	ChargeAmount   int    `json:"charge_amount"`
	Currency       string `json:"currency"`
	TipAmount      int    `json:"tip_amount"`
	CashbackAmount int    `json:"cashback_amount"`
}

// ReaderCollectPaymentContent represents the shape of the serialized protobuf sent to Rabbit Service for collecting a payment for a specific Payment Intent
type ReaderCollectPaymentContent struct {
	ChargeAmount       ChargeAmount       `json:"charge_amount"`
	TransactionContext TransactionContext `json:"transaction_context"`
}

// ReaderQueryPaymentContent represents the shape of the serialized protobuf sent to Rabbit Service for querying a payment for its collection state
type ReaderQueryPaymentContent struct {
	TransactionContext TransactionContext `json:"transaction_context"`
}

// ReaderQueryPaymentResponse is the decoded RPC response from calling the queryPaymentMethod reader method
type ReaderQueryPaymentResponse struct {
	PaymentMethod interface{} `json:"payment_method"`
	PaymentStatus string      `json:"payment_status"`
}

// ReaderConfirmPaymentContent represents the shape of the serialized protobuf sent to Rabbit Service for confirming a payment after it is collectedf
type ReaderConfirmPaymentContent struct {
	PaymentIntentID    string             `json:"payment_intent_id"`
	PaymentMethod      interface{}        `json:"payment_method"`
	TransactionContext TransactionContext `json:"transaction_context"`
}

// ReaderConfirmPaymentResponse is the decoded RPC response from calling the processPayment reader method
type ReaderConfirmPaymentResponse struct {
	SystemContext          interface{}            `json:"system_context"`
	RequestID              string                 `json:"request_id"`
	ConfirmedPaymentIntent ConfirmedPaymentIntent `json:"confirmed_payment_intent"`
}

// ConfirmedPaymentIntent belongs to ReaderConfirmPaymentResponse
type ConfirmedPaymentIntent struct {
	PaymentMethod string `json:"payment_method"`
}

// TransactionContext belongs to each Rabbit Service protobuf payload and communicates the state and identity of the current transaction
type TransactionContext struct {
	TerminalID    string `json:"terminal_id"`
	StartTime     int64  `json:"start_time"`
	OperatorID    string `json:"operator_id"`
	TransactionID string `json:"transaction_id"`
}

var rabbitMethods = []string{
	"activateTerminal",
	"setReaderDisplay",
	"clearReaderDisplay",
	"collectPaymentMethod",
	"queryPaymentMethod",
	"confirmPayment",
}

const (
	activateTerminal = iota
	setReaderDisplay
	clearReaderDisplay
	collectPaymentMethod
	queryPaymentMethod
	confirmPayment
)

// SetParentTraceID creates a string in a specific format for other methods to use for communicating which transaction a Rabbit Service call is concerning
// it returns the created trace id string
func SetParentTraceID(transactionID int, methodID int, methodName string) string {
	return fmt.Sprintf("txn!%v>%s!%v", transactionID, methodName, methodID)
}

// GetOSString finds which operating system the user is running and creates the correct string name for it to report to Rabbit Service when making a call
// this is mostly used by the TransactionContext properties
func GetOSString() string {
	var osString string

	switch plat := runtime.GOOS; plat {
	case "darwin":
		osString = "Mac OS"
	case "linux":
		osString = "Linux"
	case "windows":
		osString = "Windows"
	default:
		osString = "Unknown"
	}

	return osString
}

// GeneratePOSDeviceID creates a pseudorandom alpha string id for a point-of-sale quasi-unique identifier
// mostly used for TransactionContext related tracking / tracing and is semi-persistent to a machine but not determinant
func GeneratePOSDeviceID(seed int64) string {
	rand.Seed(seed)

	idLength := 11
	bytes := make([]byte, idLength)

	for i := 0; i < idLength; i++ {
		// pulling from ascii table a-z (97-122)
		bytes[i] = byte(97 + rand.Intn(122-97))
	}

	id := fmt.Sprintf("pos-%v", string(bytes))

	return id
}

// SetTransactionContext creates a new transaction context with a baked in start time and unique id
func SetTransactionContext(tsCtx TerminalSessionContext) TransactionContext {
	startTime := time.Now().Unix() * 1000
	transactionID := int(rand.Float64() * 100000)

	transactionContext := TransactionContext{
		TerminalID:    tsCtx.DeviceInfo.DeviceUUID,
		OperatorID:    tsCtx.DeviceInfo.DeviceUUID,
		StartTime:     startTime,
		TransactionID: strconv.Itoa(transactionID),
	}

	return transactionContext
}

// ActivateTerminalRPCSession calls Rabbit Service for a new reader session and returns the resulting session token
func ActivateTerminalRPCSession(tsCtx TerminalSessionContext) (string, error) {
	methodID := int(rand.Float64() * 100000)
	activateTraceID := fmt.Sprintf("connectReader!%v", methodID)

	activateTermContent := &ReaderActivateContent{
		POSActivationToken: tsCtx.PstToken,
		StoreName:          "empty",
		POSDeviceID:        tsCtx.DeviceInfo.DeviceUUID,
		POSSoftwareInfo: POSSoftwareInfo{
			POSType:    "pos-cli",
			SdkVersion: version.Version,
		},
	}

	var readerActivateResponse ReaderActivateResponse

	err := CallRabbitService(tsCtx, rabbitMethods[activateTerminal], activateTermContent, &readerActivateResponse, activateTraceID)

	if err != nil {
		if err.Error() == ErrDNSFailed.Error() {
			return "", err
		}

		return "", ErrActivateReaderFailed
	}

	newSessionToken := readerActivateResponse.SessionToken

	return newSessionToken, nil
}

// SetReaderDisplay calls Rabbit Service to set a cart's contents on the reader display
func SetReaderDisplay(tsCtx TerminalSessionContext, parentTraceID string) error {
	setReaderDisplayContent := &ReaderDisplayContent{
		Type: "cart",
		Cart: Cart{
			LineItems: []LineItem{{
				Description: "Stripe CLI Test Payment",
				Amount:      tsCtx.Amount,
				Quantity:    1,
			}},
			Tax:      0,
			Total:    tsCtx.Amount,
			Currency: tsCtx.Currency,
		},
		TransactionContext: tsCtx.TransactionContext,
	}

	var setReaderDisplayResponse interface{}

	err := CallRabbitService(tsCtx, rabbitMethods[setReaderDisplay], setReaderDisplayContent, &setReaderDisplayResponse, parentTraceID)

	if err != nil {
		return ErrSetReaderDisplayFailed
	}

	return nil
}

// CollectPaymentMethod calls Rabbit Service to put reader in payment collection state
func CollectPaymentMethod(tsCtx TerminalSessionContext, parentTraceID string) error {
	collectPaymentMethodContent := &ReaderCollectPaymentContent{
		ChargeAmount: ChargeAmount{
			ChargeAmount:   tsCtx.Amount,
			Currency:       tsCtx.Currency,
			CashbackAmount: 0,
			TipAmount:      0,
		},
		TransactionContext: tsCtx.TransactionContext,
	}

	var collectPaymentResponse interface{}
	err := CallRabbitService(tsCtx, rabbitMethods[collectPaymentMethod], collectPaymentMethodContent, &collectPaymentResponse, parentTraceID)

	if err != nil {
		return ErrCollectPaymentFailed
	}

	return nil
}

// ConfirmPayment calls Rabbit Service to confirm the payment that it collected, using Payment Intent and Payment Method to do so
func ConfirmPayment(tsCtx TerminalSessionContext, paymentMethod interface{}, parentTraceID string) (string, error) {
	confirmPaymentContent := &ReaderConfirmPaymentContent{
		PaymentIntentID:    tsCtx.PaymentIntentID,
		PaymentMethod:      &PaymentMethod{},
		TransactionContext: tsCtx.TransactionContext,
	}

	var confirmPaymentResponse ReaderConfirmPaymentResponse
	err := CallRabbitService(tsCtx, rabbitMethods[confirmPayment], confirmPaymentContent, &confirmPaymentResponse, parentTraceID)

	if err != nil {
		return "", ErrConfirmPaymentFailed
	}

	paymentMethodID := confirmPaymentResponse.ConfirmedPaymentIntent.PaymentMethod

	return paymentMethodID, nil
}

// WaitForPaymentCollection is a recursive function that calls Rabbit Service to query the status of the payment that is waiting to be collected (ie. user booping card on the reader)
// it exits if it either errors querying the payment, or the payment is successfully collected
// it additionally times out with an error after querying for around 60 seconds
// returns the payment method collected by the reader
func WaitForPaymentCollection(tsCtx TerminalSessionContext, parentTraceID string, tries int) (interface{}, error) {
	queryResult, err := QueryPaymentMethod(tsCtx, parentTraceID)

	if err != nil {
		return nil, err
	}

	if queryResult.PaymentStatus == "PAYMENT_PENDING" {
		tries++
		// below timeout is roughly 60 seconds
		if tries > 120 {
			err := ErrCollectPaymentTimeout
			return nil, err
		}
	} else if queryResult.PaymentMethod != nil {
		// payment method successfully collected
		return queryResult.PaymentMethod, nil
	}

	time.Sleep(200 * time.Millisecond)

	return WaitForPaymentCollection(tsCtx, parentTraceID, tries)
}

// QueryPaymentMethod calls Rabbit Service to query the status of a payment currently being collected
func QueryPaymentMethod(tsCtx TerminalSessionContext, parentTraceID string) (ReaderQueryPaymentResponse, error) {
	queryPaymentMethodContent := &ReaderQueryPaymentContent{
		TransactionContext: tsCtx.TransactionContext,
	}

	var queryPaymentResponse ReaderQueryPaymentResponse

	err := CallRabbitService(tsCtx, rabbitMethods[queryPaymentMethod], queryPaymentMethodContent, &queryPaymentResponse, parentTraceID)

	if err != nil {
		return queryPaymentResponse, ErrQueryPaymentFailed
	}

	return queryPaymentResponse, nil
}

// ClearReaderDisplay calls Rabbit Service and sets the reader display back to the splash screen and ends any payment collection status
func ClearReaderDisplay(tsCtx TerminalSessionContext) error {
	parentTraceID := SetParentTraceID(tsCtx.TransactionID, tsCtx.MethodID, "disconnectReader")

	clearReaderDisplayContent := &ReaderDisplayClearContent{
		TransactionContext: tsCtx.TransactionContext,
	}

	var clearReaderDisplayResponse interface{}

	err := CallRabbitService(tsCtx, rabbitMethods[clearReaderDisplay], clearReaderDisplayContent, &clearReaderDisplayResponse, parentTraceID)

	if err != nil {
		return ErrClearReaderDisplayFailed
	}

	return nil
}
