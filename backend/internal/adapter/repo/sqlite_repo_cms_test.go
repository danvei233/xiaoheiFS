package repo_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSQLiteRepo_CMSCrud(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	cat := domain.CMSCategory{Key: "news", Name: "News", Lang: "zh-CN", Visible: true}
	if err := repo.CreateCMSCategory(ctx, &cat); err != nil {
		t.Fatalf("create category: %v", err)
	}
	if _, err := repo.GetCMSCategory(ctx, cat.ID); err != nil {
		t.Fatalf("get category: %v", err)
	}
	if _, err := repo.GetCMSCategoryByKey(ctx, "news", "zh-CN"); err != nil {
		t.Fatalf("get category by key: %v", err)
	}
	cat.Name = "News2"
	if err := repo.UpdateCMSCategory(ctx, cat); err != nil {
		t.Fatalf("update category: %v", err)
	}
	if list, err := repo.ListCMSCategories(ctx, "zh-CN", true); err != nil || len(list) == 0 {
		t.Fatalf("list categories: %v", err)
	}

	post := domain.CMSPost{CategoryID: cat.ID, Title: "Post", Slug: "post-1", Summary: "sum", ContentHTML: "body", Lang: "zh-CN", Status: "published"}
	if err := repo.CreateCMSPost(ctx, &post); err != nil {
		t.Fatalf("create post: %v", err)
	}
	if _, err := repo.GetCMSPost(ctx, post.ID); err != nil {
		t.Fatalf("get post: %v", err)
	}
	if _, err := repo.GetCMSPostBySlug(ctx, "post-1"); err != nil {
		t.Fatalf("get post by slug: %v", err)
	}
	post.Title = "Post2"
	if err := repo.UpdateCMSPost(ctx, post); err != nil {
		t.Fatalf("update post: %v", err)
	}
	if list, _, err := repo.ListCMSPosts(ctx, shared.CMSPostFilter{Lang: "zh-CN", Limit: 10}); err != nil || len(list) == 0 {
		t.Fatalf("list posts: %v", err)
	}

	block := domain.CMSBlock{Page: "home", Type: "text", Title: "Hero", Lang: "zh-CN", Visible: true}
	if err := repo.CreateCMSBlock(ctx, &block); err != nil {
		t.Fatalf("create block: %v", err)
	}
	if _, err := repo.GetCMSBlock(ctx, block.ID); err != nil {
		t.Fatalf("get block: %v", err)
	}
	block.Title = "Hero2"
	if err := repo.UpdateCMSBlock(ctx, block); err != nil {
		t.Fatalf("update block: %v", err)
	}
	if list, err := repo.ListCMSBlocks(ctx, "home", "zh-CN", true); err != nil || len(list) == 0 {
		t.Fatalf("list blocks: %v", err)
	}

	if err := repo.DeleteCMSPost(ctx, post.ID); err != nil {
		t.Fatalf("delete post: %v", err)
	}
	if err := repo.DeleteCMSBlock(ctx, block.ID); err != nil {
		t.Fatalf("delete block: %v", err)
	}
	if err := repo.DeleteCMSCategory(ctx, cat.ID); err != nil {
		t.Fatalf("delete category: %v", err)
	}
}
