package conboarding

import "testing"

func TestOnboardingCache_StartsEmpty(t *testing.T) {
	onboardingCache.RLock()
	has := onboardingCache.lastReport != nil
	onboardingCache.RUnlock()
	if has {
		t.Fatal("expected empty cache at startup")
	}
}

func TestPathExists(t *testing.T) {
	if pathExists("/this/path/should/not/exist/anywhere/12345") {
		t.Error("nonexistent path should return false")
	}
	if !pathExists("/tmp") {
		// /tmp 永远存在
		t.Error("/tmp should exist")
	}
}
