<template>
  <div ref="el" style="width: 100%; height: 260px"></div>
</template>

<script setup>
import { nextTick, onMounted, onBeforeUnmount, ref, watch } from "vue";
import { echarts } from "@/lib/echarts";

const props = defineProps({
  data: { type: Array, default: () => [] }
});

const el = ref(null);
let chart;
let resizeObserver;

const render = () => {
  if (!el.value) return;
  chart = chart || echarts.init(el.value);
  chart.setOption({
    tooltip: { trigger: "item" },
    series: [
      {
        type: "pie",
        radius: ["40%", "70%"],
        data: props.data || [],
        label: { formatter: "{b}: {d}%" }
      }
    ]
  });
};

const resize = () => chart?.resize();

onMounted(() => {
  nextTick(() => {
    render();
    resize();
  });
  window.addEventListener("resize", resize);
  if (el.value && typeof ResizeObserver !== "undefined") {
    resizeObserver = new ResizeObserver(() => resize());
    resizeObserver.observe(el.value);
  }
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", resize);
  resizeObserver?.disconnect?.();
  chart?.dispose();
});

watch(
  () => props.data,
  async () => {
    await nextTick();
    render();
    resize();
  },
  { deep: true }
);
</script>
