<template>
  <div class="preview-hero">
    <div class="hero-badge">{{ content.badge || '徽标' }}</div>
    <h2>{{ content.title || '标题' }}</h2>
    <p>{{ content.subtitle || '副标题' }}</p>
    <div class="hero-tags">
      <a-tag v-for="(item, index) in content.features || []" :key="index">{{ item }}</a-tag>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ content: { badge?: string; title?: string; subtitle?: string; features?: string[] } }>();
</script>

<style scoped>
.preview-hero {
  padding: 16px;
  border-radius: 12px;
  background: #0f172a;
  color: #e2e8f0;
}

.hero-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(56, 189, 248, 0.2);
  color: #38bdf8;
  font-size: 12px;
}

.hero-tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 8px;
}
</style>
