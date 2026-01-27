package usecase

import (
	"context"

	"xiaoheiplay/internal/domain"
)

type CMSService struct {
	categories CMSCategoryRepository
	posts      CMSPostRepository
	blocks     CMSBlockRepository
	messages   *MessageCenterService
}

func NewCMSService(categories CMSCategoryRepository, posts CMSPostRepository, blocks CMSBlockRepository, messages *MessageCenterService) *CMSService {
	return &CMSService{categories: categories, posts: posts, blocks: blocks, messages: messages}
}

func (s *CMSService) ListCategories(ctx context.Context, lang string, includeHidden bool) ([]domain.CMSCategory, error) {
	return s.categories.ListCMSCategories(ctx, lang, includeHidden)
}

func (s *CMSService) GetCategory(ctx context.Context, id int64) (domain.CMSCategory, error) {
	return s.categories.GetCMSCategory(ctx, id)
}

func (s *CMSService) CreateCategory(ctx context.Context, category *domain.CMSCategory) error {
	return s.categories.CreateCMSCategory(ctx, category)
}

func (s *CMSService) UpdateCategory(ctx context.Context, category domain.CMSCategory) error {
	return s.categories.UpdateCMSCategory(ctx, category)
}

func (s *CMSService) DeleteCategory(ctx context.Context, id int64) error {
	return s.categories.DeleteCMSCategory(ctx, id)
}

func (s *CMSService) ListPosts(ctx context.Context, filter CMSPostFilter) ([]domain.CMSPost, int, error) {
	return s.posts.ListCMSPosts(ctx, filter)
}

func (s *CMSService) GetPost(ctx context.Context, id int64) (domain.CMSPost, error) {
	return s.posts.GetCMSPost(ctx, id)
}

func (s *CMSService) GetPostBySlug(ctx context.Context, slug string) (domain.CMSPost, error) {
	return s.posts.GetCMSPostBySlug(ctx, slug)
}

func (s *CMSService) CreatePost(ctx context.Context, post *domain.CMSPost) error {
	if err := s.posts.CreateCMSPost(ctx, post); err != nil {
		return err
	}
	s.notifyAnnouncement(ctx, *post)
	return nil
}

func (s *CMSService) UpdatePost(ctx context.Context, post domain.CMSPost) error {
	if err := s.posts.UpdateCMSPost(ctx, post); err != nil {
		return err
	}
	s.notifyAnnouncement(ctx, post)
	return nil
}

func (s *CMSService) DeletePost(ctx context.Context, id int64) error {
	return s.posts.DeleteCMSPost(ctx, id)
}

func (s *CMSService) notifyAnnouncement(ctx context.Context, post domain.CMSPost) {
	if s.messages == nil {
		return
	}
	if post.Status != "published" {
		return
	}
	category, err := s.categories.GetCMSCategory(ctx, post.CategoryID)
	if err != nil || category.Key != "announcements" {
		return
	}
	title := post.Title
	if title == "" {
		title = "New Announcement"
	}
	content := post.Summary
	if content == "" {
		content = "A new announcement has been published."
	}
	_ = s.messages.NotifyAllUsers(ctx, "announcement", title, content)
}

func (s *CMSService) ListBlocks(ctx context.Context, page, lang string, includeHidden bool) ([]domain.CMSBlock, error) {
	items, err := s.blocks.ListCMSBlocks(ctx, page, lang, includeHidden)
	if err != nil {
		return nil, err
	}
	items, err = s.ensureDefaultBlocks(ctx, page, lang, items)
	if err != nil {
		return nil, err
	}
	if includeHidden {
		return items, nil
	}
	filtered := make([]domain.CMSBlock, 0, len(items))
	for _, item := range items {
		if item.Visible {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (s *CMSService) GetBlock(ctx context.Context, id int64) (domain.CMSBlock, error) {
	return s.blocks.GetCMSBlock(ctx, id)
}

func (s *CMSService) CreateBlock(ctx context.Context, block *domain.CMSBlock) error {
	return s.blocks.CreateCMSBlock(ctx, block)
}

func (s *CMSService) UpdateBlock(ctx context.Context, block domain.CMSBlock) error {
	return s.blocks.UpdateCMSBlock(ctx, block)
}

func (s *CMSService) DeleteBlock(ctx context.Context, id int64) error {
	return s.blocks.DeleteCMSBlock(ctx, id)
}
