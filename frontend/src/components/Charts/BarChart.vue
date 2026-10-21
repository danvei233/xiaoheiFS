<template>
  <div ref="el" style="height: 260px"></div>
</template>

<script setup>
import { onMounted, onBeforeUnmount, ref, watch } from "vue";
import { echarts } from "@/lib/echarts";

const props = defineProps({
  data: { type: Object, default: () => ({ labels: [], values: [] }) }
});

const el = ref(null);
let chart;

const render = () => {
  if (!el.value) return;
  chart = chart || echarts.init(el.value);
  chart.setOption({
    grid: { left: 30, right: 20, top: 20, bottom: 30 },
    xAxis: { type: "category", data: props.data.labels || [] },
    yAxis: { type: "value" },
    series: [
      {
        type: "bar",
        data: props.data.values || [],
        itemStyle: { color: "#1677ff" }
      }
    ]
  });
};

const resize = () => chart?.resize();

onMounted(() => {
  render();
  window.addEventListener("resize", resize);
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", resize);
  chart?.dispose();
});

watch(() => props.data, render, { deep: true });
</script>
