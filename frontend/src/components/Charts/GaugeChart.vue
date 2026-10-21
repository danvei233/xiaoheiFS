<template>
  <div ref="el" style="width: 100%; height: 200px"></div>
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from "vue";
import { echarts } from "@/lib/echarts";

const props = defineProps<{
  value: number;
  title?: string;
  color?: string;
  max?: number;
}>();

const el = ref<HTMLElement | null>(null);
let chart: any;

const getColor = (v: number) => {
  if (props.color) return props.color;
  if (v < 50) return "#52c41a";
  if (v < 80) return "#faad14";
  return "#f5222d";
};

const render = () => {
  if (!el.value) return;
  chart = chart || echarts.init(el.value);
  const val = Number(props.value) || 0;
  chart.setOption({
    series: [
      {
        type: "gauge",
        startAngle: 200,
        endAngle: -20,
        min: 0,
        max: props.max || 100,
        splitNumber: 5,
        itemStyle: {
          color: getColor(val)
        },
        progress: {
          show: true,
          width: 18
        },
        pointer: {
          show: true,
          width: 5,
          length: "60%"
        },
        axisLine: {
          lineStyle: {
            width: 18
          }
        },
        axisTick: {
          show: false
        },
        splitLine: {
          length: 6,
          lineStyle: {
            width: 2,
            color: "#999"
          }
        },
        axisLabel: {
          distance: 10,
          fontSize: 10,
          color: "#666"
        },
        title: {
          offsetCenter: [0, "30%"],
          fontSize: 12,
          color: "#666"
        },
        detail: {
          valueAnimation: true,
          fontSize: 24,
          offsetCenter: [0, "-10%"],
          formatter: "{value}%",
          color: "#333"
        },
        data: [
          {
            value: val.toFixed(1),
            name: props.title || ""
          }
        ]
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

watch(() => [props.value, props.title], render, { deep: true });
</script>
