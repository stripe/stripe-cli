package samples

import (
	"fmt"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// SampleData stores the information needed for Stripe Samples to operate in
// the CLI
type SampleData struct {
	Name        string
	URL         string
	Description string
}

// BoldName returns an ansi bold string for the name
func (sd *SampleData) BoldName() string {
	return ansi.Bold(sd.Name)
}

// GitRepo returns a string of the repo with the .git prefix
func (sd *SampleData) GitRepo() string {
	return fmt.Sprintf("%s.git", sd.URL)
}

var addingSalesTax = &SampleData{
	Name:        "adding-sales-tax",
	Description: "Learn how to use PaymentIntents to build a simple checkout flow",
	URL:         "https://github.com/stripe-samples/adding-sales-tax",
}

var checkoutSubscriptionAndAddOn = &SampleData{
	Name:        "checkout-subscription-and-add-on",
	Description: "Uses Stripe Checkout to create a payment page that starts a subscription for a new customer",
	URL:         "https://github.com/stripe-samples/checkout-subscription-and-add-on",
}

var placingAHold = &SampleData{
	Name:        "placing-a-hold",
	Description: "Learn how to place a hold on a credit card (split auth / capture)",
	URL:         "https://github.com/stripe-samples/placing-a-hold",
}

var paymentFormModal = &SampleData{
	Name:        "payment-form-modal",
	Description: "How to implement Stripe Elements within a modal dialog",
	URL:         "https://github.com/stripe-samples/payment-form-modal",
}

var savingCardWithoutPayment = &SampleData{
	Name:        "saving-card-without-payment",
	Description: "How to build a form to save a credit card without taking a payment",
	URL:         "https://github.com/stripe-samples/saving-card-without-payment",
}

var checkoutOneTimePayments = &SampleData{
	Name:        "checkout-one-time-payments",
	Description: "Use Checkout to quickly collect one-time payments",
	URL:         "https://github.com/stripe-samples/checkout-one-time-payments",
}

var checkoutSingleSubscription = &SampleData{
	Name:        "checkout-single-subscription",
	Description: "Learn how to combine Checkout and Billing for fast subscription pages",
	URL:         "https://github.com/stripe-samples/checkout-single-subscription",
}

var webElementsCardPayment = &SampleData{
	Name:        "web-elements-card-payment",
	Description: "Learn how to accept a basic card payment on the web",
	URL:         "https://github.com/stripe-samples/web-elements-card-payment",
}

var savingCardAfterPayment = &SampleData{
	Name:        "saving-card-after-payment",
	Description: "Learn how to save a card for later reuse after making a payment",
	URL:         "https://github.com/stripe-samples/saving-card-after-payment",
}

var reactElementsCardPayment = &SampleData{
	Name:        "react-elements-card-payment",
	Description: "Learn how to build a checkout form with React",
	URL:         "https://github.com/stripe-samples/react-elements-card-payment",
}

// List contains a mapping of Stripe Samples that we want to be available in the
// CLI to some of their metadata
// TODO: what do we want to name these for it to be easier for users to select?
var List = map[string]*SampleData{
	"adding-sales-tax":                 addingSalesTax,
	"checkout-subscription-and-add-on": checkoutSubscriptionAndAddOn,
	"placing-a-hold":                   placingAHold,
	"payment-form-modal":               paymentFormModal,
	"saving-card-without-payment":      savingCardWithoutPayment,
	"checkout-one-time-payments":       checkoutOneTimePayments,
	"checkout-single-subscription":     checkoutSingleSubscription,
	"web-elements-card-payment":        webElementsCardPayment,
	"saving-card-after-payment":        savingCardAfterPayment,
	"react-elements-card-payment":      reactElementsCardPayment,
}

// Names returns a list of all the sample's names
func Names() []string {
	keys := make([]string, 0, len(List))
	for k := range List {
		keys = append(keys, k)
	}

	return keys
}
