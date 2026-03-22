package seller

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

var errSellerExists = errors.New("seller already exists")
var errSellerNotFound = errors.New("seller not found")
var errAccountNotFound = errors.New("account not found")
var errForbidden = errors.New("forbidden")
var errInvalidExpireAt = errors.New("invalid expire_at")
var errNoPendingEarnings = errors.New("no pending earnings available")
var errSettlementNotFound = errors.New("settlement not found")
var errSettlementProcessed = errors.New("settlement already processed")

type Seller struct {
	ID             string  `json:"id"`
	Phone          string  `json:"phone"`
	DisplayName    string  `json:"display_name"`
	PendingEarnUSD float64 `json:"pending_earn_usd"`
	TotalEarnedUSD float64 `json:"total_earned_usd"`
}

type Account struct {
	ID                   string  `json:"id"`
	SellerID             string  `json:"seller_id"`
	Vendor               string  `json:"vendor"`
	Status               string  `json:"status"`
	HealthScore          int     `json:"health_score"`
	AuthorizedCreditsUSD float64 `json:"authorized_credits_usd"`
	ConsumedCreditsUSD   float64 `json:"consumed_credits_usd"`
	TotalCreditsUSD      float64 `json:"total_credits_usd"`
	ExpectedRate         float64 `json:"expected_rate"`
	ExpireAt             string  `json:"expire_at"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
}

type Settlement struct {
	ID          string  `json:"id"`
	SellerID    string  `json:"seller_id"`
	AmountUSD   float64 `json:"amount_usd"`
	Status      string  `json:"status"`
	PeriodStart string  `json:"period_start"`
	PeriodEnd   string  `json:"period_end"`
	CreatedAt   string  `json:"created_at"`
	PaidAt      string  `json:"paid_at,omitempty"`
	TxHash      string  `json:"tx_hash,omitempty"`
}

type PendingSettlement struct {
	ID          string  `json:"id"`
	SellerID    string  `json:"seller_id"`
	DisplayName string  `json:"display_name"`
	AmountUSD   float64 `json:"amount_usd"`
	PeriodStart string  `json:"period_start"`
	PeriodEnd   string  `json:"period_end"`
	CreatedAt   string  `json:"created_at"`
}

// Week 1-2 keep seller state in memory so auth and seller APIs can be
// verified before the shared DB workflow is fully available.
type Service struct {
	mu                sync.RWMutex
	byID              map[string]*Seller
	byPhone           map[string]*Seller
	accounts          map[string]*Account
	settlementsByUser map[string][]Settlement
}

func NewService() *Service {
	return &Service{
		byID:              make(map[string]*Seller),
		byPhone:           make(map[string]*Seller),
		accounts:          make(map[string]*Account),
		settlementsByUser: make(map[string][]Settlement),
	}
}

func (s *Service) Register(_ context.Context, phone, displayName string) (*Seller, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byPhone[phone]; exists {
		return nil, errSellerExists
	}

	seller := &Seller{
		ID:          newID(),
		Phone:       phone,
		DisplayName: displayName,
	}

	s.byID[seller.ID] = seller
	s.byPhone[phone] = seller

	return cloneSeller(seller), nil
}

func (s *Service) FindByPhone(_ context.Context, phone string) (*Seller, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seller, ok := s.byPhone[phone]
	if !ok {
		return nil, errSellerNotFound
	}

	return cloneSeller(seller), nil
}

func (s *Service) GetSeller(_ context.Context, sellerID string) (*Seller, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seller, ok := s.byID[sellerID]
	if !ok {
		return nil, errSellerNotFound
	}

	return cloneSeller(seller), nil
}

func (s *Service) ListSellersWithPending(_ context.Context) ([]Seller, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Seller, 0)
	for _, item := range s.byID {
		if item.PendingEarnUSD <= 0 {
			continue
		}
		result = append(result, *cloneSeller(item))
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (s *Service) PreCreateAccount(_ context.Context, sellerID, vendor string, authorizedCreditsUSD, expectedRate, totalCreditsUSD float64, expireAt string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byID[sellerID]; !ok {
		return "", errSellerNotFound
	}

	expireTime, err := time.Parse(time.RFC3339, expireAt)
	if err != nil {
		return "", errInvalidExpireAt
	}
	if totalCreditsUSD <= 0 {
		totalCreditsUSD = authorizedCreditsUSD
	}

	now := time.Now().UTC().Format(time.RFC3339)
	accountID := newID()
	s.accounts[accountID] = &Account{
		ID:                   accountID,
		SellerID:             sellerID,
		Vendor:               vendor,
		Status:               "pending_verify",
		HealthScore:          80,
		AuthorizedCreditsUSD: authorizedCreditsUSD,
		ConsumedCreditsUSD:   0,
		TotalCreditsUSD:      totalCreditsUSD,
		ExpectedRate:         expectedRate,
		ExpireAt:             expireTime.UTC().Format(time.RFC3339),
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	return accountID, nil
}

func (s *Service) DeleteAccount(_ context.Context, accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.accounts[accountID]; !ok {
		return errAccountNotFound
	}

	delete(s.accounts, accountID)
	return nil
}

func (s *Service) VerifyOwnership(_ context.Context, accountID, sellerID string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	account, ok := s.accounts[accountID]
	if !ok {
		return errAccountNotFound
	}
	if account.SellerID != sellerID {
		return errForbidden
	}

	return nil
}

func (s *Service) UpdateAuthorization(_ context.Context, accountID string, authorizedCreditsUSD float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	account, ok := s.accounts[accountID]
	if !ok {
		return errAccountNotFound
	}

	account.AuthorizedCreditsUSD = authorizedCreditsUSD
	account.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return nil
}

func (s *Service) RevokeAuthorization(_ context.Context, accountID string) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	account, ok := s.accounts[accountID]
	if !ok {
		return 0, errAccountNotFound
	}

	revoked := account.AuthorizedCreditsUSD - account.ConsumedCreditsUSD
	if revoked < 0 {
		revoked = 0
	}
	account.AuthorizedCreditsUSD = account.ConsumedCreditsUSD
	account.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	return revoked, nil
}

func (s *Service) GetAccount(_ context.Context, accountID string) (*Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	account, ok := s.accounts[accountID]
	if !ok {
		return nil, errAccountNotFound
	}

	return cloneAccount(account), nil
}

func (s *Service) ListAllAccounts(_ context.Context) ([]Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accounts := make([]Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		accounts = append(accounts, *cloneAccount(account))
	}

	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].UpdatedAt == accounts[j].UpdatedAt {
			return accounts[i].ID < accounts[j].ID
		}
		return accounts[i].UpdatedAt < accounts[j].UpdatedAt
	})

	return accounts, nil
}

func (s *Service) ForceSuspendAccount(_ context.Context, accountID string) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	account, ok := s.accounts[accountID]
	if !ok {
		return nil, errAccountNotFound
	}

	account.Status = "suspended"
	account.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return cloneAccount(account), nil
}

func (s *Service) ApplyDispatchEarning(_ context.Context, accountID string, consumedCreditsUSD float64) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	account, ok := s.accounts[accountID]
	if !ok {
		return 0, errAccountNotFound
	}

	seller, ok := s.byID[account.SellerID]
	if !ok {
		return 0, errSellerNotFound
	}

	expectedRate := account.ExpectedRate
	if expectedRate <= 0 {
		expectedRate = 0.75
	}

	sellerEarnUSD := consumedCreditsUSD * expectedRate
	seller.PendingEarnUSD += sellerEarnUSD
	account.ConsumedCreditsUSD += consumedCreditsUSD
	account.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	return sellerEarnUSD, nil
}

func (s *Service) ListAccountsBySeller(_ context.Context, sellerID string) ([]Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.byID[sellerID]; !ok {
		return nil, errSellerNotFound
	}

	accounts := make([]Account, 0)
	for _, account := range s.accounts {
		if account.SellerID == sellerID {
			accounts = append(accounts, *cloneAccount(account))
		}
	}

	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].CreatedAt > accounts[j].CreatedAt
	})

	return accounts, nil
}

func (s *Service) GetRecentSettlements(_ context.Context, sellerID string, limit int) ([]Settlement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.byID[sellerID]; !ok {
		return nil, errSellerNotFound
	}

	settlements := s.settlementsByUser[sellerID]
	if limit > 0 && len(settlements) > limit {
		settlements = settlements[:limit]
	}

	result := make([]Settlement, 0, len(settlements))
	for _, settlement := range settlements {
		result = append(result, settlement)
	}

	return result, nil
}

func (s *Service) ListSettlements(_ context.Context, sellerID string, page, pageSize int) ([]Settlement, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.byID[sellerID]; !ok {
		return nil, 0, errSellerNotFound
	}

	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	settlements := s.settlementsByUser[sellerID]
	total := len(settlements)
	start := (page - 1) * pageSize
	if start >= total {
		return []Settlement{}, total, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	result := make([]Settlement, 0, end-start)
	for _, settlement := range settlements[start:end] {
		result = append(result, settlement)
	}

	return result, total, nil
}

func (s *Service) CreateSettlement(_ context.Context, sellerID string, amountUSD float64, periodStart, periodEnd string) (*Settlement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	seller, ok := s.byID[sellerID]
	if !ok {
		return nil, errSellerNotFound
	}

	if amountUSD <= 0 {
		amountUSD = seller.PendingEarnUSD
	}
	if amountUSD <= 0 {
		return nil, errNoPendingEarnings
	}
	if amountUSD > seller.PendingEarnUSD {
		amountUSD = seller.PendingEarnUSD
	}
	if amountUSD <= 0 {
		return nil, errNoPendingEarnings
	}
	seller.PendingEarnUSD -= amountUSD

	settlement := Settlement{
		ID:          newID(),
		SellerID:    sellerID,
		AmountUSD:   amountUSD,
		Status:      "pending",
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	s.settlementsByUser[sellerID] = append([]Settlement{settlement}, s.settlementsByUser[sellerID]...)

	copy := settlement
	return &copy, nil
}

func (s *Service) ListPendingSettlements(_ context.Context, limit int) ([]PendingSettlement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	result := make([]PendingSettlement, 0)
	for sellerID, settlements := range s.settlementsByUser {
		seller := s.byID[sellerID]
		for _, settlement := range settlements {
			if settlement.Status != "pending" {
				continue
			}

			item := PendingSettlement{
				ID:          settlement.ID,
				SellerID:    settlement.SellerID,
				AmountUSD:   settlement.AmountUSD,
				PeriodStart: settlement.PeriodStart,
				PeriodEnd:   settlement.PeriodEnd,
				CreatedAt:   settlement.CreatedAt,
			}
			if seller != nil {
				item.DisplayName = seller.DisplayName
			}
			result = append(result, item)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt < result[j].CreatedAt
	})
	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (s *Service) PaySettlement(_ context.Context, settlementID, txHash string) (*Settlement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settlement, seller, err := s.findSettlementLocked(settlementID)
	if err != nil {
		return nil, err
	}
	if settlement.Status != "pending" {
		return nil, errSettlementProcessed
	}

	settlement.Status = "paid"
	settlement.TxHash = txHash
	settlement.PaidAt = time.Now().UTC().Format(time.RFC3339)
	if seller.PendingEarnUSD >= settlement.AmountUSD {
		seller.PendingEarnUSD -= settlement.AmountUSD
	} else {
		seller.PendingEarnUSD = 0
	}
	seller.TotalEarnedUSD += settlement.AmountUSD

	copy := *settlement
	return &copy, nil
}

func IsAccountNotFound(err error) bool {
	return errors.Is(err, errAccountNotFound)
}

func IsSettlementUnavailable(err error) bool {
	return errors.Is(err, errSettlementNotFound) || errors.Is(err, errSettlementProcessed)
}

func cloneSeller(seller *Seller) *Seller {
	if seller == nil {
		return nil
	}

	copy := *seller
	return &copy
}

func cloneAccount(account *Account) *Account {
	if account == nil {
		return nil
	}

	copy := *account
	return &copy
}

func (s *Service) findSettlementLocked(settlementID string) (*Settlement, *Seller, error) {
	for sellerID := range s.settlementsByUser {
		for i := range s.settlementsByUser[sellerID] {
			settlement := &s.settlementsByUser[sellerID][i]
			if settlement.ID != settlementID {
				continue
			}

			seller := s.byID[sellerID]
			if seller == nil {
				return nil, nil, errSellerNotFound
			}
			return settlement, seller, nil
		}
	}

	return nil, nil, errSettlementNotFound
}
