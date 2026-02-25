package realnameplugin

import (
	"errors"
	"testing"
	"xiaoheiplay/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapKycRPCError_MobileRequiredMappedToPhoneRequired(t *testing.T) {
	err := status.Error(codes.InvalidArgument, "params.mobile required for three_factor")
	got := mapKycRPCError(err)
	if !errors.Is(got, domain.ErrPhoneRequired) {
		t.Fatalf("expected ErrPhoneRequired, got %v", got)
	}
}
