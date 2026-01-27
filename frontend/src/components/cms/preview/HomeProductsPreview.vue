<template>
  <div class="preview-grid">
    <div class="preview-card" v-for="(item, index) in content.items || []" :key="index">
      <div class="card-tag">{{ item.tag || '标签' }}</div>
      <div class="card-title">{{ item.title || '标题' }}</div>
      <div class="card-desc">{{ item.description || '描述内容' }}</div>
      <div class="card-price">￥{{ item.price || '--' }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface ProductItem {
  tag?: string;
  title?: string;
  description?: string;
  price?: string;
}

defineProps<{ content: { items?: ProductItem[] } }>();
</script>

<style scoped>
.preview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}

.preview-card {
  background: #ffffff;
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 10px;
  padding: 12px;
}

.card-tag {
  display: inline-block;
  font-size: 12px;
  color: #1677ff;
  background: rgba(22, 119, 255, 0.1);
  padding: 2px 8px;
  border-radius: 999px;
  margin-bottom: 6px;
}

.card-title {
  font-weight: 600;
  margin-bottom: 6px;
}

.card-desc {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.6);
  margin-bottom: 8px;
}

.card-price {
  font-weight: 600;
  color: #f97316;
}
</style>
