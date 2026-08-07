// Project 1: Payment Gateway
//
// A single Gateway processes payments through ANY type satisfying
// PaymentProcessor — credit card, PayPal, or crypto — without ever knowing
// which concrete type it's holding. Demonstrates interfaces, composition via
// embedding, type assertions for optional behavior (refunds), the empty
// interface for freeform metadata, and a small reflection-based receipt
// printer.
package main

import (
	"fmt"
	"reflect"
)

// PaymentProcessor is the core interface — ANY type with this one method
// can be handed to Gateway.Charge, no matter how differently each one
// actually works internally.
type PaymentProcessor interface {
	Process(amount float64) (transactionID string, err error)
}

// Refundable is a SEPARATE, smaller interface. Not every processor needs to
// support refunds, so this isn't folded into PaymentProcessor itself —
// callers check for it with a type assertion when they need it.
type Refundable interface {
	Refund(transactionID string, amount float64) error
}

// Logger is embedded into every processor below, so each one gets a
// promoted Log method for free — composition, not inheritance.
type Logger struct {
	Prefix string
}

func (l Logger) Log(msg string) {
	fmt.Printf("[%s] %s\n", l.Prefix, msg)
}

// --- Credit Card -----------------------------------------------------

type CreditCardProcessor struct {
	Logger
	cardNumber string
	nextTxID   int
}

func NewCreditCardProcessor(cardNumber string) *CreditCardProcessor {
	return &CreditCardProcessor{Logger: Logger{Prefix: "CARD"}, cardNumber: cardNumber, nextTxID: 1000}
}

func (c *CreditCardProcessor) Process(amount float64) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}
	txID := fmt.Sprintf("CARD-%d", c.nextTxID)
	c.nextTxID++
	c.Log(fmt.Sprintf("charged %.2f to card ending %s (tx %s)", amount, lastFour(c.cardNumber), txID))
	return txID, nil
}

// Refund makes *CreditCardProcessor satisfy Refundable — PayPal (below)
// will too, but Crypto deliberately won't, to show the type-assertion path
// actually branching differently per processor.
func (c *CreditCardProcessor) Refund(transactionID string, amount float64) error {
	c.Log(fmt.Sprintf("refunded %.2f for %s", amount, transactionID))
	return nil
}

func lastFour(s string) string {
	if len(s) < 4 {
		return s
	}
	return s[len(s)-4:]
}

// --- PayPal -----------------------------------------------------

type PayPalProcessor struct {
	Logger
	email    string
	nextTxID int
}

func NewPayPalProcessor(email string) *PayPalProcessor {
	return &PayPalProcessor{Logger: Logger{Prefix: "PAYPAL"}, email: email, nextTxID: 5000}
}

func (p *PayPalProcessor) Process(amount float64) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}
	txID := fmt.Sprintf("PP-%d", p.nextTxID)
	p.nextTxID++
	p.Log(fmt.Sprintf("charged %.2f via PayPal (%s) (tx %s)", amount, p.email, txID))
	return txID, nil
}

func (p *PayPalProcessor) Refund(transactionID string, amount float64) error {
	p.Log(fmt.Sprintf("refunded %.2f for %s", amount, transactionID))
	return nil
}

// --- Crypto -----------------------------------------------------

type CryptoProcessor struct {
	Logger
	walletAddress string
	nextTxID      int
}

func NewCryptoProcessor(wallet string) *CryptoProcessor {
	return &CryptoProcessor{Logger: Logger{Prefix: "CRYPTO"}, walletAddress: wallet, nextTxID: 1}
}

// Process is the ONLY method CryptoProcessor has — deliberately NOT
// Refundable, since crypto transactions in this simplified model can't be
// reversed. This is the point: PaymentProcessor doesn't guarantee refunds,
// so code that needs them MUST check first.
func (cr *CryptoProcessor) Process(amount float64) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}
	txID := fmt.Sprintf("BTC-%d", cr.nextTxID)
	cr.nextTxID++
	cr.Log(fmt.Sprintf("sent %.2f (BTC-equivalent) to %s (tx %s)", amount, cr.walletAddress, txID))
	return txID, nil
}

// --- Gateway -----------------------------------------------------

// Receipt uses `any` (the empty interface) for Metadata, since different
// processors might want to attach completely different extra details
// (a card's last-four, a wallet address, a PayPal email...).
type Receipt struct {
	TransactionID string
	Amount        float64
	Metadata      map[string]any
}

// Gateway knows nothing about credit cards, PayPal, or crypto specifically —
// it only knows PaymentProcessor's one method. This is the payoff of the
// whole design: adding a FOURTH payment method later needs zero changes here.
type Gateway struct {
	processor PaymentProcessor
}

func NewGateway(p PaymentProcessor) *Gateway {
	return &Gateway{processor: p}
}

func (g *Gateway) Charge(amount float64) (*Receipt, error) {
	txID, err := g.processor.Process(amount)
	if err != nil {
		return nil, err
	}
	return &Receipt{TransactionID: txID, Amount: amount, Metadata: map[string]any{
		"processorType": reflect.TypeOf(g.processor).String(),
	}}, nil
}

// TryRefund uses a TYPE ASSERTION to check, at runtime, whether the
// currently-configured processor also satisfies Refundable — Gateway
// itself never needed to know which processors support refunds in advance.
func (g *Gateway) TryRefund(transactionID string, amount float64) error {
	refundable, ok := g.processor.(Refundable)
	if !ok {
		return fmt.Errorf("this payment method does not support refunds")
	}
	return refundable.Refund(transactionID, amount)
}

// printReceipt uses REFLECTION to print every field of the Receipt struct
// generically — the same function would work unmodified even if Receipt
// grew new fields later, unlike hand-writing fmt.Println for each field.
func printReceipt(r *Receipt) {
	val := reflect.ValueOf(*r)
	typ := val.Type()
	fmt.Println("--- Receipt ---")
	for i := 0; i < typ.NumField(); i++ {
		fmt.Printf("  %-15s %v\n", typ.Field(i).Name+":", val.Field(i).Interface())
	}
}

func main() {
	fmt.Println("=== Credit Card (supports refunds) ===")
	cardGateway := NewGateway(NewCreditCardProcessor("4111111111111234"))
	receipt, err := cardGateway.Charge(59.99)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		printReceipt(receipt)
		if err := cardGateway.TryRefund(receipt.TransactionID, 59.99); err != nil {
			fmt.Println("Refund error:", err)
		}
	}

	fmt.Println("\n=== PayPal (supports refunds) ===")
	paypalGateway := NewGateway(NewPayPalProcessor("buyer@example.com"))
	receipt2, _ := paypalGateway.Charge(20.00)
	printReceipt(receipt2)

	fmt.Println("\n=== Crypto (does NOT support refunds) ===")
	cryptoGateway := NewGateway(NewCryptoProcessor("bc1qexampleaddress"))
	receipt3, _ := cryptoGateway.Charge(150.00)
	printReceipt(receipt3)
	if err := cryptoGateway.TryRefund(receipt3.TransactionID, 150.00); err != nil {
		fmt.Println("Refund error:", err) // this branch is expected to run
	}
}
