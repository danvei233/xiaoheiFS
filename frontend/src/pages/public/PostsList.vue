<template>
  <div class="posts-list-page" ref="postsPage">
    <!-- Background Effects -->
    <div class="page-background">
      <div class="grid-overlay"></div>
      <div class="glow-orb glow-1"></div>
      <div class="glow-orb glow-2"></div>
    </div>

    <!-- Blocks-driven rendering (docs/announcements/activities/tutorials share this page) -->
    <template v-for="block in resolvedBlocks" :key="block.type">
      <!-- Hero -->
      <section v-if="block.type === 'hero'" class="page-hero">
        <div class="hero-container">
          <div class="hero-breadcrumb scroll-animate">
            <router-link to="/">首页</router-link>
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path
                d="M3 7H11M11 7L7 3M11 7L7 11"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            <span>{{ categoryTitle }}</span>
          </div>

          <h1 class="hero-title scroll-animate">{{ heroTitle }}</h1>
          <p class="hero-subtitle scroll-animate">{{ heroSubtitle }}</p>

          <!-- Category Pills -->
          <div class="category-pills scroll-animate" v-if="categories.length > 1">
            <button
              v-for="cat in categories"
              :key="cat.key"
              class="category-pill"
              :class="{ active: currentCategory === cat.key }"
              @click="switchCategory(cat.key)"
            >
              <component :is="cat.icon" class="pill-icon" />
              <span class="pill-name">{{ cat.name }}</span>
              <span class="pill-count">{{ cat.count }}</span>
            </button>
          </div>
        </div>
      </section>

      <!-- Posts -->
      <template v-else-if="block.type === 'posts'">
        <section class="filter-section scroll-animate">
          <div class="filter-container">
            <div class="search-box">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                <circle cx="9" cy="9" r="7" stroke="currentColor" stroke-width="2" />
                <path d="M14 14L17 17" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
              </svg>
              <input v-model="searchQuery" type="text" :placeholder="searchPlaceholder" class="search-input" />
            </div>
            <div class="filter-actions">
              <div class="sort-select">
                <span class="sort-label">排序:</span>
                <select v-model="sortBy" class="sort-dropdown">
                  <option value="latest">最新发布</option>
                  <option value="popular">最受欢迎</option>
                  <option value="title">标题 A-Z</option>
                </select>
              </div>
            </div>
          </div>
        </section>

        <section class="featured-section" v-if="featuredPost && !searchQuery">
          <div class="featured-container scroll-animate">
            <div class="featured-badge">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <path
                  d="M8 1L2 6H5V14H11V6H14L8 1Z"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linejoin="round"
                />
              </svg>
              <span>精选推荐</span>
            </div>
            <div class="featured-card" @click="goToPost(featuredPost.slug)">
              <div class="featured-image" v-if="featuredPost.cover_url">
                <img :src="featuredPost.cover_url" :alt="featuredPost.title" />
                <div class="featured-overlay"></div>
              </div>
              <div class="featured-content">
                <div class="featured-meta">
                  <span class="featured-tag">{{ categoryTitle }}</span>
                  <span class="featured-date">{{ formatDate(featuredPost.published_at) }}</span>
                </div>
                <h2 class="featured-title">{{ featuredPost.title }}</h2>
                <p class="featured-summary">{{ featuredPost.summary }}</p>
                <div class="featured-footer">
                  <span class="read-time">{{ estimatedReadTime(featuredPost.summary) }} 阅读</span>
                  <span class="read-more">
                    阅读全文
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                      <path
                        d="M3 8H13M13 8L9 4M13 8L9 12"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                      />
                    </svg>
                  </span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="posts-section" ref="postsSection">
          <div class="posts-container">
            <div class="results-header scroll-animate" v-if="!loading">
              <span class="results-count">找到 <strong>{{ filteredPosts.length }}</strong> 篇文章</span>
            </div>

            <div class="loading-state" v-if="loading">
              <div class="loading-spinner"></div>
              <p>加载中...</p>
            </div>

            <div class="posts-grid" v-else-if="filteredPosts.length > 0">
              <article
                v-for="(post, index) in filteredPosts"
                :key="post.id"
                class="post-card scroll-animate"
                :style="{ '--delay': `${index * 0.08}s` }"
                @click="goToPost(post.slug)"
              >
                <div class="post-card-inner">
                  <div class="post-image" v-if="post.cover_url">
                    <img :src="post.cover_url" :alt="post.title" />
                    <div class="post-image-overlay"></div>
                    <div class="post-category-badge">{{ categoryTitle }}</div>
                  </div>
                  <div class="post-image-placeholder" v-else>
                    <div class="placeholder-icon">
                      <svg width="48" height="48" viewBox="0 0 24 24" fill="none">
                        <path
                          d="M4 4H20C21.1046 4 22 4.89543 22 6V18C22 19.1046 21.1046 20 20 20H4C2.89543 20 2 19.1046 2 18V6C2 4.89543 2.89543 4 4 4Z"
                          stroke="currentColor"
                          stroke-width="2"
                        />
                        <path d="M2 10H22" stroke="currentColor" stroke-width="2" />
                      </svg>
                    </div>
                    <div class="post-category-badge">{{ categoryTitle }}</div>
                  </div>

                  <div class="post-content">
                    <div class="post-meta">
                      <span class="post-date">
                        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                          <circle cx="7" cy="7" r="6" stroke="currentColor" stroke-width="1.5" />
                          <path
                            d="M7 4V7L9 9"
                            stroke="currentColor"
                            stroke-width="1.5"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                          />
                        </svg>
                        {{ formatDate(post.published_at) }}
                      </span>
                      <span class="post-read-time">{{ estimatedReadTime(post.summary) }}</span>
                    </div>

                    <h3 class="post-title">{{ post.title }}</h3>
                    <p class="post-summary">{{ post.summary }}</p>

                    <div class="post-footer">
                      <span class="post-cta">
                        阅读更多
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                          <path
                            d="M3 8H13M13 8L9 4M13 8L9 12"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                          />
                        </svg>
                      </span>
                    </div>
                  </div>

                  <div class="post-glow"></div>
                </div>
              </article>
            </div>

            <div class="empty-state" v-else>
              <h3 class="empty-title">暂无相关内容</h3>
              <p class="empty-desc">{{ searchQuery ? '尝试更换搜索关键词' : '敬请期待更多精彩内容' }}</p>
              <button v-if="searchQuery" @click="clearSearch" class="clear-search-btn">清除搜索</button>
            </div>

            <div class="pagination" v-if="posts.length > 0 && hasMore">
              <button class="pagination-btn" :disabled="loadingMore" @click="loadMore">
                <span v-if="loadingMore">加载中...</span>
                <span v-else>加载更多</span>
              </button>
            </div>
          </div>
        </section>
      </template>

      <!-- Resources -->
      <section v-else-if="block.type === 'resources'" class="resources-section scroll-animate">
        <div class="resources-container">
          <h2 class="resources-title">{{ resourcesTitle }}</h2>
          <div class="resources-grid">
            <a v-for="(resource, index) in resources" :key="index" :href="resource.url || '#'" class="resource-card">
              <div class="resource-icon" :class="`resource-icon-${index + 1}`">
                <component :is="resource.icon" />
              </div>
              <div class="resource-content">
                <h4 class="resource-title">{{ resource.title }}</h4>
                <p class="resource-desc">{{ resource.description }}</p>
              </div>
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none" class="resource-arrow">
                <path
                  d="M3 8H13M13 8L9 4M13 8L9 12"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </a>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { h, defineComponent } from 'vue'
import { getCmsBlocks, getCmsPosts } from '@/services/user'

const route = useRoute()
const router = useRouter()

const postsPage = ref<HTMLElement | null>(null)
const postsSection = ref<HTMLElement | null>(null)

const posts = ref<any[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const searchQuery = ref('')
const sortBy = ref<'latest' | 'popular' | 'title'>('latest')
const currentCategory = ref<string>((route.meta?.categoryKey as string) || 'docs')
const total = ref(0)

const blocks = ref<Array<{ type: string; visible?: boolean; sort_order?: number; content_json?: string }>>([])

const BookIcon = defineComponent({
  render: () =>
    h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
      h('path', {
        d: 'M4 4H20C21.1046 4 22 4.89543 22 6V18C22 19.1046 21.1046 20 20 20H4C2.89543 20 2 19.1046 2 18V6C2 4.89543 2.89543 4 4 4Z',
        stroke: 'currentColor',
        'stroke-width': 2
      }),
      h('path', { d: 'M2 10H22', stroke: 'currentColor', 'stroke-width': 2 })
    ])
})

const VideoIcon = defineComponent({
  render: () =>
    h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
      h('path', { d: 'M22 8L16 12L22 16V8Z', fill: 'currentColor' }),
      h('rect', { x: '2', y: '6', width: '14', height: '12', rx: '2', stroke: 'currentColor', 'stroke-width': 2 })
    ])
})

const CodeIcon = defineComponent({
  render: () =>
    h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
      h('polyline', {
        points: '16 18 22 12 16 6',
        stroke: 'currentColor',
        'stroke-width': 2,
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round'
      }),
      h('polyline', {
        points: '8 6 2 12 8 18',
        stroke: 'currentColor',
        'stroke-width': 2,
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round'
      })
    ])
})

const ChatIcon = defineComponent({
  render: () =>
    h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
      h('path', {
        d: 'M21 11.5C21 16.1944 16.9706 20 12 20C10.9406 20 9.92306 19.8361 8.97438 19.5312L4 21L5.26316 17.2105C3.82973 15.7557 3 13.7351 3 11.5C3 6.80558 7.02944 3 12 3C16.9706 3 21 6.80558 21 11.5Z',
        stroke: 'currentColor',
        'stroke-width': 2,
        'stroke-linejoin': 'round'
      })
    ])
})

const categories = ref([
  { key: 'docs', name: '文档', icon: BookIcon, count: 0 },
  { key: 'announcements', name: '公告', icon: BookIcon, count: 0 },
  { key: 'activities', name: '活动', icon: BookIcon, count: 0 },
  { key: 'tutorials', name: '教程', icon: BookIcon, count: 0 }
])

const pageTitle = computed(() => (route.meta?.title as string) || '文章列表')
const pageSubtitle = computed(() => (route.meta?.subtitle as string) || '')

const categoryTitle = computed(() => {
  const titles: Record<string, string> = {
    docs: '文档中心',
    announcements: '产品公告',
    activities: '活动中心',
    tutorials: '教程学院'
  }
  return titles[currentCategory.value] || '文章'
})

const searchPlaceholder = computed(() => {
  const placeholders: Record<string, string> = {
    docs: '搜索文档...',
    announcements: '搜索公告...',
    activities: '搜索活动...',
    tutorials: '搜索教程...'
  }
  return placeholders[currentCategory.value] || '搜索...'
})

const safeParse = (raw?: string) => {
  if (!raw) return {}
  try {
    return JSON.parse(raw)
  } catch {
    return {}
  }
}

const resolvedBlocks = computed(() => {
  const defaults = [
    { type: 'hero', visible: true, sort_order: 1 },
    { type: 'posts', visible: true, sort_order: 2 },
    { type: 'resources', visible: true, sort_order: 3 }
  ]
  const map = new Map<string, any>()
  defaults.forEach((d) => map.set(d.type, d))
  blocks.value.forEach((b) => {
    if (!b?.type) return
    map.set(b.type, b)
  })
  return Array.from(map.values())
    .filter((b) => b.visible !== false)
    .sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0))
})

const heroBlock = computed(() => safeParse(blocks.value.find((b) => b.type === 'hero')?.content_json))
const heroTitle = computed(() => heroBlock.value.title || pageTitle.value)
const heroSubtitle = computed(() => heroBlock.value.subtitle || pageSubtitle.value)

const resourcesBlock = computed(() => safeParse(blocks.value.find((b) => b.type === 'resources')?.content_json))
const resourcesTitle = computed(() => resourcesBlock.value.title || '相关资源')

const resources = computed(() => {
  const fallback = [
    { icon: BookIcon, title: 'API 文档', description: '完整的 API 参考手册和示例代码', url: '#' },
    { icon: VideoIcon, title: '视频教程', description: '手把手教您使用各项功能', url: '#' },
    { icon: CodeIcon, title: '代码示例', description: '常用场景的代码片段和最佳实践', url: '#' },
    { icon: ChatIcon, title: '社区支持', description: '加入讨论，获取帮助与经验分享', url: '#' }
  ]

  const items = Array.isArray(resourcesBlock.value.items) ? resourcesBlock.value.items : []
  if (!items.length) return fallback

  const iconMap: Record<string, any> = { book: BookIcon, video: VideoIcon, code: CodeIcon, chat: ChatIcon }
  return items.map((it: any, idx: number) => ({
    icon: iconMap[it.icon_key] || fallback[idx % fallback.length].icon,
    title: it.title || fallback[idx % fallback.length].title,
    description: it.description || fallback[idx % fallback.length].description,
    url: it.url || '#'
  }))
})

const featuredPost = computed(() => {
  if (searchQuery.value) return null
  return posts.value.find((p) => p.cover_url) || null
})

const filteredPosts = computed(() => {
  let result = [...posts.value]

  if (featuredPost.value && !searchQuery.value) {
    result = result.filter((p) => p !== featuredPost.value)
  }

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(
      (post) => (post.title || '').toLowerCase().includes(query) || (post.summary || '').toLowerCase().includes(query)
    )
  }

  if (sortBy.value === 'latest') {
    result.sort((a, b) => new Date(b.published_at).getTime() - new Date(a.published_at).getTime())
  } else if (sortBy.value === 'title') {
    result.sort((a, b) => (a.title || '').localeCompare(b.title || ''))
  }
  return result
})

const hasMore = computed(() => posts.value.length < total.value)

const formatDate = (date: string) => {
  if (!date) return ''
  const d = new Date(date)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  if (days === 0) return '今天'
  if (days === 1) return '昨天'
  if (days < 7) return `${days} 天前`
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
}

const estimatedReadTime = (text: string) => {
  const charsPerMinute = 800
  const chars = (text || '').length
  const minutes = Math.max(1, Math.ceil(chars / charsPerMinute))
  return `${minutes} 分钟`
}

const goToPost = (slug: string) => {
  const categoryKey = (route.meta?.categoryKey as string) || 'docs'
  router.push(`/${categoryKey}/${slug}`)
}

const switchCategory = async (key: string) => {
  currentCategory.value = key
  router.push(`/${key}`)
}

const clearSearch = () => {
  searchQuery.value = ''
}

const fetchBlocks = async () => {
  try {
    const lang = 'zh-CN'
    const res = await getCmsBlocks({ page: currentCategory.value, lang })
    blocks.value = res.data?.items || []
  } catch {
    blocks.value = []
  }
}

const fetchPosts = async (opts?: { append?: boolean }) => {
  const append = !!opts?.append
  loading.value = !append
  loadingMore.value = append

  try {
    const lang = 'zh-CN'
    const limit = 20
    const offset = append ? posts.value.length : 0
    const res = await getCmsPosts({ category_key: currentCategory.value, lang, limit, offset })
    total.value = res.data?.total || 0
    const items = res.data?.items || []
    posts.value = append ? posts.value.concat(items) : items
  } catch {
    if (!append) posts.value = []
    total.value = 0
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

const refreshCategoryCounts = async () => {
  try {
    const lang = 'zh-CN'
    const keys = categories.value.map((c) => c.key)
    const results = await Promise.all(
      keys.map(async (key) => {
        try {
          const res = await getCmsPosts({ category_key: key, lang, limit: 1, offset: 0 })
          return { key, total: res.data?.total || 0 }
        } catch {
          return { key, total: 0 }
        }
      })
    )
    const map = new Map(results.map((r) => [r.key, r.total]))
    categories.value = categories.value.map((c) => ({ ...c, count: map.get(c.key) || 0 }))
  } catch {
    // ignore
  }
}

const loadMore = async () => {
  if (loadingMore.value || !hasMore.value) return
  await fetchPosts({ append: true })
}

const initScrollAnimations = () => {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) entry.target.classList.add('scroll-animate-active')
      })
    },
    { threshold: 0.1, rootMargin: '0px 0px -50px 0px' }
  )
  document.querySelectorAll('.scroll-animate').forEach((el) => observer.observe(el))
}

watch(
  () => route.meta?.categoryKey,
  async (key) => {
    currentCategory.value = (key as string) || 'docs'
    await fetchBlocks()
    await fetchPosts({ append: false })
    refreshCategoryCounts()
    setTimeout(initScrollAnimations, 50)
  },
  { immediate: true }
)

onMounted(() => {
  setTimeout(initScrollAnimations, 100)
})

onUnmounted(() => {
  // Cleanup
})
</script>

<style scoped>
:root {
  --color-bg: #0a0e17;
  --color-bg-alt: #111827;
  --color-bg-card: #161f33;
  --color-primary: #0ea5e9;
  --color-primary-light: #38bdf8;
  --color-accent: #f97316;
  --color-text: #f1f5f9;
  --color-text-muted: #94a3b8;
  --color-border: #1e293b;
  --font-heading: 'Outfit', sans-serif;
  --font-body: 'Work Sans', sans-serif;
}

.posts-list-page {
  min-height: 100vh;
  font-family: var(--font-body);
  background: var(--color-bg);
  color: var(--color-text);
  position: relative;
  overflow-x: hidden;
}

.page-background {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
  z-index: 0;
}

.grid-overlay {
  position: absolute;
  inset: 0;
  background-image: linear-gradient(rgba(14, 165, 233, 0.08) 1px, transparent 1px),
    linear-gradient(90deg, rgba(14, 165, 233, 0.08) 1px, transparent 1px);
  background-size: 80px 80px;
  opacity: 0.25;
}

.glow-orb {
  position: absolute;
  width: 520px;
  height: 520px;
  border-radius: 50%;
  filter: blur(90px);
  opacity: 0.35;
}

.glow-1 {
  background: rgba(14, 165, 233, 0.55);
  top: -140px;
  left: -140px;
}

.glow-2 {
  background: rgba(249, 115, 22, 0.45);
  bottom: -160px;
  right: -160px;
}

.page-hero {
  position: relative;
  z-index: 1;
  padding: 120px 24px 60px;
}

.hero-container {
  max-width: 1200px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
}

.hero-breadcrumb {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--color-text-muted);
  font-size: 14px;
  margin-bottom: 18px;
}

.hero-breadcrumb a {
  color: var(--color-text-muted);
  text-decoration: none;
}

.hero-title {
  font-family: var(--font-heading);
  font-size: 54px;
  font-weight: 700;
  line-height: 1.1;
  margin: 0 0 18px;
}

.hero-subtitle {
  max-width: 760px;
  font-size: 18px;
  line-height: 1.7;
  color: var(--color-text-muted);
  margin: 0 0 28px;
}

.category-pills {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 8px;
}

.category-pill {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: 999px;
  background: rgba(17, 24, 39, 0.6);
  border: 1px solid rgba(30, 41, 59, 0.8);
  color: var(--color-text);
  cursor: pointer;
}

.category-pill.active {
  border-color: rgba(56, 189, 248, 0.6);
  background: rgba(14, 165, 233, 0.12);
}

.pill-icon {
  width: 16px;
  height: 16px;
  color: var(--color-primary-light);
}

.pill-count {
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.12);
  color: var(--color-text-muted);
  font-size: 12px;
}

.filter-section {
  position: relative;
  z-index: 1;
  padding: 0 24px 30px;
}

.filter-container {
  max-width: 1200px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: 14px;
  background: rgba(17, 24, 39, 0.7);
  border: 1px solid rgba(30, 41, 59, 0.9);
  min-width: 320px;
}

.search-input {
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-text);
  width: 100%;
}

.filter-actions {
  display: flex;
  align-items: center;
  gap: 14px;
}

.sort-select {
  display: flex;
  align-items: center;
  gap: 10px;
}

.sort-label {
  color: var(--color-text-muted);
  font-size: 14px;
}

.sort-dropdown {
  border-radius: 12px;
  padding: 10px 12px;
  background: rgba(17, 24, 39, 0.7);
  border: 1px solid rgba(30, 41, 59, 0.9);
  color: var(--color-text);
}

.featured-section {
  position: relative;
  z-index: 1;
  padding: 0 24px 40px;
}

.featured-container {
  max-width: 1200px;
  margin: 0 auto;
}

.featured-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(14, 165, 233, 0.12);
  border: 1px solid rgba(56, 189, 248, 0.3);
  color: var(--color-primary-light);
  margin-bottom: 14px;
}

.featured-card {
  display: grid;
  grid-template-columns: 300px 1fr;
  background: rgba(17, 24, 39, 0.85);
  border: 1px solid rgba(30, 41, 59, 0.95);
  border-radius: 18px;
  overflow: hidden;
  cursor: pointer;
}

.featured-image {
  position: relative;
}

.featured-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.featured-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, transparent, rgba(10, 14, 23, 0.85));
}

.featured-content {
  padding: 26px 28px;
  display: flex;
  flex-direction: column;
}

.featured-meta {
  display: flex;
  gap: 12px;
  color: var(--color-text-muted);
  font-size: 13px;
  margin-bottom: 10px;
}

.featured-tag {
  color: var(--color-primary-light);
}

.featured-title {
  font-family: var(--font-heading);
  font-size: 28px;
  margin: 0 0 10px;
}

.featured-summary {
  color: var(--color-text-muted);
  line-height: 1.7;
  margin: 0 0 14px;
}

.featured-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
  color: var(--color-text-muted);
  font-size: 13px;
}

.read-more {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--color-primary-light);
}

.posts-section {
  position: relative;
  z-index: 1;
  padding: 0 24px 40px;
}

.posts-container {
  max-width: 1200px;
  margin: 0 auto;
}

.results-header {
  margin: 14px 0 18px;
  color: var(--color-text-muted);
}

.loading-state {
  padding: 60px 0;
  text-align: center;
  color: var(--color-text-muted);
}

.loading-spinner {
  width: 34px;
  height: 34px;
  margin: 0 auto 12px;
  border-radius: 50%;
  border: 3px solid rgba(148, 163, 184, 0.25);
  border-top-color: rgba(56, 189, 248, 0.9);
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.posts-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.post-card {
  cursor: pointer;
}

.post-card-inner {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: rgba(17, 24, 39, 0.82);
  border: 1px solid rgba(30, 41, 59, 0.95);
  border-radius: 18px;
  overflow: hidden;
  position: relative;
  transition: transform 0.2s ease, border-color 0.2s ease;
}

.post-card-inner:hover {
  transform: translateY(-4px);
  border-color: rgba(56, 189, 248, 0.45);
}

.post-image,
.post-image-placeholder {
  position: relative;
  height: 160px;
  background: rgba(17, 24, 39, 0.6);
}

.post-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.post-image-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, transparent, rgba(10, 14, 23, 0.9));
}

.post-category-badge {
  position: absolute;
  left: 14px;
  bottom: 12px;
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(14, 165, 233, 0.14);
  border: 1px solid rgba(56, 189, 248, 0.25);
  color: var(--color-primary-light);
  font-size: 12px;
}

.placeholder-icon {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  color: rgba(148, 163, 184, 0.35);
}

.post-content {
  padding: 18px 18px 16px;
  display: flex;
  flex-direction: column;
  flex: 1;
}

.post-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--color-text-muted);
  font-size: 12px;
  margin-bottom: 10px;
}

.post-date {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.post-title {
  font-family: var(--font-heading);
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 12px;
}

.post-summary {
  font-size: 14px;
  line-height: 1.6;
  color: var(--color-text-muted);
  margin: 0 0 16px;
  flex: 1;
}

.post-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--color-primary-light);
  font-size: 13px;
}

.post-cta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.empty-state {
  padding: 60px 0;
  text-align: center;
}

.empty-title {
  font-family: var(--font-heading);
  font-size: 20px;
  margin: 0 0 10px;
}

.empty-desc {
  margin: 0;
  color: var(--color-text-muted);
}

.clear-search-btn {
  margin-top: 12px;
  padding: 10px 16px;
  background: rgba(14, 165, 233, 0.1);
  border: 1px solid rgba(14, 165, 233, 0.2);
  border-radius: 8px;
  color: var(--color-primary-light);
  cursor: pointer;
}

.pagination {
  display: flex;
  justify-content: center;
  padding: 24px 0;
}

.pagination-btn {
  padding: 12px 20px;
  background: var(--color-bg-alt);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  color: var(--color-text);
  cursor: pointer;
}

.resources-section {
  padding: 60px 24px 80px;
  position: relative;
  z-index: 1;
}

.resources-container {
  max-width: 1200px;
  margin: 0 auto;
}

.resources-title {
  font-family: var(--font-heading);
  font-size: 30px;
  margin: 0 0 24px;
}

.resources-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
}

.resource-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: var(--color-bg-alt);
  border: 1px solid var(--color-border);
  border-radius: 16px;
  text-decoration: none;
  color: var(--color-text);
}

.resource-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  color: var(--color-primary-light);
  flex-shrink: 0;
}

.resource-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.resource-title {
  font-weight: 600;
}

.resource-desc {
  color: var(--color-text-muted);
  font-size: 13px;
  line-height: 1.4;
}

/* Scroll Animations */
.scroll-animate {
  opacity: 0;
  transform: translateY(30px);
  transition: all 0.6s cubic-bezier(0.4, 0, 0.2, 1);
}

.scroll-animate-active {
  opacity: 1;
  transform: translateY(0);
}

@media (max-width: 1024px) {
  .posts-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .resources-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .featured-card {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .hero-title {
    font-size: 36px;
  }
  .posts-grid {
    grid-template-columns: 1fr;
  }
  .resources-grid {
    grid-template-columns: 1fr;
  }
  .filter-container {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
