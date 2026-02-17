package testutil

import (
	"context"
	"sync"
	"time"

	"fmt"
	appshared "xiaoheiplay/internal/app/shared"
)

type FakeAutomationClient struct {
	mu sync.Mutex

	CreateHostRequests []appshared.AutomationCreateHostRequest
	CreateHostResult   appshared.AutomationCreateHostResult
	CreateHostErr      error

	HostInfo    map[int64]appshared.AutomationHostInfo
	HostInfoErr error

	ListHostSimpleItems []appshared.AutomationHostSimple
	ListHostSimpleErr   error

	ElasticUpdates []appshared.AutomationElasticUpdateRequest
	ElasticErr     error

	RenewCalls []struct {
		HostID int64
		Next   time.Time
	}
	RenewErr error

	LockCalls    []int64
	UnlockCalls  []int64
	DeleteCalls  []int64
	StartCalls   []int64
	StopCalls    []int64
	RebootCalls  []int64
	ResetOSCalls []struct {
		HostID     int64
		TemplateID int64
		Password   string
	}
	ResetOSPasswordCalls []struct {
		HostID   int64
		Password string
	}
	SnapshotList []appshared.AutomationSnapshot
	BackupList   []appshared.AutomationBackup
	FirewallList []appshared.AutomationFirewallRule
	PortList     []appshared.AutomationPortMapping
}

type FakeAutomationResolver struct {
	Client appshared.AutomationClient
	Err    error
}

func (r *FakeAutomationResolver) ClientForGoodsType(ctx context.Context, goodsTypeID int64) (appshared.AutomationClient, error) {
	_ = ctx
	_ = goodsTypeID
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Client, nil
}

func (f *FakeAutomationClient) CreateHost(ctx context.Context, req appshared.AutomationCreateHostRequest) (appshared.AutomationCreateHostResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.CreateHostRequests = append(f.CreateHostRequests, req)
	if f.CreateHostErr != nil {
		return appshared.AutomationCreateHostResult{}, f.CreateHostErr
	}
	if f.CreateHostResult.HostID == 0 {
		f.CreateHostResult.HostID = 1001
	}
	return f.CreateHostResult, nil
}

func (f *FakeAutomationClient) GetHostInfo(ctx context.Context, hostID int64) (appshared.AutomationHostInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.HostInfoErr != nil {
		return appshared.AutomationHostInfo{}, f.HostInfoErr
	}
	if f.HostInfo == nil {
		return appshared.AutomationHostInfo{HostID: hostID, State: 2, HostName: "host"}, nil
	}
	if info, ok := f.HostInfo[hostID]; ok {
		return info, nil
	}
	return appshared.AutomationHostInfo{}, fmt.Errorf("host not found")
}

func (f *FakeAutomationClient) ListHostSimple(ctx context.Context, searchTag string) ([]appshared.AutomationHostSimple, error) {
	if f.ListHostSimpleErr != nil {
		return nil, f.ListHostSimpleErr
	}
	return f.ListHostSimpleItems, nil
}

func (f *FakeAutomationClient) ElasticUpdate(ctx context.Context, req appshared.AutomationElasticUpdateRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ElasticUpdates = append(f.ElasticUpdates, req)
	return f.ElasticErr
}

func (f *FakeAutomationClient) RenewHost(ctx context.Context, hostID int64, nextDueDate time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.RenewCalls = append(f.RenewCalls, struct {
		HostID int64
		Next   time.Time
	}{HostID: hostID, Next: nextDueDate})
	return f.RenewErr
}

func (f *FakeAutomationClient) LockHost(ctx context.Context, hostID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.LockCalls = append(f.LockCalls, hostID)
	return nil
}

func (f *FakeAutomationClient) UnlockHost(ctx context.Context, hostID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.UnlockCalls = append(f.UnlockCalls, hostID)
	return nil
}

func (f *FakeAutomationClient) DeleteHost(ctx context.Context, hostID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.DeleteCalls = append(f.DeleteCalls, hostID)
	return nil
}

func (f *FakeAutomationClient) StartHost(ctx context.Context, hostID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.StartCalls = append(f.StartCalls, hostID)
	return nil
}

func (f *FakeAutomationClient) ShutdownHost(ctx context.Context, hostID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.StopCalls = append(f.StopCalls, hostID)
	return nil
}

func (f *FakeAutomationClient) RebootHost(ctx context.Context, hostID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.RebootCalls = append(f.RebootCalls, hostID)
	return nil
}

func (f *FakeAutomationClient) ResetOS(ctx context.Context, hostID int64, templateID int64, password string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ResetOSCalls = append(f.ResetOSCalls, struct {
		HostID     int64
		TemplateID int64
		Password   string
	}{HostID: hostID, TemplateID: templateID, Password: password})
	return nil
}

func (f *FakeAutomationClient) ResetOSPassword(ctx context.Context, hostID int64, password string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ResetOSPasswordCalls = append(f.ResetOSPasswordCalls, struct {
		HostID   int64
		Password string
	}{HostID: hostID, Password: password})
	return nil
}

func (f *FakeAutomationClient) ListSnapshots(ctx context.Context, hostID int64) ([]appshared.AutomationSnapshot, error) {
	return f.SnapshotList, nil
}

func (f *FakeAutomationClient) CreateSnapshot(ctx context.Context, hostID int64) error {
	return nil
}

func (f *FakeAutomationClient) DeleteSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	return nil
}

func (f *FakeAutomationClient) RestoreSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	return nil
}

func (f *FakeAutomationClient) ListBackups(ctx context.Context, hostID int64) ([]appshared.AutomationBackup, error) {
	return f.BackupList, nil
}

func (f *FakeAutomationClient) CreateBackup(ctx context.Context, hostID int64) error {
	return nil
}

func (f *FakeAutomationClient) DeleteBackup(ctx context.Context, hostID int64, backupID int64) error {
	return nil
}

func (f *FakeAutomationClient) RestoreBackup(ctx context.Context, hostID int64, backupID int64) error {
	return nil
}

func (f *FakeAutomationClient) ListFirewallRules(ctx context.Context, hostID int64) ([]appshared.AutomationFirewallRule, error) {
	return f.FirewallList, nil
}

func (f *FakeAutomationClient) AddFirewallRule(ctx context.Context, req appshared.AutomationFirewallRuleCreate) error {
	return nil
}

func (f *FakeAutomationClient) DeleteFirewallRule(ctx context.Context, hostID int64, ruleID int64) error {
	return nil
}

func (f *FakeAutomationClient) ListPortMappings(ctx context.Context, hostID int64) ([]appshared.AutomationPortMapping, error) {
	return f.PortList, nil
}

func (f *FakeAutomationClient) AddPortMapping(ctx context.Context, req appshared.AutomationPortMappingCreate) error {
	return nil
}

func (f *FakeAutomationClient) DeletePortMapping(ctx context.Context, hostID int64, mappingID int64) error {
	return nil
}

func (f *FakeAutomationClient) FindPortCandidates(ctx context.Context, hostID int64, keywords string) ([]int64, error) {
	return []int64{}, nil
}

func (f *FakeAutomationClient) GetPanelURL(ctx context.Context, hostName string, panelPassword string) (string, error) {
	return "https://panel.local/" + hostName, nil
}

func (f *FakeAutomationClient) ListAreas(ctx context.Context) ([]appshared.AutomationArea, error) {
	return []appshared.AutomationArea{}, nil
}

func (f *FakeAutomationClient) ListImages(ctx context.Context, lineID int64) ([]appshared.AutomationImage, error) {
	return []appshared.AutomationImage{}, nil
}

func (f *FakeAutomationClient) ListLines(ctx context.Context) ([]appshared.AutomationLine, error) {
	return []appshared.AutomationLine{}, nil
}

func (f *FakeAutomationClient) ListProducts(ctx context.Context, lineID int64) ([]appshared.AutomationProduct, error) {
	return []appshared.AutomationProduct{}, nil
}

func (f *FakeAutomationClient) GetMonitor(ctx context.Context, hostID int64) (appshared.AutomationMonitor, error) {
	return appshared.AutomationMonitor{CPUPercent: 10, MemoryPercent: 20}, nil
}

func (f *FakeAutomationClient) GetVNCURL(ctx context.Context, hostID int64) (string, error) {
	return "https://vnc.local/" + "host", nil
}

type FakePaymentProvider struct {
	KeyVal     string
	NameVal    string
	Schema     string
	CreateRes  appshared.PaymentCreateResult
	CreateErr  error
	VerifyRes  appshared.PaymentNotifyResult
	VerifyErr  error
	VerifyFunc func(req appshared.RawHTTPRequest) (appshared.PaymentNotifyResult, error)
}

func (f *FakePaymentProvider) Key() string        { return f.KeyVal }
func (f *FakePaymentProvider) Name() string       { return f.NameVal }
func (f *FakePaymentProvider) SchemaJSON() string { return f.Schema }
func (f *FakePaymentProvider) SetConfig(configJSON string) error {
	return nil
}
func (f *FakePaymentProvider) CreatePayment(ctx context.Context, req appshared.PaymentCreateRequest) (appshared.PaymentCreateResult, error) {
	if f.CreateErr != nil {
		return appshared.PaymentCreateResult{}, f.CreateErr
	}
	return f.CreateRes, nil
}
func (f *FakePaymentProvider) VerifyNotify(ctx context.Context, req appshared.RawHTTPRequest) (appshared.PaymentNotifyResult, error) {
	if f.VerifyFunc != nil {
		return f.VerifyFunc(req)
	}
	if f.VerifyErr != nil {
		return appshared.PaymentNotifyResult{}, f.VerifyErr
	}
	return f.VerifyRes, nil
}

type FakePaymentRegistry struct {
	mu      sync.Mutex
	enabled map[string]bool
	config  map[string]string
	prov    map[string]appshared.PaymentProvider
}

func NewFakePaymentRegistry() *FakePaymentRegistry {
	return &FakePaymentRegistry{
		enabled: map[string]bool{},
		config:  map[string]string{},
		prov:    map[string]appshared.PaymentProvider{},
	}
}

func (r *FakePaymentRegistry) ListProviders(ctx context.Context, includeDisabled bool) ([]appshared.PaymentProvider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]appshared.PaymentProvider, 0, len(r.prov))
	for key, provider := range r.prov {
		enabled := r.enabled[key]
		if !enabled && !includeDisabled {
			continue
		}
		out = append(out, provider)
	}
	return out, nil
}

func (r *FakePaymentRegistry) GetProvider(ctx context.Context, key string) (appshared.PaymentProvider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	provider, ok := r.prov[key]
	if !ok {
		return nil, appshared.ErrNotFound
	}
	if !r.enabled[key] {
		return nil, appshared.ErrForbidden
	}
	return provider, nil
}

func (r *FakePaymentRegistry) GetProviderConfig(ctx context.Context, key string) (string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.config[key], r.enabled[key], nil
}

func (r *FakePaymentRegistry) UpdateProviderConfig(ctx context.Context, key string, enabled bool, configJSON string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled[key] = enabled
	if configJSON != "" {
		r.config[key] = configJSON
	}
	return nil
}

func (r *FakePaymentRegistry) RegisterProvider(provider appshared.PaymentProvider, enabled bool, configJSON string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prov[provider.Key()] = provider
	r.enabled[provider.Key()] = enabled
	if configJSON != "" {
		r.config[provider.Key()] = configJSON
	}
}

type FakeEmailSender struct {
	mu    sync.Mutex
	Sends []EmailSend
	Err   error
}

type EmailSend struct {
	To      string
	Subject string
	Body    string
}

func (f *FakeEmailSender) Send(ctx context.Context, to string, subject string, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Sends = append(f.Sends, EmailSend{To: to, Subject: subject, Body: body})
	return f.Err
}

type FakeRobotNotifier struct {
	mu      sync.Mutex
	Payload []appshared.RobotOrderPayload
	Err     error
}

func (f *FakeRobotNotifier) NotifyOrderPending(ctx context.Context, payload appshared.RobotOrderPayload) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Payload = append(f.Payload, payload)
	return f.Err
}

type FakeRealNameProvider struct {
	KeyVal  string
	NameVal string
	OK      bool
	Reason  string
	Err     error
}

func (f *FakeRealNameProvider) Key() string  { return f.KeyVal }
func (f *FakeRealNameProvider) Name() string { return f.NameVal }
func (f *FakeRealNameProvider) Verify(ctx context.Context, realName string, idNumber string) (bool, string, error) {
	return f.OK, f.Reason, f.Err
}

type FakeRealNameRegistry struct {
	mu        sync.Mutex
	providers map[string]appshared.RealNameProvider
}

func NewFakeRealNameRegistry() *FakeRealNameRegistry {
	reg := &FakeRealNameRegistry{providers: map[string]appshared.RealNameProvider{}}
	reg.providers["fake"] = &FakeRealNameProvider{KeyVal: "fake", NameVal: "Fake", OK: true}
	return reg
}

func (r *FakeRealNameRegistry) GetProvider(key string) (appshared.RealNameProvider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.providers[key]; ok {
		return p, nil
	}
	return nil, appshared.ErrNotFound
}

func (r *FakeRealNameRegistry) ListProviders() []appshared.RealNameProvider {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]appshared.RealNameProvider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	return out
}

func (r *FakeRealNameRegistry) Register(provider appshared.RealNameProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Key()] = provider
}

var _ appshared.AutomationClient = (*FakeAutomationClient)(nil)
var _ appshared.PaymentProviderRegistry = (*FakePaymentRegistry)(nil)
var _ appshared.EmailSender = (*FakeEmailSender)(nil)
var _ appshared.RobotNotifier = (*FakeRobotNotifier)(nil)
var _ appshared.RealNameProviderRegistry = (*FakeRealNameRegistry)(nil)
var _ appshared.RealNameProvider = (*FakeRealNameProvider)(nil)
