package deployer

import "testing"

func TestNewWizardStateDoesNotInjectOpenObserveDefaults(t *testing.T) {
	state := newWizardState(Options{})

	if state.OpenObserveRootEmail != "" {
		t.Fatalf("OpenObserveRootEmail = %q, want empty", state.OpenObserveRootEmail)
	}
	if state.OpenObserveRootPass != "" {
		t.Fatalf("OpenObserveRootPass = %q, want empty", state.OpenObserveRootPass)
	}
}
