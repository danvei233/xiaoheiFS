<template>
  <div class="preview-grid">
    <div class="preview-card" v-for="(item, index) in content.products || []" :key="index">
      <div class="card-title">{{ item.name }}</div>
      <div class="card-desc">{{ item.description }}</div>
      <div class="card-price">￥{{ item.price }}</div>
      <ul class="card-features">
        <li v-for="(feature, fIndex) in item.features || []" :key="fIndex">{{ feature }}</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ content: { products?: Array<{ name?: string; description?: string; price?: string; features?: string[] }> } }>();
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

.card-title {
  font-weight: 600;
  margin-bottom: 6px;
}

.card-desc {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.6);
  margin-bottom: 6px;
}

.card-price {
  color: #f97316;
  font-weight: 600;
  margin-bottom: 6px;
}

.card-features {
  padding-left: 16px;
  margin: 0;
  font-size: 12px;
}
</style>
