<template>
  <div ref="el" style="width: 100%; height: 260px"></div>
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
        type: "line",
        smooth: true,
        data: props.data.values || [],
        areaStyle: { color: "rgba(22, 119, 255, 0.12)" },
        lineStyle: { color: "#1677ff" }
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
