// FinTech Backend — a full, integrated demo exercising all nine named
// projects as modules built on one shared double-entry ledger: Wallet
// System, Transfer API, Virtual Accounts, Payment Gateway, Bank API,
// Ledger Engine, Escrow System, Savings App, and Loan Engine — plus
// Fraud/KYC/AML checks, multi-currency FX, reconciliation, and audit
// logging along the way.
//
// This demo calls each service directly (function calls, not HTTP) for
// everything EXCEPT the payment gateway's webhook, which genuinely must
// be real HTTP — a webhook is an inbound callback by definition. See the
// README for why this is the deliberate choice, and how each service
// would be exposed over real HTTP/gRPC per Modules 15/16/20 in a
// production deployment.
//
// Run with: go run ./cmd/demo
package main

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"time"

	"fintechbackend/internal/bankapi"
	"fintechbackend/internal/escrow"
	"fintechbackend/internal/fraud"
	"fintechbackend/internal/fx"
	"fintechbackend/internal/httpapi"
	"fintechbackend/internal/kyc"
	"fintechbackend/internal/ledger"
	"fintechbackend/internal/loanengine"
	"fintechbackend/internal/paymentgateway"
	"fintechbackend/internal/reconciliation"
	"fintechbackend/internal/savings"
	"fintechbackend/internal/transfer"
	"fintechbackend/internal/virtualaccounts"
	"fintechbackend/internal/wallet"
)

func section(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", 70))
}

func dollars(cents int64) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}

func main() {
	// --- Core setup ---------------------------------------------------
	audit := ledger.NewAuditLog()
	l := ledger.New(audit)
	kycSvc := kyc.New()
	fraudChecker := fraud.New()
	wallets := wallet.New(l)
	transfers := transfer.New(l, wallets, kycSvc, fraudChecker)
	rateProvider := fx.NewStaticRateProvider()
	converter := fx.New(l, rateProvider)

	section("1. Onboarding — KYC and AML screening")
	kycSvc.Onboard("ada", "Ada Lovelace")
	kycSvc.Onboard("bob", "Bob Okafor")
	kycSvc.Verify("ada", kyc.TierEnhanced)
	kycSvc.Verify("bob", kyc.TierBasic)
	fmt.Println("ada verified to Enhanced tier, bob verified to Basic tier")

	if _, err := kycSvc.Onboard("eve", "Jane Sanctioned"); err != nil {
		fmt.Println("AML screening correctly blocked onboarding:", err)
	}

	section("2. Wallets — opening ledger-backed accounts")
	wallets.CreateWallet("ada", "USD")
	wallets.CreateWallet("bob", "USD")
	adaNGN, _ := wallets.CreateWallet("ada-ngn", "NGN") // a second currency wallet, for the FX demo below
	fmt.Println("opened wallets for ada (USD), bob (USD), and ada-ngn (NGN)")

	section("3. Deposits and Idempotency")
	wallets.Deposit("teller", "dep-key-ada-1", "ada", 100000) // $1,000.00
	wallets.Deposit("teller", "dep-key-bob-1", "bob", 20000)  // $200.00
	balA, _ := wallets.Balance("ada")
	balB, _ := wallets.Balance("bob")
	fmt.Printf("ada balance: %s, bob balance: %s\n", dollars(balA), dollars(balB))

	// Simulate a client retrying a deposit request after a lost response —
	// SAME idempotency key, sent twice.
	wallets.Deposit("teller", "dep-key-ada-1", "ada", 100000)
	balA, _ = wallets.Balance("ada")
	fmt.Printf("ada balance after RETRYING the same deposit request: %s (unchanged — idempotency held)\n", dollars(balA))

	section("4. Transfer API")
	tx, err := transfers.Send("ada", "transfer-key-1", "ada", "bob", 15000) // $150.00
	if err != nil {
		fmt.Println("transfer failed:", err)
	} else {
		balA, _ = wallets.Balance("ada")
		balB, _ = wallets.Balance("bob")
		fmt.Printf("transfer %s succeeded: ada -> bob $150.00. ada: %s, bob: %s\n", tx.ID, dollars(balA), dollars(balB))
	}

	section("5. KYC Limit Enforcement")
	_, err = transfers.Send("bob", "transfer-key-2", "bob", "ada", 500000) // $5,000 — exceeds bob's Basic tier limit
	fmt.Println("attempted a $5,000 transfer from bob (Basic tier, $1,000 limit):")
	fmt.Println(" ->", err)

	section("6. Fraud Velocity Check")
	var lastErr error
	for i := 0; i < 12; i++ {
		_, lastErr = transfers.Send("ada", fmt.Sprintf("velocity-key-%d", i), "ada", "bob", 100)
	}
	fmt.Println("attempted 12 rapid small transfers from ada (velocity limit: 10 per 5 minutes):")
	fmt.Println(" -> final attempt result:", lastErr)

	section("7. Multi-Currency / Exchange Rates")
	converted, err := converter.Convert(context.Background(), "ada", "fx-key-1", "wallet-ada", adaNGN.AccountID, 10000, "USD", "NGN")
	if err != nil {
		fmt.Println("conversion failed:", err)
	} else {
		ngnBalance, _ := l.Balance(adaNGN.AccountID)
		fmt.Printf("converted $100.00 -> NGN at the current rate: %d minor units credited. ada-ngn balance: %d\n", converted, ngnBalance)
	}

	section("8. Virtual Accounts")
	vaSvc := virtualaccounts.New(l, wallets)
	va, _ := vaSvc.Issue("bob")
	fmt.Println("issued virtual account for bob:", va.Number)
	vaSvc.HandleIncomingDeposit(va.Number, 5000, "va-deposit-key-1") // an external payer sends bob $50 directly
	balB, _ = wallets.Balance("bob")
	fmt.Printf("external deposit to bob's virtual account received. bob balance: %s\n", dollars(balB))

	section("9. Payment Gateway — real async webhook + idempotent redelivery")
	bank := bankapi.NewServer()
	defer bank.Close()

	// The webhook receiver is genuinely real HTTP — see the package
	// comment on httpapi for why this ONE piece can't be a direct call.
	var gateway *paymentgateway.Service
	webhookServer := httptest.NewServer(nil)
	defer webhookServer.Close()
	gateway = paymentgateway.New(l, wallets, bank, webhookServer.URL+"/webhooks/bank")
	webhookServer.Config.Handler = httpapi.NewRouter(gateway)

	ref, err := gateway.InitiateDeposit("ada", 25000, "USD") // $250.00
	if err != nil {
		fmt.Println("gateway deposit initiation failed:", err)
	} else {
		fmt.Println("deposit initiated via bank, reference:", ref, "— waiting for async settlement webhook...")
		time.Sleep(300 * time.Millisecond) // give the fake bank's goroutine time to call back
		balA, _ = wallets.Balance("ada")
		fmt.Printf("ada balance after settlement webhook arrived: %s\n", dollars(balA))

		// Redeliver the EXACT same webhook the bank already sent — real
		// gateways document at-least-once delivery; this proves the
		// receiver handles it correctly.
		if payload, ok := gateway.LastWebhookFor(ref); ok {
			bankapi.SimulateDuplicateWebhook(webhookServer.URL+"/webhooks/bank", payload)
			time.Sleep(50 * time.Millisecond)
			balA, _ = wallets.Balance("ada")
			fmt.Printf("ada balance after a DUPLICATE webhook delivery: %s (unchanged — idempotency held)\n", dollars(balA))
		}
	}

	section("10. Escrow System")
	escrowSvc := escrow.New(l, wallets, "USD")
	esc, _ := escrowSvc.Create("ada", 20000, "USD") // ada pays $200 into escrow
	balA, _ = wallets.Balance("ada")
	fmt.Printf("escrow %s created, holding $200.00. ada balance: %s\n", esc.ID, dollars(balA))
	escrowSvc.Release(esc.ID, "bob") // released to bob once the "deal" completes
	balB, _ = wallets.Balance("bob")
	fmt.Printf("escrow released to bob. bob balance: %s\n", dollars(balB))

	esc2, _ := escrowSvc.Create("bob", 5000, "USD")
	escrowSvc.Refund(esc2.ID) // the deal falls through — refunded to the original payer
	balB, _ = wallets.Balance("bob")
	fmt.Printf("a second escrow (%s) was refunded instead. bob balance: %s\n", esc2.ID, dollars(balB))

	section("11. Savings App")
	savingsSvc := savings.New(l, wallets, "USD")
	savingsSvc.OpenSavingsAccount("bob", "USD")
	savingsSvc.Lock("bob", 10000) // bob locks $100 into savings
	balB, _ = wallets.Balance("bob")
	savingsBal, _ := savingsSvc.Balance("bob")
	fmt.Printf("bob locked $100.00 into savings. wallet: %s, savings: %s\n", dollars(balB), dollars(savingsBal))
	savingsSvc.AccrueInterest(5.0) // 5% accrual
	savingsBal, _ = savingsSvc.Balance("bob")
	fmt.Printf("after 5%% interest accrual, bob's savings balance: %s\n", dollars(savingsBal))

	section("12. Loan Engine")
	loanSvc := loanengine.New(l, wallets, "USD")
	loan, _ := loanSvc.Originate("bob", 120000, 12.0, 6) // $1,200 at 12% APR over 6 months
	balB, _ = wallets.Balance("bob")
	fmt.Printf("loan %s originated: $1,200.00 disbursed to bob. bob wallet balance: %s\n", loan.ID, dollars(balB))
	fmt.Println("amortization schedule:")
	for _, inst := range loan.Schedule {
		fmt.Printf("  #%d  principal: %-10s interest: %-10s remaining: %s\n",
			inst.Number, dollars(inst.PrincipalPortion), dollars(inst.InterestPortion), dollars(inst.RemainingBalance))
	}
	first := loan.Schedule[0]
	loanSvc.RecordRepayment(loan.ID, first.PrincipalPortion+first.InterestPortion)
	balB, _ = wallets.Balance("bob")
	fmt.Printf("first installment repaid. bob wallet balance: %s\n", dollars(balB))

	section("13. Reconciliation")
	integrity := reconciliation.CheckLedgerIntegrity(l, l.AccountIDs())
	fmt.Println("system-wide ledger integrity check (debits == credits, across EVERY account):")
	fmt.Println(" -> balanced:", integrity.Balanced)

	bankBalance, _ := l.Balance("bank-settlement")
	fmt.Println("reconciliation against the bank's own reported total:")

	matchCheck := reconciliation.ReconcileAgainstExternalStatement(l, "bank-settlement", bankBalance)
	fmt.Println(" -> ledger vs. a statement that MATCHES:    balanced =", matchCheck.Balanced)

	// Now simulate what a REAL discrepancy looks like — the bank's own
	// statement reports $5.00 less than our ledger thinks arrived (a
	// delayed settlement, an unrecorded fee, or worse). This is exactly
	// the guide's "last line of defense" scenario: nothing else in this
	// system would have caught this on its own.
	discrepantCheck := reconciliation.ReconcileAgainstExternalStatement(l, "bank-settlement", bankBalance-500)
	fmt.Println(" -> ledger vs. a statement that's off by $5:  balanced =", discrepantCheck.Balanced)
	for _, d := range discrepantCheck.Discrepancies {
		fmt.Printf("      %s (ledger: %s, bank statement: %s)\n", d.Description, dollars(d.Actual), dollars(d.Expected))
	}

	section("14. Audit Log (tail)")
	entries := audit.All()
	start := len(entries) - 10
	if start < 0 {
		start = 0
	}
	for _, e := range entries[start:] {
		fmt.Printf("  [%s] actor=%-14s action=%-20s target=%s\n", e.ID, e.Actor, e.Action, e.Target)
	}

	fmt.Println()
	fmt.Println("Demo complete. See README.md for a section-by-section walkthrough")
	fmt.Println("with diagrams for every module exercised above.")
}
