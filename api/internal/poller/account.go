package poller

import (
	"context"
	"time"

	"github.com/ZachCharles666/gatelink/api/internal/seller"
	"github.com/rs/zerolog/log"
)

type accountSource interface {
	ListAllAccounts(ctx context.Context) ([]seller.Account, error)
}

type AccountPoller struct {
	source    accountSource
	lastState map[string]string
	interval  time.Duration
	onChange  func(accountID, status string)
}

func NewAccountPoller(source accountSource, onChange func(accountID, status string)) *AccountPoller {
	return &AccountPoller{
		source:    source,
		lastState: make(map[string]string),
		interval:  30 * time.Second,
		onChange:  onChange,
	}
}

func (p *AccountPoller) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	log.Info().Dur("interval", p.interval).Msg("account poller started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("account poller stopped")
			return
		case <-ticker.C:
			if err := p.poll(ctx); err != nil {
				log.Error().Err(err).Msg("account poller error")
			}
		}
	}
}

func (p *AccountPoller) poll(ctx context.Context) error {
	accounts, err := p.source.ListAllAccounts(ctx)
	if err != nil {
		return err
	}

	nextState := make(map[string]string, len(accounts))
	for _, account := range accounts {
		nextState[account.ID] = account.Status

		prevStatus, seen := p.lastState[account.ID]
		if !seen {
			continue
		}
		if prevStatus == account.Status || !isTrackedStatus(account.Status) {
			continue
		}

		log.Info().
			Str("account_id", account.ID).
			Str("status", account.Status).
			Msg("account status changed")

		if p.onChange != nil {
			p.onChange(account.ID, account.Status)
		}
	}

	p.lastState = nextState
	return nil
}

func isTrackedStatus(status string) bool {
	switch status {
	case "suspended", "active", "revoked", "expired":
		return true
	default:
		return false
	}
}
