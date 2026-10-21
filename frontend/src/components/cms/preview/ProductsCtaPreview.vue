<template>
  <div class="preview-cta">
    <h3>{{ content.title || 'CTA 标题' }}</h3>
    <p>{{ content.desc || 'CTA 描述' }}</p>
    <div class="cta-actions">
      <a-button type="primary">{{ content.contact_text || '按钮文案' }}</a-button>
      <a-tag>{{ content.email || '邮箱' }}</a-tag>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ content: { title?: string; desc?: string; contact_text?: string; email?: string } }>();
</script>

<style scoped>
.preview-cta {
  padding: 16px;
  border-radius: 12px;
  background: #f0f9ff;
  border: 1px solid rgba(14, 165, 233, 0.2);
}

.cta-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 8px;
}
</style>
