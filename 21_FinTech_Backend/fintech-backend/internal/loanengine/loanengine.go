// Package loanengine implements Project — Loan Engine: disbursing a loan
// to a borrower's wallet, generating a standard amortization schedule, and
// applying repayments as two separate ledger postings — one recovering
// principal (paying down the platform's loan pool liability), one
// recognizing interest income — so each is visible in the ledger as its
// own distinct, correctly-typed movement rather than one blended number.
package loanengine

import (
	"fmt"
	"math"
	"sync"

	"fintechbackend/internal/ledger"
	"fintechbackend/internal/wallet"
)

const (
	loanPoolAccountID       = "loan-pool"       // Liability: funds the platform has committed to lend out
	interestIncomeAccountID = "interest-income" // Asset: interest collected — see the README's case study
)                                                // on why this is Asset-typed rather than the textbook
                                                  // Revenue type, given this codebase's Asset-wallet convention

type Installment struct {
	Number           int
	PrincipalPortion int64
	InterestPortion  int64
	RemainingBalance int64
	Paid             bool
}

type Status string

const (
	Active Status = "active"
	Closed Status = "closed"
)

type Loan struct {
	ID                string
	BorrowerUserID    string
	Principal         int64
	AnnualRatePercent float64
	TermMonths        int
	Schedule          []*Installment
	Status            Status
}

type Service struct {
	ledger  *ledger.Ledger
	wallets *wallet.Service

	mu     sync.Mutex
	loans  map[string]*Loan
	nextID int
}

func New(l *ledger.Ledger, w *wallet.Service, currency string) *Service {
	l.OpenAccount(loanPoolAccountID, "Loan Disbursement Pool", ledger.Liability, currency)
	l.OpenAccount(interestIncomeAccountID, "Interest Income (collected)", ledger.Asset, currency)
	return &Service{ledger: l, wallets: w, loans: make(map[string]*Loan)}
}

// buildSchedule computes a standard equal-payment (annuity) amortization
// schedule — the same formula real loan systems use, applied here to
// int64 minor units throughout, per the guide's money-representation rule.
func buildSchedule(principal int64, annualRatePercent float64, termMonths int) []*Installment {
	monthlyRate := annualRatePercent / 100 / 12
	p := float64(principal)

	var payment float64
	if monthlyRate == 0 {
		payment = p / float64(termMonths)
	} else {
		payment = p * monthlyRate / (1 - math.Pow(1+monthlyRate, -float64(termMonths)))
	}

	schedule := make([]*Installment, 0, termMonths)
	remaining := p
	for i := 1; i <= termMonths; i++ {
		interest := remaining * monthlyRate
		principalPortion := payment - interest
		if i == termMonths {
			principalPortion = remaining // absorb any rounding drift on the final installment
		}
		remaining -= principalPortion
		if remaining < 0 {
			remaining = 0
		}
		schedule = append(schedule, &Installment{
			Number:           i,
			PrincipalPortion: int64(math.Round(principalPortion)),
			InterestPortion:  int64(math.Round(interest)),
			RemainingBalance: int64(math.Round(remaining)),
		})
	}
	return schedule
}

// Originate disburses the loan immediately (Debit borrower wallet,
// Credit loan pool — the SAME shape as wallet.Deposit's external-funding
// pattern, since a loan disbursement is money entering the borrower's
// wallet from a source external to them, exactly like a deposit) and
// generates the repayment schedule.
func (s *Service) Originate(borrowerUserID string, principal int64, annualRatePercent float64, termMonths int) (*Loan, error) {
	w, err := s.wallets.Get(borrowerUserID)
	if err != nil {
		return nil, err
	}
	if principal <= 0 || termMonths <= 0 {
		return nil, fmt.Errorf("principal and term must be positive")
	}

	s.mu.Lock()
	id := fmt.Sprintf("loan_%d", s.nextID+1)
	s.nextID++
	loan := &Loan{
		ID: id, BorrowerUserID: borrowerUserID, Principal: principal,
		AnnualRatePercent: annualRatePercent, TermMonths: termMonths,
		Schedule: buildSchedule(principal, annualRatePercent, termMonths),
		Status:   Active,
	}
	s.loans[id] = loan
	s.mu.Unlock()

	_, err = s.ledger.Post("loan-engine", "loan-disburse-"+id, fmt.Sprintf("loan disbursement %s", id), []ledger.Entry{
		{AccountID: w.AccountID, Direction: ledger.Debit, Amount: principal},
		{AccountID: loanPoolAccountID, Direction: ledger.Credit, Amount: principal},
	})
	if err != nil {
		return nil, err
	}
	return loan, nil
}

// RecordRepayment applies a payment against the loan's NEXT unpaid
// installment — this demo requires an exact match to that installment's
// total due, a simplification called out in the README alongside what a
// fuller implementation (partial payments, early payoff) would add.
func (s *Service) RecordRepayment(loanID string, amount int64) (*Installment, error) {
	s.mu.Lock()
	loan, ok := s.loans[loanID]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no loan %q", loanID)
	}
	if loan.Status != Active {
		return nil, fmt.Errorf("loan %q is not active (status: %s)", loanID, loan.Status)
	}

	var next *Installment
	for _, inst := range loan.Schedule {
		if !inst.Paid {
			next = inst
			break
		}
	}
	if next == nil {
		return nil, fmt.Errorf("loan %q has no remaining installments", loanID)
	}

	due := next.PrincipalPortion + next.InterestPortion
	if amount != due {
		return nil, fmt.Errorf("payment %d does not match installment %d's amount due (%d)", amount, next.Number, due)
	}

	w, err := s.wallets.Get(loan.BorrowerUserID)
	if err != nil {
		return nil, err
	}
	balance, err := s.ledger.Balance(w.AccountID)
	if err != nil {
		return nil, err
	}
	if balance < amount {
		return nil, fmt.Errorf("insufficient wallet balance: have %d, need %d", balance, amount)
	}

	// TWO separate, individually-balanced postings — principal recovery
	// against the loan pool liability, interest recognized as collected
	// income — rather than one blended entry. See the README for the
	// double-entry reasoning that led to this split.
	ref := fmt.Sprintf("loan-repay-%s-%d", loanID, next.Number)
	if next.PrincipalPortion > 0 {
		_, err = s.ledger.Post("loan-engine", ref+"-principal", fmt.Sprintf("loan %s installment %d principal", loanID, next.Number), []ledger.Entry{
			{AccountID: w.AccountID, Direction: ledger.Credit, Amount: next.PrincipalPortion},
			{AccountID: loanPoolAccountID, Direction: ledger.Debit, Amount: next.PrincipalPortion},
		})
		if err != nil {
			return nil, err
		}
	}
	if next.InterestPortion > 0 {
		_, err = s.ledger.Post("loan-engine", ref+"-interest", fmt.Sprintf("loan %s installment %d interest", loanID, next.Number), []ledger.Entry{
			{AccountID: w.AccountID, Direction: ledger.Credit, Amount: next.InterestPortion},
			{AccountID: interestIncomeAccountID, Direction: ledger.Debit, Amount: next.InterestPortion},
		})
		if err != nil {
			return nil, err
		}
	}

	s.mu.Lock()
	next.Paid = true
	allPaid := true
	for _, inst := range loan.Schedule {
		if !inst.Paid {
			allPaid = false
			break
		}
	}
	if allPaid {
		loan.Status = Closed
	}
	s.mu.Unlock()

	return next, nil
}

func (s *Service) Get(loanID string) (*Loan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	loan, ok := s.loans[loanID]
	if !ok {
		return nil, fmt.Errorf("no loan %q", loanID)
	}
	return loan, nil
}
