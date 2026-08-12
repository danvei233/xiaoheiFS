package realnameplugin

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	pluginv1 "xiaoheiplay/plugin/v1"
)

func TestMapKycRPCError_MobileRequiredMappedToPhoneRequired(t *testing.T) {
	err := status.Error(codes.InvalidArgument, "params.mobile required for three_factor")
	got := mapKycRPCError(err)
	if !errors.Is(got, domain.ErrPhoneRequired) {
		t.Fatalf("expected ErrPhoneRequired, got %v", got)
	}
}

// fakeKYCClient lets tests drive Start/QueryResult without a plugin Manager.
type fakeKYCClient struct {
	startFn func(ctx context.Context, in *pluginv1.KycStartRequest) (*pluginv1.KycStartResponse, error)
	queryFn func(ctx context.Context, in *pluginv1.KycQueryRequest) (*pluginv1.KycQueryResponse, error)
}

func (f *fakeKYCClient) Start(ctx context.Context, in *pluginv1.KycStartRequest, opts ...grpc.CallOption) (*pluginv1.KycStartResponse, error) {
	if f.startFn != nil {
		return f.startFn(ctx, in)
	}
	return nil, nil
}

func (f *fakeKYCClient) QueryResult(ctx context.Context, in *pluginv1.KycQueryRequest, opts ...grpc.CallOption) (*pluginv1.KycQueryResponse, error) {
	if f.queryFn != nil {
		return f.queryFn(ctx, in)
	}
	return nil, nil
}

// F1: a plugin that answers Ok:true without a token (or with a nil response)
// must never be interpreted as "verified"; only a QueryResult that reports a
// verified status is proof of verification.
func TestVerifyWithClient_EmptyTokenIsNotVerified(t *testing.T) {
	p := &Provider{canQuery: true}
	fake := &fakeKYCClient{
		startFn: func(ctx context.Context, in *pluginv1.KycStartRequest) (*pluginv1.KycStartResponse, error) {
			return &pluginv1.KycStartResponse{Ok: true, Token: ""}, nil
		},
	}
	ok, reason, err := p.verifyWithClient(context.Background(), fake, appshared.RealNameVerifyInput{})
	if err == nil {
		t.Fatalf("expected error for empty token, got ok=%v reason=%q", ok, reason)
	}
	if ok {
		t.Fatalf("empty token must not be treated as verified")
	}
}

func TestVerifyWithClient_NilStartIsNotVerified(t *testing.T) {
	p := &Provider{canQuery: true}
	fake := &fakeKYCClient{
		startFn: func(ctx context.Context, in *pluginv1.KycStartRequest) (*pluginv1.KycStartResponse, error) {
			return nil, nil
		},
	}
	ok, _, err := p.verifyWithClient(context.Background(), fake, appshared.RealNameVerifyInput{})
	if err == nil {
		t.Fatalf("expected error for nil start response, got ok=%v", ok)
	}
	if ok {
		t.Fatalf("nil start must not be treated as verified")
	}
}

func TestVerifyWithClient_RedirectWithTokenReturnsPendingFace(t *testing.T) {
	p := &Provider{canQuery: true}
	fake := &fakeKYCClient{
		startFn: func(ctx context.Context, in *pluginv1.KycStartRequest) (*pluginv1.KycStartResponse, error) {
			return &pluginv1.KycStartResponse{Ok: true, Token: "tok", NextStep: "redirect", Url: "https://example.com/face"}, nil
		},
	}
	ok, reason, err := p.verifyWithClient(context.Background(), fake, appshared.RealNameVerifyInput{})
	if err != nil || ok {
		t.Fatalf("expected pending_face, got ok=%v reason=%q err=%v", ok, reason, err)
	}
	if !strings.HasPrefix(reason, "pending_face:baidu:tok:") {
		t.Fatalf("unexpected reason: %q", reason)
	}
}

// A redirect flow without a token cannot be polled by this host, so it must be
// surfaced as an error rather than an unresolvable pending record.
func TestVerifyWithClient_RedirectWithoutTokenIsNotVerified(t *testing.T) {
	p := &Provider{canQuery: true}
	fake := &fakeKYCClient{
		startFn: func(ctx context.Context, in *pluginv1.KycStartRequest) (*pluginv1.KycStartResponse, error) {
			return &pluginv1.KycStartResponse{Ok: true, Token: "", NextStep: "redirect", Url: "https://example.com/face"}, nil
		},
	}
	ok, _, err := p.verifyWithClient(context.Background(), fake, appshared.RealNameVerifyInput{})
	if err == nil {
		t.Fatalf("expected error for redirect without token, got ok=%v", ok)
	}
	if ok {
		t.Fatalf("redirect without token must not be treated as verified")
	}
}

func TestVerifyWithClient_NilQueryResponseIsError(t *testing.T) {
	p := &Provider{canQuery: true}
	fake := &fakeKYCClient{
		startFn: func(ctx context.Context, in *pluginv1.KycStartRequest) (*pluginv1.KycStartResponse, error) {
			return &pluginv1.KycStartResponse{Ok: true, Token: "tok"}, nil
		},
		queryFn: func(ctx context.Context, in *pluginv1.KycQueryRequest) (*pluginv1.KycQueryResponse, error) {
			return nil, nil
		},
	}
	ok, _, err := p.verifyWithClient(context.Background(), fake, appshared.RealNameVerifyInput{})
	if err == nil {
		t.Fatalf("expected error for nil query response, got ok=%v", ok)
	}
}

func TestVerifyWithClient_VerifiedThroughQuery(t *testing.T) {
	p := &Provider{canQuery: true}
	fake := &fakeKYCClient{
		startFn: func(ctx context.Context, in *pluginv1.KycStartRequest) (*pluginv1.KycStartResponse, error) {
			return &pluginv1.KycStartResponse{Ok: true, Token: "tok", NextStep: "query_result"}, nil
		},
		queryFn: func(ctx context.Context, in *pluginv1.KycQueryRequest) (*pluginv1.KycQueryResponse, error) {
			return &pluginv1.KycQueryResponse{Ok: true, Status: "verified"}, nil
		},
	}
	ok, reason, err := p.verifyWithClient(context.Background(), fake, appshared.RealNameVerifyInput{})
	if err != nil || !ok || reason != "" {
		t.Fatalf("expected verified, got ok=%v reason=%q err=%v", ok, reason, err)
	}
}

// F7: transient plugin state (manager missing / plugin not loaded) must be
// surfaced as an error so the service does not persist a permanent failed
// record for a condition the user can simply retry through.
func TestVerifyWithInput_TransientPluginStateReturnsError(t *testing.T) {
	p := &Provider{mgr: nil}
	ok, reason, err := p.VerifyWithInput(context.Background(), appshared.RealNameVerifyInput{RealName: "A", IDNumber: "B"})
	if err == nil {
		t.Fatalf("expected error for nil manager, got ok=%v reason=%q", ok, reason)
	}
	if ok {
		t.Fatalf("must not be verified")
	}
}

// F3: QueryResult must get its own request budget. When Start consumes the
// whole Start timeout, the subsequent QueryResult must still succeed on a
// fresh context instead of immediately failing on the expired shared one.
func TestVerifyWithClient_QueryResultGetsFreshContext(t *testing.T) {
	p := &Provider{canQuery: true, timeout: 100 * time.Millisecond}
	fake := &fakeKYCClient{
		startFn: func(ctx context.Context, in *pluginv1.KycStartRequest) (*pluginv1.KycStartResponse, error) {
			<-ctx.Done() // consume the entire Start budget
			return &pluginv1.KycStartResponse{Ok: true, Token: "tok", NextStep: "query_result"}, nil
		},
		queryFn: func(ctx context.Context, in *pluginv1.KycQueryRequest) (*pluginv1.KycQueryResponse, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err() // simulate gRPC deadline exceeded
			}
			return &pluginv1.KycQueryResponse{Ok: true, Status: "verified"}, nil
		},
	}
	ok, reason, err := p.verifyWithClient(context.Background(), fake, appshared.RealNameVerifyInput{})
	if err != nil {
		t.Fatalf("expected success despite Start consuming the budget, got ok=%v reason=%q err=%v", ok, reason, err)
	}
	if !ok {
		t.Fatalf("expected verified, got ok=%v reason=%q", ok, reason)
	}
}

// F4: a provider that does not support QueryResult must not leave pending
// records to be polled forever; the poll resolves them to failed immediately.
func TestQueryPending_NoQuerySupportReturnsFailed(t *testing.T) {
	p := &Provider{canQuery: false}
	status, reason, err := p.QueryPending(context.Background(), "tok", "plugin/x/y")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "failed" {
		t.Fatalf("expected failed status, got %q reason=%q", status, reason)
	}
	if reason == "" {
		t.Fatalf("expected a reason")
	}
}
