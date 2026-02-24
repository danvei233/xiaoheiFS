package plugins

import (
	"errors"
	"fmt"
	"strings"
	"xiaoheiplay/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MapRPCError normalizes plugin gRPC transport errors into business-friendly errors.
func MapRPCError(err error, pluginType string) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	msg := strings.TrimSpace(st.Message())
	switch st.Code() {
	case codes.InvalidArgument:
		if msg == "" {
			return domain.ErrInvalidInput
		}
		return fmt.Errorf("%w: %s", domain.ErrInvalidInput, msg)
	case codes.FailedPrecondition:
		if msg == "" {
			msg = "failed precondition"
		}
		return fmt.Errorf("%s", msg)
	case codes.Unavailable, codes.DeadlineExceeded:
		pt := strings.TrimSpace(pluginType)
		if pt == "" {
			pt = "plugin"
		}
		if msg == "" {
			msg = "service unavailable"
		}
		return fmt.Errorf("%s unavailable: %s", pt, msg)
	default:
		if msg == "" {
			msg = st.Code().String()
		}
		return errors.New(msg)
	}
}

