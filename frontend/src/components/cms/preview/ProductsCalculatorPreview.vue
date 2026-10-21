<template>
  <div class="preview-calculator">
    <h3>{{ content.title || '智能推荐' }}</h3>
    <p>{{ content.desc || '说明文字' }}</p>
    <div class="scenario-list">
      <div class="scenario" v-for="(item, index) in content.scenarios || []" :key="index">
        <div class="scenario-name">{{ item.icon }} {{ item.name }}</div>
        <div class="scenario-rec">{{ item.recommended }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ content: { title?: string; desc?: string; scenarios?: Array<{ icon?: string; name?: string; recommended?: string }> } }>();
</script>

<style scoped>
.preview-calculator {
  padding: 16px;
  border-radius: 12px;
  border: 1px solid rgba(0, 0, 0, 0.06);
  background: #fff;
}

.scenario-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 10px;
  margin-top: 12px;
}

.scenario {
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 8px;
  padding: 8px 10px;
  background: #f8fafc;
}

.scenario-name {
  font-weight: 600;
}

.scenario-rec {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.6);
}
</style>
