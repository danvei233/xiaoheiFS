package repo

import (
	"database/sql"
	"fmt"
	"testing"

	"xiaoheiplay/internal/domain"
)

type walletOrderScanner struct {
	values []any
}

func (s walletOrderScanner) Scan(dest ...any) error {
	if len(dest) != len(s.values) {
		return fmt.Errorf("dest/value mismatch: %d != %d", len(dest), len(s.values))
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *int64:
			if s.values[i] == nil {
				*d = 0
				continue
			}
			v, ok := s.values[i].(int64)
			if !ok {
				return fmt.Errorf("index %d type mismatch", i)
			}
			*d = v
		case *string:
			if s.values[i] == nil {
				*d = ""
				continue
			}
			v, ok := s.values[i].(string)
			if !ok {
				return fmt.Errorf("index %d type mismatch", i)
			}
			*d = v
		case *domain.WalletOrderType:
			if s.values[i] == nil {
				*d = ""
				continue
			}
			v, ok := s.values[i].(domain.WalletOrderType)
			if !ok {
				return fmt.Errorf("index %d type mismatch", i)
			}
			*d = v
		case *domain.WalletOrderStatus:
			if s.values[i] == nil {
				*d = ""
				continue
			}
			v, ok := s.values[i].(domain.WalletOrderStatus)
			if !ok {
				return fmt.Errorf("index %d type mismatch", i)
			}
			*d = v
		case *sql.NullInt64:
			if s.values[i] == nil {
				*d = sql.NullInt64{}
				continue
			}
			v, ok := s.values[i].(int64)
			if !ok {
				return fmt.Errorf("index %d type mismatch", i)
			}
			*d = sql.NullInt64{Int64: v, Valid: true}
		case *sql.NullString:
			if s.values[i] == nil {
				*d = sql.NullString{}
				continue
			}
			v, ok := s.values[i].(string)
			if !ok {
				return fmt.Errorf("index %d type mismatch", i)
			}
			*d = sql.NullString{String: v, Valid: true}
		case *sql.NullTime:
			if s.values[i] == nil {
				*d = sql.NullTime{}
				continue
			}
			return fmt.Errorf("index %d unexpected non-nil time", i)
		default:
			return fmt.Errorf("index %d unsupported dest type", i)
		}
	}
	return nil
}

func TestScanWalletOrder_NullCreatedUpdatedAt(t *testing.T) {
	order, err := scanWalletOrder(walletOrderScanner{values: []any{
		int64(1), int64(2), domain.WalletOrderRecharge, int64(100), "CNY", domain.WalletOrderPendingReview,
		"note", "{}", nil, nil, nil, nil,
	}})
	if err != nil {
		t.Fatalf("scan wallet order: %v", err)
	}
	if order.ID != 1 || order.UserID != 2 {
		t.Fatalf("unexpected order ids: %+v", order)
	}
	if !order.CreatedAt.IsZero() || !order.UpdatedAt.IsZero() {
		t.Fatalf("expected zero timestamps when db value is null")
	}
}

type userScanner struct {
	values []any
}

func (s userScanner) Scan(dest ...any) error {
	if len(dest) != len(s.values) {
		return fmt.Errorf("dest/value mismatch: %d != %d", len(dest), len(s.values))
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *int64:
			if s.values[i] == nil {
				*d = 0
				continue
			}
			v, ok := s.values[i].(int64)
			if !ok {
				return fmt.Errorf("index %d type mismatch", i)
			}
			*d = v
		case *string:
			if s.values[i] == nil {
				*d = ""
				continue
			}
			v, ok := s.values[i].(string)
			if !ok {
				return fmt.Errorf("index %d type mismatch", i)
			}
			*d = v
		case *domain.UserRole:
			if s.values[i] == nil {
				*d = ""
				continue
			}
			v, ok := s.values[i].(domain.UserRole)
			if !ok {
				return fmt.Errorf("index %d type mismatch", i)
			}
			*d = v
		case *domain.UserStatus:
			if s.values[i] == nil {
				*d = ""
				continue
			}
			v, ok := s.values[i].(domain.UserStatus)
			if !ok {
				return fmt.Errorf("index %d type mismatch", i)
			}
			*d = v
		case *sql.NullInt64:
			if s.values[i] == nil {
				*d = sql.NullInt64{}
				continue
			}
			v, ok := s.values[i].(int64)
			if !ok {
				return fmt.Errorf("index %d type mismatch", i)
			}
			*d = sql.NullInt64{Int64: v, Valid: true}
		case *sql.NullString:
			if s.values[i] == nil {
				*d = sql.NullString{}
				continue
			}
			v, ok := s.values[i].(string)
			if !ok {
				return fmt.Errorf("index %d type mismatch", i)
			}
			*d = sql.NullString{String: v, Valid: true}
		case *sql.NullTime:
			if s.values[i] == nil {
				*d = sql.NullTime{}
				continue
			}
			return fmt.Errorf("index %d unexpected non-nil time", i)
		default:
			return fmt.Errorf("index %d unsupported dest type", i)
		}
	}
	return nil
}

func TestScanUser_NullCreatedUpdatedAt(t *testing.T) {
	user, err := scanUser(userScanner{values: []any{
		int64(7), "admin", "admin@local", nil, nil, nil, nil, nil, nil,
		"$2a$10$hash", domain.UserRoleAdmin, domain.UserStatusActive, nil, nil,
	}})
	if err != nil {
		t.Fatalf("scan user: %v", err)
	}
	if user.ID != 7 || user.Username != "admin" {
		t.Fatalf("unexpected user: %+v", user)
	}
	if !user.CreatedAt.IsZero() || !user.UpdatedAt.IsZero() {
		t.Fatalf("expected zero timestamps when db value is null")
	}
}

// scanUser and scanWalletOrder are the only remaining users of the legacy
// hand-written scan helpers (all production queries go through GORM). They
// are kept next to the test that exercises them.
func scanUser(row scanner) (domain.User, error) {
	var u domain.User
	var email sql.NullString
	var qq sql.NullString
	var avatar sql.NullString
	var phone sql.NullString
	var bio sql.NullString
	var intro sql.NullString
	var permissionGroupID sql.NullInt64
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	if err := row.Scan(&u.ID, &u.Username, &email, &qq, &avatar, &phone, &bio, &intro, &permissionGroupID, &u.PasswordHash, &u.Role, &u.Status, &createdAt, &updatedAt); err != nil {
		return domain.User{}, rEnsure(err)
	}
	if email.Valid {
		u.Email = email.String
	}
	if qq.Valid {
		u.QQ = qq.String
	}
	if avatar.Valid {
		u.Avatar = avatar.String
	}
	if phone.Valid {
		u.Phone = phone.String
	}
	if bio.Valid {
		u.Bio = bio.String
	}
	if intro.Valid {
		u.Intro = intro.String
	}
	if permissionGroupID.Valid {
		u.PermissionGroupID = &permissionGroupID.Int64
	}
	if createdAt.Valid {
		u.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		u.UpdatedAt = updatedAt.Time
	}
	return u, nil
}
func scanWalletOrder(row scanner) (domain.WalletOrder, error) {
	var order domain.WalletOrder
	var reviewed sql.NullInt64
	var reason sql.NullString
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	if err := row.Scan(&order.ID, &order.UserID, &order.Type, &order.Amount, &order.Currency, &order.Status, &order.Note, &order.MetaJSON, &reviewed, &reason, &createdAt, &updatedAt); err != nil {
		return domain.WalletOrder{}, rEnsure(err)
	}
	if reviewed.Valid {
		order.ReviewedBy = &reviewed.Int64
	}
	if reason.Valid {
		order.ReviewReason = reason.String
	}
	if createdAt.Valid {
		order.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		order.UpdatedAt = updatedAt.Time
	}
	return order, nil
}
type scanner interface {
	Scan(dest ...any) error
}
