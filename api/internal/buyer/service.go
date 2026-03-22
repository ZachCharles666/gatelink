package buyer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/ZachCharles666/gatelink/api/internal/auth"
)

var errBuyerExists = errors.New("buyer already exists")
var errBuyerNotFound = errors.New("buyer not found")
var errInvalidBuyerCredentials = errors.New("invalid credentials")
var errTopupNotFound = errors.New("topup record not found")
var errTopupProcessed = errors.New("topup record already processed")

var ErrDuplicate = errors.New("duplicate")
var ErrInsufficientBalance = errors.New("insufficient balance")

type Buyer struct {
	ID               string
	Email            string
	Phone            string
	Password         string
	APIKey           string
	BalanceUSD       float64
	TotalConsumedUSD float64
	Tier             string
	Status           string
	CreatedAt        time.Time
}

type TopupRecord struct {
	ID          string     `json:"id"`
	BuyerID     string     `json:"buyer_id"`
	AmountUSD   float64    `json:"amount_usd"`
	TxHash      string     `json:"tx_hash"`
	Network     string     `json:"network"`
	Status      string     `json:"status"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	RejectedAt  *time.Time `json:"rejected_at,omitempty"`
	Notes       string     `json:"notes,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type PendingTopupRecord struct {
	ID        string    `json:"id"`
	BuyerID   string    `json:"buyer_id"`
	Email     string    `json:"email,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	AmountUSD float64   `json:"amount_usd"`
	TxHash    string    `json:"tx_hash"`
	Network   string    `json:"network"`
	CreatedAt time.Time `json:"created_at"`
}

type UsageRecord struct {
	Vendor          string    `json:"vendor"`
	Model           string    `json:"model"`
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	CostUSD         float64   `json:"cost_usd"`
	BuyerChargedUSD float64   `json:"buyer_charged_usd"`
	CreatedAt       time.Time `json:"created_at"`
}

// Week 3 continues to use an in-memory store until the shared DB workflow is
// fully available for Dev-B. The API surface matches the planned DB-backed
// service so it can be swapped later with minimal handler churn.
type Service struct {
	mu           sync.RWMutex
	byID         map[string]*Buyer
	byEmail      map[string]*Buyer
	byPhone      map[string]*Buyer
	byAPIKey     map[string]*Buyer
	topupByBuyer map[string][]TopupRecord
	topupByHash  map[string]TopupRecord
	usageByBuyer map[string][]UsageRecord
}

func NewService() *Service {
	return &Service{
		byID:         make(map[string]*Buyer),
		byEmail:      make(map[string]*Buyer),
		byPhone:      make(map[string]*Buyer),
		byAPIKey:     make(map[string]*Buyer),
		topupByBuyer: make(map[string][]TopupRecord),
		topupByHash:  make(map[string]TopupRecord),
		usageByBuyer: make(map[string][]UsageRecord),
	}
}

func (s *Service) Register(_ context.Context, email, phone, password, apiKey string) (*Buyer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if email != "" {
		if _, exists := s.byEmail[email]; exists {
			return nil, errBuyerExists
		}
	}
	if phone != "" {
		if _, exists := s.byPhone[phone]; exists {
			return nil, errBuyerExists
		}
	}

	buyer := &Buyer{
		ID:               newID(),
		Email:            email,
		Phone:            phone,
		Password:         password,
		APIKey:           apiKey,
		BalanceUSD:       0,
		TotalConsumedUSD: 0,
		Tier:             "standard",
		Status:           "active",
		CreatedAt:        time.Now().UTC(),
	}

	s.byID[buyer.ID] = buyer
	if email != "" {
		s.byEmail[email] = buyer
	}
	if phone != "" {
		s.byPhone[phone] = buyer
	}
	s.byAPIKey[apiKey] = buyer

	return cloneBuyer(buyer), nil
}

func (s *Service) Login(_ context.Context, email, phone, password string) (*Buyer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var buyer *Buyer
	switch {
	case email != "":
		buyer = s.byEmail[email]
	case phone != "":
		buyer = s.byPhone[phone]
	}

	if buyer == nil {
		return nil, errBuyerNotFound
	}
	if buyer.Password != password {
		return nil, errInvalidBuyerCredentials
	}

	return cloneBuyer(buyer), nil
}

func (s *Service) FindByID(_ context.Context, id string) (*Buyer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	buyer, ok := s.byID[id]
	if !ok {
		return nil, errBuyerNotFound
	}

	return cloneBuyer(buyer), nil
}

func (s *Service) FindByAPIKey(_ context.Context, apiKey string) (*auth.BuyerInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	buyer, ok := s.byAPIKey[apiKey]
	if !ok {
		return nil, errBuyerNotFound
	}

	return &auth.BuyerInfo{
		ID:         buyer.ID,
		BalanceUSD: buyer.BalanceUSD,
		Tier:       buyer.Tier,
		Status:     buyer.Status,
	}, nil
}

func (s *Service) GetBalance(ctx context.Context, buyerID string) (*Buyer, error) {
	return s.FindByID(ctx, buyerID)
}

func (s *Service) CreateTopup(_ context.Context, buyerID string, amountUSD float64, txHash, network string) (*TopupRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byID[buyerID]; !ok {
		return nil, errBuyerNotFound
	}
	if _, exists := s.topupByHash[txHash]; exists {
		return nil, ErrDuplicate
	}

	record := TopupRecord{
		ID:        newID(),
		BuyerID:   buyerID,
		AmountUSD: amountUSD,
		TxHash:    txHash,
		Network:   network,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}

	s.topupByHash[txHash] = record
	s.topupByBuyer[buyerID] = append([]TopupRecord{record}, s.topupByBuyer[buyerID]...)

	copy := record
	return &copy, nil
}

func (s *Service) ListTopupRecords(_ context.Context, buyerID string, page, pageSize int) ([]TopupRecord, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.byID[buyerID]; !ok {
		return nil, 0, errBuyerNotFound
	}

	page, pageSize = normalizePage(page, pageSize)
	records := s.topupByBuyer[buyerID]
	total := len(records)
	start := (page - 1) * pageSize
	if start >= total {
		return []TopupRecord{}, total, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	result := make([]TopupRecord, 0, end-start)
	for _, record := range records[start:end] {
		result = append(result, record)
	}

	return result, total, nil
}

func (s *Service) ListPendingTopups(_ context.Context, limit int) ([]PendingTopupRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	records := make([]PendingTopupRecord, 0)
	for buyerID, topups := range s.topupByBuyer {
		buyer := s.byID[buyerID]
		for _, record := range topups {
			if record.Status != "pending" {
				continue
			}
			item := PendingTopupRecord{
				ID:        record.ID,
				BuyerID:   record.BuyerID,
				AmountUSD: record.AmountUSD,
				TxHash:    record.TxHash,
				Network:   record.Network,
				CreatedAt: record.CreatedAt,
			}
			if buyer != nil {
				item.Email = buyer.Email
				item.Phone = buyer.Phone
			}
			records = append(records, item)
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})
	if len(records) > limit {
		records = records[:limit]
	}

	return records, nil
}

func (s *Service) ConfirmTopup(_ context.Context, topupID string) (*TopupRecord, *Buyer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, buyer, err := s.findTopupLocked(topupID)
	if err != nil {
		return nil, nil, err
	}
	if record.Status != "pending" {
		return nil, nil, errTopupProcessed
	}

	now := time.Now().UTC()
	record.Status = "confirmed"
	record.ConfirmedAt = &now
	record.RejectedAt = nil
	record.Notes = ""
	buyer.BalanceUSD += record.AmountUSD
	s.topupByHash[record.TxHash] = *record

	return cloneTopup(record), cloneBuyer(buyer), nil
}

func (s *Service) RejectTopup(_ context.Context, topupID, reason string) (*TopupRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, _, err := s.findTopupLocked(topupID)
	if err != nil {
		return nil, err
	}
	if record.Status != "pending" {
		return nil, errTopupProcessed
	}

	now := time.Now().UTC()
	record.Status = "rejected"
	record.RejectedAt = &now
	record.ConfirmedAt = nil
	record.Notes = reason
	s.topupByHash[record.TxHash] = *record

	return cloneTopup(record), nil
}

func (s *Service) ResetAPIKey(_ context.Context, buyerID, newAPIKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	buyer, ok := s.byID[buyerID]
	if !ok {
		return errBuyerNotFound
	}

	delete(s.byAPIKey, buyer.APIKey)
	buyer.APIKey = newAPIKey
	s.byAPIKey[newAPIKey] = buyer
	return nil
}

func (s *Service) CreditBalance(_ context.Context, buyerID string, amountUSD float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	buyer, ok := s.byID[buyerID]
	if !ok {
		return errBuyerNotFound
	}

	buyer.BalanceUSD += amountUSD
	return nil
}

func IsTopupUnavailable(err error) bool {
	return errors.Is(err, errTopupNotFound) || errors.Is(err, errTopupProcessed)
}

func (s *Service) ApplyDispatchCharge(_ context.Context, buyerID string, chargedUSD float64, usage UsageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	buyer, ok := s.byID[buyerID]
	if !ok {
		return errBuyerNotFound
	}
	if buyer.BalanceUSD < chargedUSD {
		return ErrInsufficientBalance
	}

	buyer.BalanceUSD -= chargedUSD
	buyer.TotalConsumedUSD += chargedUSD

	usage.BuyerChargedUSD = chargedUSD
	if usage.CreatedAt.IsZero() {
		usage.CreatedAt = time.Now().UTC()
	}
	s.usageByBuyer[buyerID] = append([]UsageRecord{usage}, s.usageByBuyer[buyerID]...)

	return nil
}

func (s *Service) GetUsageRecords(_ context.Context, buyerID string, page, pageSize int) ([]UsageRecord, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.byID[buyerID]; !ok {
		return nil, 0, errBuyerNotFound
	}

	page, pageSize = normalizePage(page, pageSize)
	records := s.usageByBuyer[buyerID]
	total := len(records)
	start := (page - 1) * pageSize
	if start >= total {
		return []UsageRecord{}, total, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	result := make([]UsageRecord, 0, end-start)
	for _, record := range records[start:end] {
		result = append(result, record)
	}

	return result, total, nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return page, pageSize
}

func cloneBuyer(buyer *Buyer) *Buyer {
	if buyer == nil {
		return nil
	}

	copy := *buyer
	return &copy
}

func cloneTopup(record *TopupRecord) *TopupRecord {
	if record == nil {
		return nil
	}

	copy := *record
	return &copy
}

func (s *Service) findTopupLocked(topupID string) (*TopupRecord, *Buyer, error) {
	for buyerID := range s.topupByBuyer {
		for i := range s.topupByBuyer[buyerID] {
			record := &s.topupByBuyer[buyerID][i]
			if record.ID != topupID {
				continue
			}

			buyer := s.byID[buyerID]
			if buyer == nil {
				return nil, nil, errBuyerNotFound
			}
			return record, buyer, nil
		}
	}

	return nil, nil, errTopupNotFound
}

func newID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "buyer-fallback-id"
	}
	return hex.EncodeToString(buf)
}
