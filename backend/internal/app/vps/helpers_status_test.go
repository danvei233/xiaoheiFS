package vps

import (
	"testing"

	"xiaoheiplay/internal/domain"
)

func TestMapAutomationStateUsesCanonicalLifecycleStatuses(t *testing.T) {
	tests := []struct {
		state int
		want  domain.VPSStatus
	}{
		{state: 11, want: domain.VPSStatusFailed},
		{state: 12, want: domain.VPSStatusDeleting},
	}
	for _, tt := range tests {
		if got := MapAutomationState(tt.state); got != tt.want {
			t.Fatalf("MapAutomationState(%d) = %q, want %q", tt.state, got, tt.want)
		}
	}
}
