<template>
  <div class="preview-cta">
    <h3>{{ content.title || 'CTA 标题' }}</h3>
    <p>{{ content.desc || 'CTA 描述' }}</p>
    <div class="cta-actions">
      <a-button type="primary">{{ content.button_text || '按钮文案' }}</a-button>
    </div>
    <ul class="cta-list">
      <li v-for="(item, index) in content.features || []" :key="index">{{ item }}</li>
    </ul>
  </div>
</template>

<script setup lang="ts">
defineProps<{ content: { title?: string; desc?: string; button_text?: string; features?: string[] } }>();
</script>

<style scoped>
.preview-cta {
  padding: 16px;
  border-radius: 12px;
  background: #fff7ed;
  border: 1px solid rgba(249, 115, 22, 0.2);
}

.cta-actions {
  margin: 12px 0;
}

.cta-list {
  padding-left: 18px;
  margin: 0;
}
</style>
