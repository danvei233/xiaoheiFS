<template>
  <div class="preview-grid">
    <div class="preview-card" v-for="(item, index) in content.items || []" :key="index">
      <div class="card-title">{{ item.title || '标题' }}</div>
      <div class="card-desc">{{ item.description || '描述内容' }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface FeatureItem {
  title?: string;
  description?: string;
}

defineProps<{ content: { items?: FeatureItem[] } }>();
</script>

<style scoped>
.preview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 12px;
}

.preview-card {
  background: #ffffff;
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 10px;
  padding: 12px;
}

.card-title {
  font-weight: 600;
  margin-bottom: 6px;
}

.card-desc {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.6);
}
</style>
