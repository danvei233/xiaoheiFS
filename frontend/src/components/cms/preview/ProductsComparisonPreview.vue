<template>
  <div class="preview-comparison">
    <h3>{{ content.title || '对比表' }}</h3>
    <div class="comparison-row" v-for="(row, index) in content.rows || []" :key="index">
      <strong>{{ row.feature }}</strong>
      <span>{{ (row.values || []).join(' / ') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ content: { title?: string; rows?: Array<{ feature?: string; values?: string[] }> } }>();
</script>

<style scoped>
.preview-comparison {
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 10px;
  padding: 12px;
  background: #fff;
}

.comparison-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 0;
  border-bottom: 1px dashed rgba(0, 0, 0, 0.06);
  font-size: 12px;
}

.comparison-row:last-child {
  border-bottom: none;
}
</style>
