<template>
  <a-row :gutter="16">
    <a-col v-for="item in items" :key="item.title" :xs="24" :md="6">
      <a-card class="card" :bordered="false">
        <a-statistic :title="item.title" :value="item.value" />
      </a-card>
    </a-col>
  </a-row>
</template>

<script setup>
defineProps({
  items: { type: Array, default: () => [] }
});
</script>
