package payment

import (
	"context"

	"fmt"
	appshared "xiaoheiplay/internal/app/shared"
)

const approvalSchemaJSON = `{"title":"Approval Payment","type":"object","properties":{}}`
const balanceSchemaJSON = `{"title":"Balance Payment","type":"object","properties":{}}`

type simpleProvider struct {
	key        string
	name       string
	schemaJSON string
}

func newApprovalProvider() *simpleProvider {
	return &simpleProvider{key: "approval", name: "Approval", schemaJSON: approvalSchemaJSON}
}

func newBalanceProvider() *simpleProvider {
	return &simpleProvider{key: "balance", name: "Balance", schemaJSON: balanceSchemaJSON}
}

func (p *simpleProvider) Key() string {
	return p.key
}

func (p *simpleProvider) Name() string {
	return p.name
}

func (p *simpleProvider) SchemaJSON() string {
	return p.schemaJSON
}

func (p *simpleProvider) SetConfig(configJSON string) error {
	return nil
}

func (p *simpleProvider) CreatePayment(ctx context.Context, req appshared.PaymentCreateRequest) (appshared.PaymentCreateResult, error) {
	return appshared.PaymentCreateResult{}, fmt.Errorf("provider does not create payments")
}

func (p *simpleProvider) VerifyNotify(ctx context.Context, req appshared.RawHTTPRequest) (appshared.PaymentNotifyResult, error) {
	return appshared.PaymentNotifyResult{}, fmt.Errorf("provider does not handle notify")
}
