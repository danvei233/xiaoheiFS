package repo

import (
	"context"
	"time"
	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) ListCartItems(ctx context.Context, userID int64) ([]domain.CartItem, error) {

	var rows []cartItemRow
	if err := r.gdb.WithContext(ctx).Where("user_id = ?", userID).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CartItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromCartItemRow(row))
	}
	return out, nil

}

func (r *GormRepo) AddCartItem(ctx context.Context, item *domain.CartItem) error {

	row := toCartItemRow(*item)
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*item = fromCartItemRow(row)
	return nil

}

func (r *GormRepo) UpdateCartItem(ctx context.Context, item domain.CartItem) error {

	return r.gdb.WithContext(ctx).Model(&cartItemRow{}).
		Where("id = ? AND user_id = ?", item.ID, item.UserID).
		Updates(map[string]any{"spec_json": item.SpecJSON, "qty": item.Qty, "amount": item.Amount, "updated_at": time.Now()}).Error

}

func (r *GormRepo) DeleteCartItem(ctx context.Context, id int64, userID int64) error {

	return r.gdb.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&cartItemRow{}).Error

}

func (r *GormRepo) ClearCart(ctx context.Context, userID int64) error {

	return r.gdb.WithContext(ctx).Where("user_id = ?", userID).Delete(&cartItemRow{}).Error

}
