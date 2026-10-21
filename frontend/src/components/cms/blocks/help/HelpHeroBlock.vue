<template>
  <section class="help-hero">
    <div class="hero-glow"></div>
    <div class="hero-content">
      <div class="hero-badge">
        <BulbOutlined class="badge-icon" />
        <span>{{ resolved.badge }}</span>
      </div>
      <h1 class="hero-title">
        <span class="title-main">{{ resolved.title_main }}</span>
        <span class="title-gradient">{{ resolved.title_gradient }}</span>
      </h1>
      <p class="hero-subtitle">{{ resolved.subtitle }}</p>

      <!-- Search Bar -->
      <div class="search-container">
        <SearchOutlined class="search-icon" />
        <input v-model="queryModel" type="text" :placeholder="resolved.search_placeholder" class="search-input" />
        <div v-if="queryModel" class="search-clear" @click="queryModel = ''">✕</div>
      </div>

      <!-- Quick Stats -->
      <div class="quick-stats">
        <div class="stat-item" v-for="stat in resolved.quick_stats" :key="stat.label">
          <span class="stat-value">{{ stat.value }}</span>
          <span class="stat-label">{{ stat.label }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { BulbOutlined, SearchOutlined } from "@ant-design/icons-vue";

type StatItem = { value: string; label: string };

const props = defineProps<{
  content?: any;
  searchQuery: string;
}>();

const emit = defineEmits<{
  (e: "update:searchQuery", v: string): void;
}>();

const queryModel = computed({
  get: () => props.searchQuery,
  set: (v: string) => emit("update:searchQuery", v),
});

const resolved = computed(() => {
  const c = props.content || {};
  const stats: StatItem[] = Array.isArray(c.quick_stats)
    ? c.quick_stats.map((x: any) => ({ value: String(x?.value ?? ""), label: String(x?.label ?? "") }))
    : [];

  const fallbackStats: StatItem[] = [
    { value: "100+", label: "常见问题" },
    { value: "24/7", label: "在线支持" },
    { value: "<5m", label: "平均响应" },
    { value: "99.9%", label: "满意度" },
  ];

  return {
    badge: String(c.badge ?? "帮助中心"),
    title_main: String(c.title_main ?? "我们能为您"),
    title_gradient: String(c.title_gradient ?? "做些什么？"),
    subtitle: String(c.subtitle ?? "快速找到您需要的答案，或联系我们的专业团队获取支持"),
    search_placeholder: String(c.search_placeholder ?? "搜索问题、关键词..."),
    quick_stats: stats.length > 0 ? stats : fallbackStats,
  };
});
</script>

<style scoped>
/* Hero Section */
.help-hero {
  position: relative;
  padding: 80px 20px 60px;
  text-align: center;
  z-index: 1;
}

.hero-glow {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 600px;
  height: 600px;
  background: radial-gradient(circle, rgba(14, 165, 233, 0.15) 0%, transparent 70%);
  pointer-events: none;
}

.hero-content {
  position: relative;
  max-width: 800px;
  margin: 0 auto;
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: rgba(14, 165, 233, 0.1);
  border: 1px solid rgba(14, 165, 233, 0.3);
  border-radius: 20px;
  font-size: 14px;
  color: var(--color-primary-light);
  margin-bottom: 24px;
}

.badge-icon {
  font-size: 16px;
}

.hero-title {
  font-family: var(--font-heading);
  font-size: 56px;
  font-weight: 800;
  line-height: 1.1;
  margin: 0 0 20px;
}

.title-main {
  display: block;
  color: var(--color-text);
}

.title-gradient {
  display: block;
  background: linear-gradient(135deg, var(--color-primary-light), var(--color-accent));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-subtitle {
  font-size: 18px;
  color: var(--color-text-muted);
  margin: 0 0 40px;
  line-height: 1.6;
}

/* Search */
.search-container {
  position: relative;
  max-width: 600px;
  margin: 0 auto 40px;
}

.search-icon {
  position: absolute;
  left: 20px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 18px;
  color: var(--color-text-muted);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 18px 60px 18px 56px;
  font-size: 16px;
  font-family: var(--font-body);
  background: rgba(17, 24, 39, 0.8);
  border: 2px solid rgba(30, 41, 59, 1);
  border-radius: 16px;
  color: var(--color-text);
  outline: none;
  transition: all 0.3s;
}

.search-input:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 4px rgba(14, 165, 233, 0.1);
}

.search-input::placeholder {
  color: var(--color-text-muted);
}

.search-clear {
  position: absolute;
  right: 20px;
  top: 50%;
  transform: translateY(-50%);
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 50%;
  cursor: pointer;
  font-size: 14px;
  color: var(--color-text-muted);
  transition: all 0.2s;
}

.search-clear:hover {
  background: rgba(255, 255, 255, 0.2);
}

/* Quick Stats */
.quick-stats {
  display: flex;
  justify-content: center;
  gap: 40px;
  flex-wrap: wrap;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  background: linear-gradient(135deg, var(--color-primary-light), var(--color-accent));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.stat-label {
  font-size: 13px;
  color: var(--color-text-muted);
}

@media (max-width: 768px) {
  .hero-title {
    font-size: 40px;
  }

  .hero-subtitle {
    font-size: 16px;
  }

  .quick-stats {
    gap: 24px;
  }

  .stat-value {
    font-size: 24px;
  }
}
</style>
