<template>
  <div class="preview-block">
    <div class="hero-badge">
      <span class="badge-dot"></span>
      <span>{{ content.badge || '徽标' }}</span>
    </div>
    <h2 class="hero-title">
      <span>{{ content.title1 || '主标题' }}</span>
      <span class="title-gradient">{{ (content.typewriter_words && content.typewriter_words[0]) || '关键词' }}</span>
    </h2>
    <p class="hero-subtitle">{{ content.subtitle || '副标题文案' }}</p>
    <div class="hero-actions">
      <a-button type="primary">{{ content.primary_button_text || '主按钮' }}</a-button>
      <a-button>{{ content.secondary_button_text || '次按钮' }}</a-button>
    </div>
    <div class="hero-stats">
      <div class="stat-item" v-for="(stat, index) in content.stats || []" :key="index">
        <div class="stat-value">{{ stat.value }}{{ stat.suffix }}</div>
        <div class="stat-label">{{ stat.label }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface HeroContent {
  badge?: string;
  title1?: string;
  subtitle?: string;
  primary_button_text?: string;
  secondary_button_text?: string;
  typewriter_words?: string[];
  stats?: Array<{ value: number | string; suffix?: string; label?: string }>;
}

defineProps<{ content: HeroContent }>();
</script>

<style scoped>
.preview-block {
  padding: 16px;
  border-radius: 12px;
  background: #0f172a;
  color: #e2e8f0;
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(56, 189, 248, 0.2);
  font-size: 12px;
}

.badge-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #38bdf8;
}

.hero-title {
  margin: 12px 0 8px;
  font-size: 20px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.title-gradient {
  color: #38bdf8;
}

.hero-subtitle {
  color: #cbd5f5;
  margin-bottom: 12px;
}

.hero-actions {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.hero-stats {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.stat-item {
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 8px;
  padding: 8px 10px;
  min-width: 100px;
}

.stat-value {
  font-weight: 600;
}

.stat-label {
  font-size: 12px;
  color: #94a3b8;
}
</style>
