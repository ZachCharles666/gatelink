package scheduler

import (
	"testing"
)

func TestScoreHardExclude_LowHealth(t *testing.T) {
	acct := &AccountInfo{
		ID: "test", Status: "active",
		Health: 29, BalanceUSD: 100, RPMLimit: 60, RPMUsed: 0,
	}
	excluded, reason := acct.IsHardExcluded()
	if !excluded {
		t.Errorf("expected health<30 to be excluded")
	}
	t.Logf("excluded reason: %s", reason)
}

func TestScoreHardExclude_LowBalance(t *testing.T) {
	acct := &AccountInfo{
		ID: "test", Status: "active",
		Health: 80, BalanceUSD: 0.5, RPMLimit: 60, RPMUsed: 0,
	}
	excluded, reason := acct.IsHardExcluded()
	if !excluded {
		t.Errorf("expected balance<1 to be excluded")
	}
	t.Logf("excluded reason: %s", reason)
}

func TestScoreHardExclude_RPMOver80Pct(t *testing.T) {
	acct := &AccountInfo{
		ID: "test", Status: "active",
		Health: 80, BalanceUSD: 100,
		RPMLimit: 100, RPMUsed: 80, // exactly 80%
	}
	excluded, _ := acct.IsHardExcluded()
	if !excluded {
		t.Errorf("expected RPM>=80%% to be excluded")
	}
}

func TestScoreHardExclude_DailyOverPeak150Pct(t *testing.T) {
	acct := &AccountInfo{
		ID: "test", Status: "active",
		Health: 80, BalanceUSD: 100, RPMLimit: 60, RPMUsed: 0,
		PeakDaily: 10, DailyUsed: 15.1, // 151% > 150%
	}
	excluded, _ := acct.IsHardExcluded()
	if !excluded {
		t.Errorf("expected daily>=peak*150%% to be excluded")
	}
}

func TestScoreHardExclude_Suspended(t *testing.T) {
	acct := &AccountInfo{
		ID: "test", Status: "suspended",
		Health: 80, BalanceUSD: 100,
	}
	excluded, _ := acct.IsHardExcluded()
	if !excluded {
		t.Errorf("expected suspended status to be excluded")
	}
}

func TestScoreHardExclude_Eligible(t *testing.T) {
	acct := &AccountInfo{
		ID: "test", Status: "active",
		Health: 80, BalanceUSD: 100,
		RPMLimit: 60, RPMUsed: 20,
		PeakDaily: 10, DailyUsed: 5,
	}
	excluded, reason := acct.IsHardExcluded()
	if excluded {
		t.Errorf("healthy account should not be excluded, got reason: %s", reason)
	}
}

func TestScore_HigherBalanceHigherScore(t *testing.T) {
	poor := &AccountInfo{
		Status: "active", Health: 80, BalanceUSD: 5, RPMLimit: 60, RPMUsed: 0,
	}
	rich := &AccountInfo{
		Status: "active", Health: 80, BalanceUSD: 500, RPMLimit: 60, RPMUsed: 0,
	}
	scorePoor := Score(poor)
	scoreRich := Score(rich)
	if scoreRich <= scorePoor {
		t.Errorf("higher balance should produce higher score: rich=%.2f poor=%.2f", scoreRich, scorePoor)
	}
}

func TestScore_HigherHealthHigherScore(t *testing.T) {
	sick := &AccountInfo{
		Status: "active", Health: 40, BalanceUSD: 100, RPMLimit: 60, RPMUsed: 0,
	}
	healthy := &AccountInfo{
		Status: "active", Health: 90, BalanceUSD: 100, RPMLimit: 60, RPMUsed: 0,
	}
	if Score(healthy) <= Score(sick) {
		t.Errorf("higher health should produce higher score")
	}
}

func TestScore_FullRPMReducesScore(t *testing.T) {
	idle := &AccountInfo{
		Status: "active", Health: 80, BalanceUSD: 100, RPMLimit: 60, RPMUsed: 0,
	}
	busy := &AccountInfo{
		Status: "active", Health: 80, BalanceUSD: 100, RPMLimit: 60, RPMUsed: 55,
	}
	if Score(idle) <= Score(busy) {
		t.Errorf("idle RPM should produce higher score than busy")
	}
}

func TestScore_InRange(t *testing.T) {
	acct := &AccountInfo{
		Status: "active", Health: 75, BalanceUSD: 200,
		RPMLimit: 100, RPMUsed: 30, PeakDaily: 20, DailyUsed: 10,
	}
	s := Score(acct)
	if s < 0 || s > 100 {
		t.Errorf("score should be in [0, 100], got %.2f", s)
	}
}
