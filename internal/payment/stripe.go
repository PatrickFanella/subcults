// Package payment provides Stripe integration for payment processing and Connect onboarding.
package payment

import (
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/account"
	"github.com/stripe/stripe-go/v81/accountlink"
	"github.com/stripe/stripe-go/v81/checkout/session"
)

// CheckoutSessionParams represents parameters for creating a Checkout Session.
type CheckoutSessionParams struct {
	ConnectedAccountID string
	Items              []CheckoutItem
	SuccessURL         string
	CancelURL          string
	ApplicationFee     int64 // Fee in cents
	UserDID            string
}

// CheckoutItem represents a line item for checkout.
type CheckoutItem struct {
	PriceID  string
	Quantity int64
}

// Client is an interface for Stripe operations to enable testing with mocks.
type Client interface {
	CreateConnectAccount() (*stripe.Account, error)
	CreateAccountLink(accountID, returnURL, refreshURL string) (*stripe.AccountLink, error)
	CreateCheckoutSession(params *CheckoutSessionParams) (*stripe.CheckoutSession, error)
}

// StripeClient implements the Client interface using the real Stripe SDK.
type StripeClient struct{}

// NewStripeClient creates a new Stripe client with the given API key.
func NewStripeClient(apiKey string) *StripeClient {
	stripe.Key = apiKey
	return &StripeClient{}
}

// CreateConnectAccount creates a new Stripe Connect Express account.
func (c *StripeClient) CreateConnectAccount() (*stripe.Account, error) {
	params := &stripe.AccountParams{
		Type: stripe.String(string(stripe.AccountTypeExpress)),
	}

	return account.New(params)
}

// CreateAccountLink creates an account onboarding link for a Stripe Connect account.
func (c *StripeClient) CreateAccountLink(accountID, returnURL, refreshURL string) (*stripe.AccountLink, error) {
	params := &stripe.AccountLinkParams{
		Account:    stripe.String(accountID),
		ReturnURL:  stripe.String(returnURL),
		RefreshURL: stripe.String(refreshURL),
		Type:       stripe.String("account_onboarding"),
	}

	return accountlink.New(params)
}

// CreateCheckoutSession creates a Stripe Checkout Session with platform fee and Connect account.
func (c *StripeClient) CreateCheckoutSession(params *CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	lineItems := make([]*stripe.CheckoutSessionLineItemParams, len(params.Items))
	for i, item := range params.Items {
		lineItems[i] = &stripe.CheckoutSessionLineItemParams{
			Price:    stripe.String(item.PriceID),
			Quantity: stripe.Int64(item.Quantity),
		}
	}

	sessionParams := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems:  lineItems,
		SuccessURL: stripe.String(params.SuccessURL),
		CancelURL:  stripe.String(params.CancelURL),
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			ApplicationFeeAmount: stripe.Int64(params.ApplicationFee),
			OnBehalfOf:           stripe.String(params.ConnectedAccountID),
		},
	}

	// Create the session first to get the session ID
	sess, err := session.New(sessionParams)
	if err != nil {
		return nil, err
	}

	// NOTE: We cannot set metadata on the PaymentIntent at session creation time
	// because the PaymentIntent doesn't exist yet. Stripe creates the PaymentIntent
	// after the session is created. To work around this, webhook handlers should
	// look up payment records by payment_intent_id in the database rather than
	// relying on metadata. This is documented in webhook_handlers.go.

	return sess, nil
}
