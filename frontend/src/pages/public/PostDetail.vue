<template>
  <div class="post-detail-page">
    <!-- Reading Progress Bar -->
    <div class="reading-progress">
      <div class="progress-fill" :style="{ width: `${readingProgress}%` }"></div>
    </div>

    <div class="page-container">
      <!-- Breadcrumb -->
      <nav class="breadcrumb">
        <router-link to="/" class="breadcrumb-item">
          <HomeOutlined class="breadcrumb-icon" />
          <span>首页</span>
        </router-link>
        <span class="breadcrumb-separator">/</span>
        <router-link :to="`/${postCategory}`" class="breadcrumb-item">
          <span>{{ categoryTitle }}</span>
        </router-link>
        <span class="breadcrumb-separator">/</span>
        <span class="breadcrumb-current">{{ post?.title || '加载中...' }}</span>
      </nav>

      <!-- Main Content -->
      <div class="content-wrapper">
        <!-- Sidebar (TOC) -->
        <aside class="sidebar" v-if="showTOC && headings.length > 0">
          <div class="toc-card">
            <h4 class="toc-title">目录</h4>
            <ul class="toc-list">
              <li v-for="heading in headings" :key="heading.id" :class="['toc-item', `toc-level-${heading.level}`]">
                <a :href="`#${heading.id}`" @click.prevent="scrollToHeading(heading.id)">
                  {{ heading.text }}
                </a>
              </li>
            </ul>
          </div>
        </aside>

        <!-- Article -->
        <a-spin :spinning="loading" size="large">
          <article class="post-article" v-if="post" ref="articleRef">
            <!-- Article Header -->
            <header class="article-header">
              <div class="article-meta">
                <span class="meta-badge">{{ categoryTitle }}</span>
                <span class="meta-date">
                  <CalendarOutlined class="meta-icon" />
                  {{ formatDate(post.published_at) }}
                </span>
                <span class="meta-read-time">
                  <ClockCircleOutlined class="meta-icon" />
                  {{ readTime }}
                </span>
                <span class="meta-views" v-if="post.views">
                  <EyeOutlined class="meta-icon" />
                  {{ post.views }}
                </span>
              </div>

              <h1 class="article-title">{{ post.title }}</h1>

              <p class="article-excerpt" v-if="post.summary">{{ post.summary }}</p>

              <!-- Author Info -->
              <div class="author-info" v-if="post.author">
                <div class="author-avatar">
                  {{ post.author.charAt(0) }}
                </div>
                <div class="author-details">
                  <span class="author-name">{{ post.author }}</span>
                  <span class="author-role">技术作者</span>
                </div>
              </div>
            </header>

            <!-- Cover Image -->
            <div class="article-cover" v-if="post.cover_url">
              <img :src="post.cover_url" :alt="post.title" />
              <div class="cover-overlay"></div>
            </div>

            <!-- Article Content -->
            <div class="article-content" v-html="post.content_html" ref="contentRef"></div>

            <!-- Article Footer -->
            <footer class="article-footer">
              <!-- Tags -->
              <div class="article-tags" v-if="post.tags && post.tags.length > 0">
                <span class="tags-label">标签：</span>
                <router-link
                  v-for="tag in post.tags"
                  :key="tag"
                  :to="`/${postCategory}?tag=${tag}`"
                  class="tag"
                >
                  {{ tag }}
                </router-link>
              </div>

              <!-- Share -->
              <div class="article-share">
                <span class="share-label">分享：</span>
                <button class="share-button" @click="copyLink" title="复制链接">
                  <LinkOutlined />
                </button>
                <button class="share-button" @click="shareOnWeibo" title="分享到微博">
                  <ShareAltOutlined />
                </button>
              </div>
            </footer>
          </article>

          <a-empty v-else-if="!loading" description="文章不存在">
            <router-link to="/docs">
              <a-button type="primary">返回文档中心</a-button>
            </router-link>
          </a-empty>
        </a-spin>
      </div>

      <!-- Navigation -->
      <div class="article-navigation" v-if="post">
        <router-link
          v-if="prevPost"
          :to="`/${postCategory}/${prevPost.slug}`"
          class="nav-card nav-prev"
        >
          <span class="nav-label">上一篇</span>
          <span class="nav-title">{{ prevPost.title }}</span>
          <span class="nav-arrow">←</span>
        </router-link>
        <div v-else class="nav-placeholder"></div>

        <router-link to="/docs" class="nav-card nav-back">
          <BookOutlined class="nav-icon" />
          <span>文档中心</span>
        </router-link>

        <router-link
          v-if="nextPost"
          :to="`/${postCategory}/${nextPost.slug}`"
          class="nav-card nav-next"
        >
          <span class="nav-label">下一篇</span>
          <span class="nav-title">{{ nextPost.title }}</span>
          <span class="nav-arrow">→</span>
        </router-link>
        <div v-else class="nav-placeholder"></div>
      </div>

      <!-- Related Articles -->
      <section class="related-section" v-if="relatedPosts.length > 0">
        <h3 class="related-title">相关文章</h3>
        <div class="related-grid">
          <router-link
            v-for="related in relatedPosts"
            :key="related.id"
            :to="`/${postCategory}/${related.slug}`"
            class="related-card"
          >
            <div class="related-cover" v-if="related.cover_url">
              <img :src="related.cover_url" :alt="related.title" />
            </div>
            <div class="related-icon" v-else>
              <FileTextOutlined />
            </div>
            <div class="related-content">
              <h4 class="related-title-text">{{ related.title }}</h4>
              <p class="related-summary">{{ related.summary || '点击查看详情' }}</p>
              <span class="related-date">{{ formatDate(related.published_at) }}</span>
            </div>
          </router-link>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  HomeOutlined,
  BookOutlined,
  FileTextOutlined,
  LinkOutlined,
  PhoneOutlined,
  CalendarOutlined,
  ClockCircleOutlined,
  EyeOutlined,
  ShareAltOutlined
} from '@ant-design/icons-vue'

const route = useRoute()
const router = useRouter()

const post = ref<any>(null)
const loading = ref(false)
const articleRef = ref<HTMLElement>()
const contentRef = ref<HTMLElement>()
const readingProgress = ref(0)
const showTOC = ref(false)
const headings = ref<Array<{ id: string; text: string; level: number }>>([])

// Mock related posts (in real app, fetch from API)
const prevPost = ref<any>(null)
const nextPost = ref<any>(null)
const relatedPosts = ref<any[]>([])

const postCategory = computed(() => route.params.category as string || 'docs')

const categoryTitles: Record<string, string> = {
  docs: '文档中心',
  announcements: '系统公告',
  activities: '活动动态',
  tutorials: '教程学院'
}

const categoryTitle = computed(() => categoryTitles[postCategory.value] || '文章')

const readTime = computed(() => {
  if (!post.value?.content_html) return '1 分钟'
  const words = post.value.content_html.length / 5
  const minutes = Math.ceil(words / 200)
  return minutes < 1 ? '1 分钟' : `${minutes} 分钟`
})

const formatDate = (date: string) => {
  if (!date) return ''
  const d = new Date(date)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (days === 0) return '今天'
  if (days === 1) return '昨天'
  if (days < 7) return `${days} 天前`
  if (days < 30) return `${Math.floor(days / 7)} 周前`
  if (days < 365) return `${Math.floor(days / 30)} 个月前`
  return `${Math.floor(days / 365)} 年前`
}

const extractHeadings = () => {
  if (!contentRef.value) return

  const elements = contentRef.value.querySelectorAll('h1, h2, h3')
  const extracted: Array<{ id: string; text: string; level: number }> = []

  elements.forEach((el, index) => {
    const id = `heading-${index}`
    el.id = id
    extracted.push({
      id,
      text: el.textContent || '',
      level: parseInt(el.tagName.charAt(1))
    })
  })

  headings.value = extracted
  showTOC.value = extracted.length > 2
}

const scrollToHeading = (id: string) => {
  const element = document.getElementById(id)
  if (element) {
    element.scrollIntoView({ behavior: 'smooth' })
  }
}

const updateReadingProgress = () => {
  if (!articleRef.value) return

  const scrollTop = window.scrollY
  const docHeight = articleRef.value.offsetTop + articleRef.value.offsetHeight
  const winHeight = window.innerHeight

  const progress = Math.min((scrollTop / (docHeight - winHeight)) * 100, 100)
  readingProgress.value = progress
}

const copyLink = () => {
  const url = window.location.href
  navigator.clipboard.writeText(url).then(() => {
    message.success('链接已复制到剪贴板')
  })
}

const shareOnWeibo = () => {
  const url = encodeURIComponent(window.location.href)
  const title = encodeURIComponent(post.value?.title || '')
  window.open(`https://service.weibo.com/share/share.php?url=${url}&title=${title}`, '_blank')
}

const fetchRelatedPosts = async () => {
  if (!post.value) return

  try {
    // Fetch posts from same category
    const response = await fetch(`/api/v1/cms/posts?category=${postCategory.value}&limit=6`)
    if (response.ok) {
      const data = await response.json()
      const posts = data.items || data || []

      // Find current post index
      const currentIndex = posts.findIndex((p: any) => p.id === post.value.id)

      // Set prev/next
      if (currentIndex > 0) {
        prevPost.value = posts[currentIndex - 1]
      }
      if (currentIndex < posts.length - 1) {
        nextPost.value = posts[currentIndex + 1]
      }

      // Set related (exclude current and prev/next)
      relatedPosts.value = posts
        .filter((p: any) => p.id !== post.value.id)
        .slice(0, 4)
    }
  } catch (error) {
    console.error('Failed to fetch related posts:', error)
  }
}

onMounted(async () => {
  const slug = route.params.slug as string
  if (!slug) return

  loading.value = true
  try {
    const response = await fetch(`/api/v1/cms/posts/${slug}`)
    if (response.ok) {
      post.value = await response.json()

      // Wait for content to render, then extract headings
      setTimeout(() => {
        extractHeadings()
      }, 100)

      // Fetch related posts
      fetchRelatedPosts()
    }
  } catch (error) {
    console.error('Failed to fetch post:', error)
  } finally {
    loading.value = false
  }

  window.addEventListener('scroll', updateReadingProgress)
  window.addEventListener('resize', () => {
    if (window.innerWidth > 1024) {
      showTOC.value = headings.value.length > 2
    } else {
      showTOC.value = false
    }
  })
})

onUnmounted(() => {
  window.removeEventListener('scroll', updateReadingProgress)
})
</script>

<style scoped>
/* ===== 全宽文档布局 - Modern Full-Width Documentation =====
 * 设计理念：文章内容占据大部分屏幕空间，宽敞舒适
 * 布局：侧边栏固定宽度，文章内容自适应剩余空间
 */

@import url('https://fonts.googleapis.com/css2?family=Crimson+Pro:ital,wght@0,400;0,500;0,600;1,400&family=Space+Grotesk:wght@400;500;600;700&display=swap');

.post-detail-page {
  min-height: 100vh;
  background: var(--color-bg);
  padding-top: 60px;
  position: relative;
  overflow-x: hidden;
}

/* ===== 背景效果 ===== */
.post-detail-page::before {
  content: '';
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noise)'/%3E%3C/svg%3E");
  opacity: 0.02;
  pointer-events: none;
  z-index: 1;
}

.post-detail-page::after {
  content: '';
  position: fixed;
  top: -20%;
  right: -10%;
  width: 700px;
  height: 700px;
  background: radial-gradient(circle, rgba(14, 165, 233, 0.08) 0%, transparent 70%);
  border-radius: 50%;
  filter: blur(100px);
  animation: floatOrb 20s ease-in-out infinite;
  pointer-events: none;
  z-index: 0;
}

@keyframes floatOrb {
  0%, 100% { transform: translate(0, 0) scale(1); }
  33% { transform: translate(30px, -30px) scale(1.05); }
  66% { transform: translate(-20px, 20px) scale(0.95); }
}

/* ===== 阅读进度条 ===== */
.reading-progress {
  position: fixed;
  top: 60px;
  left: 0;
  right: 0;
  height: 3px;
  background: rgba(30, 41, 59, 0.5);
  z-index: 1000;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #38bdf8, #818cf8, #c084fc);
  transition: width 0.1s ease;
  box-shadow: 0 0 15px rgba(56, 189, 248, 0.5);
}

/* ===== 页面容器 - 全宽布局 ===== */
.page-container {
  width: 100%;
  max-width: 100%;
  margin: 0;
  padding: 32px 0 80px;
  position: relative;
  z-index: 2;
}

/* ===== 面包屑 ===== */
.breadcrumb {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin: 0 48px 24px;
  font-size: 13px;
  font-family: 'Space Grotesk', sans-serif;
  font-weight: 500;
}

.breadcrumb-item {
  display: flex;
  align-items: center;
  gap: 6px;
  color: rgba(148, 163, 184, 0.8);
  text-decoration: none;
  transition: color 0.2s;
  padding: 4px 8px;
  border-radius: 6px;
}

.breadcrumb-item:hover {
  color: #38bdf8;
  background: rgba(56, 189, 248, 0.06);
}

.breadcrumb-icon {
  font-size: 14px;
}

.breadcrumb-separator {
  color: rgba(148, 163, 184, 0.4);
  font-size: 12px;
}

.breadcrumb-current {
  color: #f1f5f9;
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
}

/* ===== 内容包装器 - 宽敞布局 ===== */
.content-wrapper {
  display: grid;
  grid-template-columns: 1fr;
  gap: 32px;
  position: relative;
}

/* ===== 侧边栏（目录） ===== */
.sidebar {
  position: relative;
}

.toc-card {
  position: sticky;
  top: 80px;
  background: rgba(17, 24, 39, 0.8);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(56, 189, 248, 0.1);
  border-radius: 16px;
  padding: 20px;
  margin: 0 24px;
  max-height: calc(100vh - 120px);
  overflow-y: auto;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
}

.toc-card::-webkit-scrollbar {
  width: 4px;
}

.toc-card::-webkit-scrollbar-thumb {
  background: rgba(56, 189, 248, 0.3);
  border-radius: 2px;
}

.toc-title {
  font-family: 'Space Grotesk', sans-serif;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: #38bdf8;
  margin: 0 0 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(56, 189, 248, 0.15);
}

.toc-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.toc-item {
  margin-bottom: 2px;
}

.toc-item a {
  display: block;
  padding: 8px 12px;
  color: rgba(148, 163, 184, 0.8);
  text-decoration: none;
  border-radius: 8px;
  transition: all 0.2s;
  font-size: 13px;
  line-height: 1.5;
  font-family: 'Space Grotesk', sans-serif;
  font-weight: 500;
}

.toc-item a:hover {
  background: rgba(56, 189, 248, 0.1);
  color: #38bdf8;
}

.toc-level-3 {
  font-size: 12px;
}

.toc-level-3 a {
  padding-left: 24px;
}

/* ===== 文章卡片 - 宽敞布局 ===== */
.post-article {
  background: transparent;
  border: none;
  border-radius: 0;
  overflow: visible;
  box-shadow: none;
  margin: 0 48px;
}

/* ===== 文章头部 ===== */
.article-header {
  padding: 32px 0 24px;
  position: relative;
}

.article-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}

.meta-badge {
  padding: 6px 14px;
  background: linear-gradient(135deg, #38bdf8, #0ea5e9);
  border-radius: 20px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: white;
  box-shadow: 0 2px 8px rgba(56, 189, 248, 0.3);
}

.meta-date,
.meta-read-time,
.meta-views {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: rgba(148, 163, 184, 0.8);
  font-family: 'Space Grotesk', sans-serif;
}

.meta-icon {
  font-size: 14px;
  color: #38bdf8;
}

.article-title {
  font-family: 'Space Grotesk', sans-serif;
  font-size: clamp(32px, 4vw, 44px);
  font-weight: 700;
  line-height: 1.2;
  margin: 0 0 20px;
  color: #f1f5f9;
  letter-spacing: -0.02em;
}

.article-excerpt {
  font-family: 'Crimson Pro', serif;
  font-size: 18px;
  line-height: 1.7;
  color: rgba(148, 163, 184, 0.8);
  margin: 0 0 28px;
  font-weight: 400;
  font-style: italic;
}

/* 作者信息 */
.author-info {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 20px;
  background: rgba(56, 189, 248, 0.05);
  border: 1px solid rgba(56, 189, 248, 0.1);
  border-radius: 12px;
  max-width: fit-content;
}

.author-avatar {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #38bdf8, #818cf8);
  border-radius: 12px;
  font-size: 20px;
  font-weight: 700;
  color: white;
  box-shadow: 0 2px 8px rgba(56, 189, 248, 0.3);
}

.author-details {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.author-name {
  font-size: 15px;
  font-weight: 600;
  color: #f1f5f9;
  font-family: 'Space Grotesk', sans-serif;
}

.author-role {
  font-size: 11px;
  color: rgba(148, 163, 184, 0.7);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* ===== 封面图 ===== */
.article-cover {
  position: relative;
  width: 100%;
  aspect-ratio: 21 / 9;
  overflow: hidden;
  margin: 24px 0;
  border-radius: 16px;
}

.article-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.cover-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 100px;
  background: linear-gradient(to top, rgba(10, 14, 23, 1), transparent);
  pointer-events: none;
}

/* ===== 文章内容 - 宽敞舒适 ===== */
.article-content {
  padding: 0;
  margin-top: 32px;
  font-size: 17px;
  line-height: 1.9;
  color: rgba(241, 245, 249, 0.9);
  font-family: 'Crimson Pro', serif;
  max-width: none;
}

/* ===== 内容样式 ===== */
.article-content :deep(h1),
.article-content :deep(h2),
.article-content :deep(h3) {
  font-family: 'Space Grotesk', sans-serif;
  font-weight: 700;
  margin-top: 48px;
  margin-bottom: 20px;
  scroll-margin-top: 80px;
  letter-spacing: -0.01em;
}

.article-content :deep(h1) {
  font-size: 30px;
  padding-bottom: 12px;
  border-bottom: 2px solid rgba(56, 189, 248, 0.2);
  color: #f1f5f9;
}

.article-content :deep(h2) {
  font-size: 24px;
  padding-bottom: 10px;
  border-bottom: 1px solid rgba(56, 189, 248, 0.15);
  color: #f1f5f9;
}

.article-content :deep(h3) {
  font-size: 20px;
  color: #e2e8f0;
}

.article-content :deep(p) {
  margin-bottom: 18px;
  max-width: 900px;
}

.article-content :deep(a) {
  color: #38bdf8;
  text-decoration: none;
  border-bottom: 1px solid rgba(56, 189, 248, 0.3);
  transition: all 0.2s ease;
  position: relative;
}

.article-content :deep(a:hover) {
  border-bottom-color: #38bdf8;
  text-shadow: 0 0 20px rgba(56, 189, 248, 0.4);
}

.article-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 16px;
  margin: 32px auto;
  display: block;
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.3),
    0 0 0 1px rgba(255, 255, 255, 0.05) inset;
}

/* ===== CODE BLOCKS ===== */
.article-content :deep(pre) {
  background: linear-gradient(135deg,
    rgba(15, 23, 42, 0.95) 0%,
    rgba(30, 41, 59, 0.95) 100%
  );
  border-radius: 16px;
  padding: 24px;
  overflow-x: auto;
  margin: 28px 0;
  border: 1px solid rgba(56, 189, 248, 0.15);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.05),
    0 4px 20px rgba(0, 0, 0, 0.3);
  position: relative;
}

/* Subtle glow for code blocks */
.article-content :deep(pre)::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(90deg,
    transparent,
    rgba(56, 189, 248, 0.3),
    transparent
  );
}

.article-content :deep(code) {
  font-family: 'Fira Code', 'SFMono-Regular', Consolas, monospace;
  font-size: 14px;
  color: #e2e8f0;
}

.article-content :deep(p code),
.article-content :deep(li code) {
  background: rgba(56, 189, 248, 0.12);
  color: #38bdf8;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 0.9em;
  border: 1px solid rgba(56, 189, 248, 0.2);
}

.article-content :deep(pre code) {
  background: transparent;
  padding: 0;
  color: inherit;
  border: none;
}

/* ===== BLOCKQUOTE ===== */
.article-content :deep(blockquote) {
  border-left: 4px solid #38bdf8;
  padding: 20px 28px;
  margin: 28px 0;
  background: linear-gradient(135deg,
    rgba(56, 189, 248, 0.08) 0%,
    rgba(56, 189, 248, 0.04) 100%
  );
  border-radius: 0 16px 16px 0;
  font-style: italic;
  position: relative;
}

.article-content :deep(blockquote)::before {
  content: '"';
  position: absolute;
  top: -10px;
  left: 15px;
  font-size: 48px;
  color: rgba(56, 189, 248, 0.2);
  font-family: Georgia, serif;
  line-height: 1;
}

.article-content :deep(ul),
.article-content :deep(ol) {
  padding-left: 32px;
  margin-bottom: 24px;
}

.article-content :deep(li) {
  margin-bottom: 12px;
}

.article-content :deep(li)::marker {
  color: #38bdf8;
}

.article-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 28px 0;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
}

.article-content :deep(th) {
  background: linear-gradient(135deg,
    rgba(56, 189, 248, 0.15) 0%,
    rgba(56, 189, 248, 0.08) 100%
  );
  padding: 16px;
  text-align: left;
  font-weight: 600;
  border: 1px solid rgba(56, 189, 248, 0.2);
  color: #f1f5f9;
  font-family: 'Space Grotesk', sans-serif;
}

.article-content :deep(td) {
  padding: 16px;
  border: 1px solid rgba(56, 189, 248, 0.1);
  background: rgba(15, 23, 42, 0.5);
}

.article-content :deep(tr:hover td) {
  background: rgba(56, 189, 248, 0.05);
}

.article-content :deep(hr) {
  border: none;
  height: 2px;
  background: linear-gradient(90deg,
    transparent,
    rgba(56, 189, 248, 0.3),
    transparent
  );
  margin: 48px 0;
}

/* ===== 文章页脚 ===== */
.article-footer {
  padding: 32px 0;
  margin-top: 48px;
  border-top: 1px solid rgba(56, 189, 248, 0.15);
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 24px;
}

.article-tags {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.tags-label {
  font-size: 13px;
  color: rgba(148, 163, 184, 0.8);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-family: 'Space Grotesk', sans-serif;
}

.tag {
  padding: 8px 16px;
  background: rgba(56, 189, 248, 0.08);
  border: 1px solid rgba(56, 189, 248, 0.2);
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  color: #38bdf8;
  text-decoration: none;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.tag:hover {
  background: linear-gradient(135deg, #38bdf8, #0ea5e9);
  color: white;
  border-color: transparent;
  box-shadow: 0 4px 12px rgba(56, 189, 248, 0.4);
  transform: translateY(-2px);
}

.article-share {
  display: flex;
  align-items: center;
  gap: 12px;
}

.share-label {
  font-size: 13px;
  color: rgba(148, 163, 184, 0.8);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-family: 'Space Grotesk', sans-serif;
}

.share-button {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(56, 189, 248, 0.08);
  border: 1px solid rgba(56, 189, 248, 0.2);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  font-size: 18px;
  color: #38bdf8;
}

.share-button:hover {
  background: linear-gradient(135deg, #38bdf8, #0ea5e9);
  border-color: transparent;
  color: white;
  transform: translateY(-3px) scale(1.05);
  box-shadow: 0 6px 20px rgba(56, 189, 248, 0.4);
}

/* ===== 文章导航 ===== */
.article-navigation {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 20px;
  margin: 48px 0;
}

.nav-card {
  padding: 20px 24px;
  background: rgba(17, 24, 39, 0.8);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(56, 189, 248, 0.1);
  border-radius: 16px;
  text-decoration: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
  transition: all 0.3s;
  min-height: 120px;
  justify-content: center;
}

.nav-card:hover {
  border-color: rgba(56, 189, 248, 0.3);
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
}

.nav-label {
  font-size: 11px;
  color: rgba(148, 163, 184, 0.7);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  font-weight: 700;
  font-family: 'Space Grotesk', sans-serif;
}

.nav-title {
  font-size: 15px;
  font-weight: 600;
  color: #f1f5f9;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  font-family: 'Space Grotesk', sans-serif;
  line-height: 1.4;
}

.nav-arrow {
  font-size: 20px;
  color: #38bdf8;
}

.nav-prev {
  align-items: flex-start;
}

.nav-next {
  align-items: flex-end;
  text-align: right;
}

.nav-back {
  align-items: center;
  text-align: center;
  background: linear-gradient(135deg, #38bdf8, #0ea5e9);
  border: none;
}

.nav-back .nav-label,
.nav-back .nav-title,
.nav-back .nav-icon {
  color: white;
}

.nav-icon {
  font-size: 28px;
  color: #38bdf8;
}

.nav-placeholder {
  min-height: 120px;
}

/* ===== 相关文章 ===== */
.related-section {
  margin-top: 64px;
  padding: 0 48px;
}

.related-title {
  font-family: 'Space Grotesk', sans-serif;
  font-size: 24px;
  font-weight: 700;
  color: #f1f5f9;
  margin: 0 0 32px;
  letter-spacing: -0.01em;
}

.related-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.related-card {
  display: flex;
  gap: 16px;
  padding: 16px;
  background: rgba(17, 24, 39, 0.6);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(56, 189, 248, 0.1);
  border-radius: 16px;
  text-decoration: none;
  transition: all 0.3s;
}

.related-card:hover {
  border-color: rgba(56, 189, 248, 0.3);
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
}

.related-cover {
  width: 100px;
  height: 70px;
  border-radius: 10px;
  overflow: hidden;
  flex-shrink: 0;
}

.related-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.related-icon {
  width: 100px;
  height: 70px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(56, 189, 248, 0.1);
  border-radius: 10px;
  font-size: 28px;
  color: #38bdf8;
  flex-shrink: 0;
}

.related-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.related-title-text {
  font-size: 15px;
  font-weight: 600;
  color: #f1f5f9;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  font-family: 'Space Grotesk', sans-serif;
  line-height: 1.4;
}

.related-summary {
  font-size: 13px;
  color: rgba(148, 163, 184, 0.7);
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.related-date {
  font-size: 11px;
  color: rgba(148, 163, 184, 0.6);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* ===== 响应式设计 ===== */
@media (min-width: 1024px) {
  .content-wrapper {
    grid-template-columns: 280px 1fr;
    gap: 40px;
  }

  .sidebar {
    order: -1;
  }

  .toc-card {
    margin: 0;
  }
}

@media (min-width: 1440px) {
  .post-article {
    margin: 0 64px;
  }

  .breadcrumb {
    margin: 0 64px 24px;
  }

  .related-section {
    padding: 0 64px;
  }
}

@media (max-width: 768px) {
  .post-article {
    margin: 0 20px;
  }

  .breadcrumb {
    margin: 0 20px 20px;
    font-size: 12px;
  }

  .breadcrumb-current {
    max-width: 150px;
  }

  .article-title {
    font-size: 28px;
  }

  .article-excerpt {
    font-size: 16px;
  }

  .article-footer {
    flex-direction: column;
    align-items: flex-start;
  }

  .article-navigation {
    grid-template-columns: 1fr;
  }

  .nav-placeholder {
    display: none;
  }

  .nav-back {
    order: -1;
    margin-bottom: 12px;
  }

  .related-section {
    padding: 0 20px;
  }

  .related-grid {
    grid-template-columns: 1fr;
  }
}

/* ===== 打印样式 ===== */
@media print {
  .reading-progress,
  .sidebar,
  .article-navigation,
  .related-section,
  .article-footer,
  .breadcrumb {
    display: none !important;
  }

  .post-article {
    margin: 0;
  }

  .article-content {
    color: black;
  }
}

/* ===== 动画 ===== */
@keyframes floatOrb {
  0%, 100% { transform: translate(0, 0) scale(1); }
  33% { transform: translate(30px, -30px) scale(1.05); }
  66% { transform: translate(-20px, 20px) scale(0.95); }
}
</style>

<style>
/* ===== GLOBAL STYLES FOR POST DETAIL PAGE ===== */
/* Ensure Ant Design components use dark theme colors */

.post-detail-page {
  color: #f1f5f9;
}

.post-detail-page :deep(.ant-spin) {
  color: #38bdf8;
}

.post-detail-page :deep(.ant-spin-dot-item) {
  background-color: #38bdf8;
}

.post-detail-page :deep(.ant-empty) {
  color: rgba(148, 163, 184, 0.8);
}

.post-detail-page :deep(.ant-empty-description) {
  color: rgba(148, 163, 184, 0.8);
  font-size: 15px;
}

.post-detail-page :deep(.ant-btn) {
  font-family: 'Space Grotesk', sans-serif;
  font-weight: 600;
  border-radius: 12px;
  height: 44px;
  padding: 0 24px;
}

.post-detail-page :deep(.ant-btn-primary) {
  background: linear-gradient(135deg, #38bdf8 0%, #0ea5e9 100%);
  border: none;
  box-shadow: 0 4px 12px rgba(56, 189, 248, 0.3);
}

.post-detail-page :deep(.ant-btn-primary:hover) {
  background: linear-gradient(135deg, #0ea5e9 0%, #0284c7 100%);
  box-shadow: 0 6px 20px rgba(56, 189, 248, 0.4);
  transform: translateY(-2px);
}

.post-detail-page :deep(.ant-btn-link) {
  color: #38bdf8;
}
</style>
