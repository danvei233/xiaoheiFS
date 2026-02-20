package usertier

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type AutoCondition struct {
	Metric   string  `json:"metric"`
	Operator string  `json:"operator"`
	Value    float64 `json:"value"`
}

type Service struct {
	repo    appports.UserTierRepository
	catalog appports.CatalogRepository
	users   appports.UserRepository
	wallets appports.WalletRepository
	audit   appports.AuditRepository

	rebuildMu sync.Map
}

func NewService(repo appports.UserTierRepository, catalog appports.CatalogRepository, users appports.UserRepository, wallets appports.WalletRepository, audit appports.AuditRepository) *Service {
	return &Service{repo: repo, catalog: catalog, users: users, wallets: wallets, audit: audit}
}

func (s *Service) EnsureDefaultGroup(ctx context.Context) (domain.UserTierGroup, error) {
	groups, err := s.repo.ListUserTierGroups(ctx)
	if err != nil {
		return domain.UserTierGroup{}, err
	}
	for _, g := range groups {
		if g.IsDefault {
			return g, nil
		}
	}
	group := domain.UserTierGroup{
		Name:               "默认组",
		Color:              "#1677ff",
		Icon:               "badge",
		Priority:           0,
		AutoApproveEnabled: true,
		IsDefault:          true,
	}
	if err := s.repo.CreateUserTierGroup(ctx, &group); err != nil {
		return domain.UserTierGroup{}, err
	}
	rule := domain.UserTierAutoRule{
		GroupID:        group.ID,
		DurationDays:   -1,
		ConditionsJSON: "[]",
		SortOrder:      0,
	}
	_ = s.repo.CreateUserTierAutoRule(ctx, &rule)
	s.RebuildGroupPriceCacheAsync(group.ID)
	return group, nil
}

func (s *Service) EnsureUserHasGroup(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return appshared.ErrInvalidInput
	}
	def, err := s.EnsureDefaultGroup(ctx)
	if err != nil {
		return err
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	_, err = s.assignDefaultGroupIfMissing(ctx, user, def.ID)
	return err
}

func (s *Service) BackfillUsersWithoutGroup(ctx context.Context, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = 500
	}
	def, err := s.EnsureDefaultGroup(ctx)
	if err != nil {
		return 0, err
	}
	totalUpdated := 0
	offset := 0
	for {
		users, total, err := s.users.ListUsersByRoleStatus(ctx, string(domain.UserRoleUser), "", batchSize, offset)
		if err != nil {
			return totalUpdated, err
		}
		if len(users) == 0 {
			break
		}
		for _, user := range users {
			updated, assignErr := s.assignDefaultGroupIfMissing(ctx, user, def.ID)
			if assignErr != nil {
				return totalUpdated, assignErr
			}
			if updated {
				totalUpdated++
			}
		}
		offset += len(users)
		if offset >= total {
			break
		}
	}
	return totalUpdated, nil
}

func (s *Service) ListGroups(ctx context.Context) ([]domain.UserTierGroup, error) {
	if _, err := s.EnsureDefaultGroup(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListUserTierGroups(ctx)
}

func (s *Service) assignDefaultGroupIfMissing(ctx context.Context, user domain.User, defaultGroupID int64) (bool, error) {
	if user.Role != domain.UserRoleUser {
		return false, nil
	}
	if user.UserTierGroupID != nil && *user.UserTierGroupID > 0 {
		return false, nil
	}
	member := domain.UserTierMembership{
		UserID:    user.ID,
		GroupID:   defaultGroupID,
		Source:    domain.UserTierMembershipSourceAuto,
		ExpiresAt: nil,
	}
	if err := s.repo.UpsertUserTierMembership(ctx, &member); err != nil {
		return false, err
	}
	user.UserTierGroupID = &defaultGroupID
	user.UserTierExpireAt = nil
	if err := s.users.UpdateUser(ctx, user); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) GetGroup(ctx context.Context, id int64) (domain.UserTierGroup, error) {
	return s.repo.GetUserTierGroup(ctx, id)
}

func (s *Service) CreateGroup(ctx context.Context, adminID int64, group *domain.UserTierGroup) error {
	group.Name = strings.TrimSpace(group.Name)
	if group.Name == "" {
		return appshared.ErrInvalidInput
	}
	if group.Icon == "" {
		group.Icon = "badge"
	}
	if group.Color == "" {
		group.Color = "#1677ff"
	}
	if err := s.repo.CreateUserTierGroup(ctx, group); err != nil {
		return err
	}
	s.RebuildGroupPriceCacheAsync(group.ID)
	s.auditLog(ctx, adminID, "user_tier_group.create", "user_tier_group", group.ID)
	return nil
}

func (s *Service) UpdateGroup(ctx context.Context, adminID int64, group domain.UserTierGroup) error {
	old, err := s.repo.GetUserTierGroup(ctx, group.ID)
	if err != nil {
		return err
	}
	group.Name = strings.TrimSpace(group.Name)
	if group.Name == "" {
		return appshared.ErrInvalidInput
	}
	if old.IsDefault {
		group.Priority = old.Priority
		group.AutoApproveEnabled = old.AutoApproveEnabled
		group.IsDefault = true
	}
	if err := s.repo.UpdateUserTierGroup(ctx, group); err != nil {
		return err
	}
	s.RebuildGroupPriceCacheAsync(group.ID)
	s.auditLog(ctx, adminID, "user_tier_group.update", "user_tier_group", group.ID)
	return nil
}

func (s *Service) DeleteGroup(ctx context.Context, adminID int64, id int64) error {
	group, err := s.repo.GetUserTierGroup(ctx, id)
	if err != nil {
		return err
	}
	if group.IsDefault {
		return appshared.ErrInvalidInput
	}
	if err := s.repo.DeleteUserTierGroup(ctx, id); err != nil {
		return err
	}
	_ = s.repo.DeleteUserTierPriceCachesByGroup(ctx, id)
	s.auditLog(ctx, adminID, "user_tier_group.delete", "user_tier_group", id)
	return nil
}

func (s *Service) ListDiscountRules(ctx context.Context, groupID int64) ([]domain.UserTierDiscountRule, error) {
	return s.repo.ListUserTierDiscountRules(ctx, groupID)
}

func (s *Service) CreateDiscountRule(ctx context.Context, adminID int64, rule *domain.UserTierDiscountRule) error {
	if err := s.validateDiscountRule(ctx, *rule, 0); err != nil {
		return err
	}
	if err := s.repo.CreateUserTierDiscountRule(ctx, rule); err != nil {
		return err
	}
	s.RebuildGroupPriceCacheAsync(rule.GroupID)
	s.auditLog(ctx, adminID, "user_tier_rule.create", "user_tier_rule", rule.ID)
	return nil
}

func (s *Service) UpdateDiscountRule(ctx context.Context, adminID int64, rule domain.UserTierDiscountRule) error {
	if err := s.validateDiscountRule(ctx, rule, rule.ID); err != nil {
		return err
	}
	if err := s.repo.UpdateUserTierDiscountRule(ctx, rule); err != nil {
		return err
	}
	s.RebuildGroupPriceCacheAsync(rule.GroupID)
	s.auditLog(ctx, adminID, "user_tier_rule.update", "user_tier_rule", rule.ID)
	return nil
}

func (s *Service) DeleteDiscountRule(ctx context.Context, adminID int64, groupID, id int64) error {
	group, err := s.repo.GetUserTierGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group.IsDefault {
		return appshared.ErrInvalidInput
	}
	if err := s.repo.DeleteUserTierDiscountRule(ctx, id); err != nil {
		return err
	}
	s.RebuildGroupPriceCacheAsync(groupID)
	s.auditLog(ctx, adminID, "user_tier_rule.delete", "user_tier_rule", id)
	return nil
}

func (s *Service) ListAutoRules(ctx context.Context, groupID int64) ([]domain.UserTierAutoRule, error) {
	return s.repo.ListUserTierAutoRules(ctx, groupID)
}

func (s *Service) CreateAutoRule(ctx context.Context, adminID int64, rule *domain.UserTierAutoRule) error {
	group, err := s.repo.GetUserTierGroup(ctx, rule.GroupID)
	if err != nil {
		return err
	}
	if group.IsDefault {
		return appshared.ErrInvalidInput
	}
	if !s.validDuration(rule.DurationDays) {
		return appshared.ErrInvalidInput
	}
	if !validConditionsJSON(rule.ConditionsJSON) {
		return appshared.ErrInvalidInput
	}
	if err := s.repo.CreateUserTierAutoRule(ctx, rule); err != nil {
		return err
	}
	s.auditLog(ctx, adminID, "user_tier_auto_rule.create", "user_tier_auto_rule", rule.ID)
	return nil
}

func (s *Service) UpdateAutoRule(ctx context.Context, adminID int64, rule domain.UserTierAutoRule) error {
	group, err := s.repo.GetUserTierGroup(ctx, rule.GroupID)
	if err != nil {
		return err
	}
	if group.IsDefault {
		return appshared.ErrInvalidInput
	}
	if !s.validDuration(rule.DurationDays) {
		return appshared.ErrInvalidInput
	}
	if !validConditionsJSON(rule.ConditionsJSON) {
		return appshared.ErrInvalidInput
	}
	if err := s.repo.UpdateUserTierAutoRule(ctx, rule); err != nil {
		return err
	}
	s.auditLog(ctx, adminID, "user_tier_auto_rule.update", "user_tier_auto_rule", rule.ID)
	return nil
}

func (s *Service) DeleteAutoRule(ctx context.Context, adminID int64, groupID, id int64) error {
	group, err := s.repo.GetUserTierGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group.IsDefault {
		return appshared.ErrInvalidInput
	}
	if err := s.repo.DeleteUserTierAutoRule(ctx, id); err != nil {
		return err
	}
	s.auditLog(ctx, adminID, "user_tier_auto_rule.delete", "user_tier_auto_rule", id)
	return nil
}

func (s *Service) SetUserGroup(ctx context.Context, adminID, userID, groupID int64, expireAt *time.Time) error {
	if userID <= 0 || groupID <= 0 {
		return appshared.ErrInvalidInput
	}
	group, err := s.repo.GetUserTierGroup(ctx, groupID)
	if err != nil {
		return err
	}
	member := domain.UserTierMembership{
		UserID:    userID,
		GroupID:   group.ID,
		Source:    domain.UserTierMembershipSourceManual,
		ExpiresAt: expireAt,
	}
	if err := s.repo.UpsertUserTierMembership(ctx, &member); err != nil {
		return err
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	user.UserTierGroupID = &group.ID
	user.UserTierExpireAt = expireAt
	if err := s.users.UpdateUser(ctx, user); err != nil {
		return err
	}
	s.auditLog(ctx, adminID, "user_tier.set_user_group", "user", userID)
	return nil
}

func (s *Service) TryAutoApproveForUser(ctx context.Context, userID int64, reason string) error {
	if userID <= 0 {
		return appshared.ErrInvalidInput
	}
	if _, err := s.EnsureDefaultGroup(ctx); err != nil {
		return err
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	currentGroupID := int64(0)
	currentPriority := -1
	currentExpired := true
	currentSource := ""
	member, err := s.repo.GetUserTierMembership(ctx, userID)
	if err == nil {
		currentGroupID = member.GroupID
		currentSource = member.Source
		if member.ExpiresAt == nil || member.ExpiresAt.After(time.Now()) {
			currentExpired = false
		}
		if g, gErr := s.repo.GetUserTierGroup(ctx, member.GroupID); gErr == nil {
			currentPriority = g.Priority
			if currentSource == domain.UserTierMembershipSourceManual && !g.AutoApproveEnabled && !currentExpired {
				return nil
			}
		}
	}
	groups, err := s.repo.ListUserTierGroups(ctx)
	if err != nil {
		return err
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Priority == groups[j].Priority {
			return groups[i].ID < groups[j].ID
		}
		return groups[i].Priority > groups[j].Priority
	})
	for _, g := range groups {
		if !g.AutoApproveEnabled {
			continue
		}
		if !currentExpired && g.Priority <= currentPriority {
			continue
		}
		ok, duration, matchErr := s.matchAutoRules(ctx, user, g.ID)
		if matchErr != nil {
			return matchErr
		}
		if !ok {
			continue
		}
		var exp *time.Time
		if duration >= 0 {
			t := time.Now().Add(time.Duration(duration) * 24 * time.Hour)
			exp = &t
		}
		member = domain.UserTierMembership{
			UserID:    userID,
			GroupID:   g.ID,
			Source:    domain.UserTierMembershipSourceAuto,
			ExpiresAt: exp,
		}
		if err := s.repo.UpsertUserTierMembership(ctx, &member); err != nil {
			return err
		}
		user.UserTierGroupID = &g.ID
		user.UserTierExpireAt = exp
		if err := s.users.UpdateUser(ctx, user); err != nil {
			return err
		}
		if g.ID != currentGroupID {
			s.auditLog(ctx, 0, "user_tier.auto_approve."+reason, "user", userID)
		}
		return nil
	}
	return nil
}

func (s *Service) ReconcileExpired(ctx context.Context, limit int) (int, error) {
	items, err := s.repo.ListExpiredUserTierMemberships(ctx, time.Now(), limit)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, item := range items {
		if err := s.TryAutoApproveForUser(ctx, item.UserID, "expire"); err == nil {
			count++
		}
	}
	return count, nil
}

func (s *Service) RebuildAllPriceCachesAsync() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		groups, err := s.repo.ListUserTierGroups(ctx)
		if err != nil {
			return
		}
		for _, g := range groups {
			s.RebuildGroupPriceCacheAsync(g.ID)
		}
	}()
}

func (s *Service) RebuildGroupPriceCacheAsync(groupID int64) {
	if groupID <= 0 {
		return
	}
	go s.rebuildGroupPriceCache(groupID)
}

func (s *Service) ResolvePackagePricing(ctx context.Context, userID, packageID int64) (domain.UserTierPriceCache, int64, error) {
	if userID <= 0 || packageID <= 0 {
		return domain.UserTierPriceCache{}, 0, appshared.ErrInvalidInput
	}
	_ = s.TryAutoApproveForUser(ctx, userID, "pricing")
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return domain.UserTierPriceCache{}, 0, err
	}
	groupID := int64(0)
	if user.UserTierGroupID != nil {
		groupID = *user.UserTierGroupID
	}
	if groupID <= 0 {
		def, derr := s.EnsureDefaultGroup(ctx)
		if derr != nil {
			return domain.UserTierPriceCache{}, 0, derr
		}
		groupID = def.ID
	}
	cache, err := s.repo.GetUserTierPriceCache(ctx, groupID, packageID)
	if err == nil {
		return cache, groupID, nil
	}
	s.RebuildGroupPriceCacheAsync(groupID)
	pkg, perr := s.catalog.GetPackage(ctx, packageID)
	if perr != nil {
		return domain.UserTierPriceCache{}, 0, perr
	}
	plan, plerr := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID)
	if plerr != nil {
		return domain.UserTierPriceCache{}, 0, plerr
	}
	return domain.UserTierPriceCache{
		GroupID:      groupID,
		PackageID:    packageID,
		MonthlyPrice: pkg.Monthly,
		UnitCore:     plan.UnitCore,
		UnitMem:      plan.UnitMem,
		UnitDisk:     plan.UnitDisk,
		UnitBW:       plan.UnitBW,
		UpdatedAt:    time.Now(),
	}, groupID, nil
}

func (s *Service) rebuildGroupPriceCache(groupID int64) {
	lock := s.getRebuildLock(groupID)
	lock.Lock()
	defer lock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	rules, err := s.repo.ListUserTierDiscountRules(ctx, groupID)
	if err != nil {
		return
	}
	packages, err := s.catalog.ListPackages(ctx)
	if err != nil {
		return
	}
	plans, err := s.catalog.ListPlanGroups(ctx)
	if err != nil {
		return
	}
	planMap := make(map[int64]domain.PlanGroup, len(plans))
	for _, p := range plans {
		planMap[p.ID] = p
	}
	items := make([]domain.UserTierPriceCache, 0, len(packages))
	for _, pkg := range packages {
		plan, ok := planMap[pkg.PlanGroupID]
		if !ok {
			continue
		}
		cache := domain.UserTierPriceCache{
			GroupID:      groupID,
			PackageID:    pkg.ID,
			MonthlyPrice: pkg.Monthly,
			UnitCore:     plan.UnitCore,
			UnitMem:      plan.UnitMem,
			UnitDisk:     plan.UnitDisk,
			UnitBW:       plan.UnitBW,
			UpdatedAt:    time.Now(),
		}
		baseRule := selectBestBaseRule(rules, pkg, plan)
		addonRule := selectBestAddonRule(rules, pkg, plan)
		if baseRule != nil {
			if baseRule.FixedPrice != nil && baseRule.Scope == domain.UserTierScopePackage {
				cache.MonthlyPrice = *baseRule.FixedPrice
			} else {
				cache.MonthlyPrice = applyDiscount(cache.MonthlyPrice, baseRule.DiscountPermille)
			}
		}
		if addonRule != nil {
			cache.UnitCore = applyDiscount(cache.UnitCore, addonRule.AddCorePermille)
			cache.UnitMem = applyDiscount(cache.UnitMem, addonRule.AddMemPermille)
			cache.UnitDisk = applyDiscount(cache.UnitDisk, addonRule.AddDiskPermille)
			cache.UnitBW = applyDiscount(cache.UnitBW, addonRule.AddBWPermille)
		}
		items = append(items, cache)
	}
	_ = s.repo.DeleteUserTierPriceCachesByGroup(ctx, groupID)
	_ = s.repo.UpsertUserTierPriceCaches(ctx, items)
}

func (s *Service) getRebuildLock(groupID int64) *sync.Mutex {
	actual, _ := s.rebuildMu.LoadOrStore(groupID, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

func (s *Service) validDuration(days int) bool {
	return days == -1 || days > 0
}

func (s *Service) validateDiscountRule(ctx context.Context, rule domain.UserTierDiscountRule, selfID int64) error {
	group, err := s.repo.GetUserTierGroup(ctx, rule.GroupID)
	if err != nil {
		return err
	}
	if group.IsDefault {
		return appshared.ErrInvalidInput
	}
	if rule.DiscountPermille < 0 || rule.DiscountPermille > 10000 {
		return appshared.ErrInvalidInput
	}
	if rule.AddCorePermille < 0 || rule.AddCorePermille > 10000 ||
		rule.AddMemPermille < 0 || rule.AddMemPermille > 10000 ||
		rule.AddDiskPermille < 0 || rule.AddDiskPermille > 10000 ||
		rule.AddBWPermille < 0 || rule.AddBWPermille > 10000 {
		return appshared.ErrInvalidInput
	}
	if rule.FixedPrice != nil && *rule.FixedPrice < 0 {
		return appshared.ErrInvalidInput
	}
	rules, err := s.repo.ListUserTierDiscountRules(ctx, rule.GroupID)
	if err != nil {
		return err
	}
	for _, existing := range rules {
		if existing.ID == selfID {
			continue
		}
		if sameRuleScope(existing, rule) {
			return appshared.ErrConflict
		}
	}
	return nil
}

func (s *Service) matchAutoRules(ctx context.Context, user domain.User, groupID int64) (bool, int, error) {
	rules, err := s.repo.ListUserTierAutoRules(ctx, groupID)
	if err != nil {
		return false, 0, err
	}
	if len(rules) == 0 {
		return false, 0, nil
	}
	wallet, _ := s.wallets.GetWallet(ctx, user.ID)
	registerMonths := int(time.Since(user.CreatedAt).Hours() / 24 / 30)
	for _, rule := range rules {
		var conditions []AutoCondition
		raw := strings.TrimSpace(rule.ConditionsJSON)
		if raw != "" {
			if err := json.Unmarshal([]byte(raw), &conditions); err != nil {
				continue
			}
		}
		if len(conditions) == 0 {
			return true, rule.DurationDays, nil
		}
		allMatch := true
		for _, c := range conditions {
			var left float64
			switch strings.TrimSpace(c.Metric) {
			case "register_months":
				left = float64(registerMonths)
			case "wallet_balance":
				left = float64(wallet.Balance) / 100.0
			default:
				allMatch = false
				continue
			}
			switch strings.TrimSpace(c.Operator) {
			case "gt":
				allMatch = left > c.Value
			case "lt":
				allMatch = left < c.Value
			case "eq":
				allMatch = math.Abs(left-c.Value) < 0.0001
			default:
				allMatch = false
			}
			if !allMatch {
				break
			}
		}
		if allMatch {
			return true, rule.DurationDays, nil
		}
	}
	return false, 0, nil
}

func (s *Service) auditLog(ctx context.Context, adminID int64, action, targetType string, targetID int64) {
	if s.audit == nil {
		return
	}
	_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{
		AdminID:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetID:   toString(targetID),
		DetailJSON: "{}",
	})
}

func toString(id int64) string {
	return strconv.FormatInt(id, 10)
}

func validConditionsJSON(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	var conditions []AutoCondition
	return json.Unmarshal([]byte(raw), &conditions) == nil
}

func sameRuleScope(a, b domain.UserTierDiscountRule) bool {
	return a.GroupID == b.GroupID &&
		a.Scope == b.Scope &&
		a.GoodsTypeID == b.GoodsTypeID &&
		a.RegionID == b.RegionID &&
		a.PlanGroupID == b.PlanGroupID &&
		a.PackageID == b.PackageID
}

func applyDiscount(v int64, permille int) int64 {
	if permille <= 0 {
		return v
	}
	if permille >= 10000 {
		return 0
	}
	return int64(math.Round(float64(v) * float64(10000-permille) / 10000.0))
}

func ruleSpecificity(scope domain.UserTierScope) int {
	switch scope {
	case domain.UserTierScopePackage:
		return 60
	case domain.UserTierScopePlanGroup, domain.UserTierScopeAddonConfig:
		return 50
	case domain.UserTierScopeGoodsTypeArea:
		return 40
	case domain.UserTierScopeGoodsType:
		return 30
	case domain.UserTierScopeAllAddons:
		return 20
	case domain.UserTierScopeAll:
		return 10
	default:
		return 0
	}
}

func selectBestBaseRule(rules []domain.UserTierDiscountRule, pkg domain.Package, plan domain.PlanGroup) *domain.UserTierDiscountRule {
	var best *domain.UserTierDiscountRule
	bestScore := -1
	for i := range rules {
		r := &rules[i]
		if !baseRuleMatch(*r, pkg, plan) {
			continue
		}
		score := ruleSpecificity(r.Scope)
		if score > bestScore {
			best = r
			bestScore = score
		}
	}
	return best
}

func selectBestAddonRule(rules []domain.UserTierDiscountRule, pkg domain.Package, plan domain.PlanGroup) *domain.UserTierDiscountRule {
	var best *domain.UserTierDiscountRule
	bestScore := -1
	for i := range rules {
		r := &rules[i]
		if !addonRuleMatch(*r, pkg, plan) {
			continue
		}
		score := ruleSpecificity(r.Scope)
		if score > bestScore {
			best = r
			bestScore = score
		}
	}
	return best
}

func baseRuleMatch(rule domain.UserTierDiscountRule, pkg domain.Package, plan domain.PlanGroup) bool {
	switch rule.Scope {
	case domain.UserTierScopeAll:
		return true
	case domain.UserTierScopeGoodsType:
		return rule.GoodsTypeID > 0 && rule.GoodsTypeID == pkg.GoodsTypeID
	case domain.UserTierScopeGoodsTypeArea:
		return rule.GoodsTypeID == pkg.GoodsTypeID && rule.RegionID == plan.RegionID
	case domain.UserTierScopePlanGroup:
		return rule.PlanGroupID > 0 && rule.PlanGroupID == plan.ID
	case domain.UserTierScopePackage:
		return rule.PackageID > 0 && rule.PackageID == pkg.ID
	default:
		return false
	}
}

func addonRuleMatch(rule domain.UserTierDiscountRule, pkg domain.Package, plan domain.PlanGroup) bool {
	switch rule.Scope {
	case domain.UserTierScopeAllAddons:
		return true
	case domain.UserTierScopeGoodsType:
		return rule.GoodsTypeID > 0 && rule.GoodsTypeID == pkg.GoodsTypeID
	case domain.UserTierScopeGoodsTypeArea:
		return rule.GoodsTypeID == pkg.GoodsTypeID && rule.RegionID == plan.RegionID
	case domain.UserTierScopeAddonConfig, domain.UserTierScopePlanGroup:
		return rule.PlanGroupID > 0 && rule.PlanGroupID == plan.ID
	default:
		return false
	}
}
