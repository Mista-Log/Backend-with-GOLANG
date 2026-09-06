// Package kyc implements the guide's KYC and AML sections: verification
// tiers with scaling transaction limits, and a sanctions-list screening
// check every new customer must pass before onboarding.
package kyc

import (
	"fmt"
	"strings"
	"sync"
)

type Tier int

const (
	TierUnverified Tier = iota // can hold a wallet, cannot move real money
	TierBasic                   // name + ID document verified
	TierEnhanced                 // address + income verification
)

// tierLimits caps the amount a single transaction can move, per tier — the
// guide's point that limits should SCALE with verification level, not be
// all-or-nothing.
var tierLimits = map[Tier]int64{
	TierUnverified: 0,
	TierBasic:      100000,    // $1,000.00 in cents
	TierEnhanced:   10000000,  // $100,000.00 in cents
}

// sanctionsWatchlist stands in for a real AML sanctions screening API
// (OFAC and similar) — a hardcoded list here purely so the CHECK itself is
// visible and testable; a real deployment calls an external screening
// service instead, with the SAME interface shape.
var sanctionsWatchlist = map[string]bool{
	"jane sanctioned": true,
	"blocked entity":  true,
}

type Customer struct {
	UserID string
	Name   string
	Tier   Tier
}

type Service struct {
	mu        sync.Mutex
	customers map[string]*Customer
}

func New() *Service {
	return &Service{customers: make(map[string]*Customer)}
}

// Onboard is the AML gate: every new customer is screened against the
// sanctions watchlist BEFORE an account is created — required in most
// regulated jurisdictions, not optional.
func (s *Service) Onboard(userID, fullName string) (*Customer, error) {
	if sanctionsWatchlist[strings.ToLower(fullName)] {
		return nil, fmt.Errorf("onboarding blocked: %q matches a sanctions watchlist entry", fullName)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.customers[userID]; exists {
		return nil, fmt.Errorf("customer %q already onboarded", userID)
	}
	c := &Customer{UserID: userID, Name: fullName, Tier: TierUnverified}
	s.customers[userID] = c
	return c, nil
}

// Verify upgrades a customer's tier — in a real system this follows an
// actual document-verification flow; here it's a direct call, so the
// TIER MECHANICS are the visible part.
func (s *Service) Verify(userID string, tier Tier) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.customers[userID]
	if !ok {
		return fmt.Errorf("no customer %q", userID)
	}
	c.Tier = tier
	return nil
}

// CheckLimit is called before any transfer/withdrawal — the practical
// payoff of tiers: an unverified user's transactions are rejected here,
// long before they'd ever reach the ledger.
func (s *Service) CheckLimit(userID string, amount int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.customers[userID]
	if !ok {
		return fmt.Errorf("no customer %q — KYC required before any transaction", userID)
	}
	limit := tierLimits[c.Tier]
	if amount > limit {
		return fmt.Errorf("transaction of %d exceeds tier %d's limit of %d — verify a higher KYC tier to proceed", amount, c.Tier, limit)
	}
	return nil
}

func (s *Service) Get(userID string) (*Customer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.customers[userID]
	if !ok {
		return nil, fmt.Errorf("no customer %q", userID)
	}
	return c, nil
}
