<template>
  <div ref="el" style="width: 100%; height: 160px"></div>
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from "vue";
import { echarts } from "@/lib/echarts";

const props = defineProps<{
  value: number;
  title?: string;
  subtitle?: string;
}>();

const el = ref<HTMLElement | null>(null);
let chart: any;

const getColor = (v: number) => {
  if (v < 60) return "#52c41a";
  if (v < 85) return "#faad14";
  return "#f5222d";
};

const render = () => {
  if (!el.value) return;
  chart = chart || echarts.init(el.value);
  const val = Math.min(100, Math.max(0, Number(props.value) || 0));
  chart.setOption({
    title: {
      text: `${val.toFixed(1)}%`,
      subtext: props.subtitle || "",
      left: "center",
      top: "35%",
      textStyle: {
        fontSize: 24,
        fontWeight: "bold",
        color: "#333"
      },
      subtextStyle: {
        fontSize: 12,
        color: "#999"
      }
    },
    series: [
      {
        type: "pie",
        radius: ["65%", "80%"],
        center: ["50%", "50%"],
        startAngle: 90,
        label: { show: false },
        data: [
          {
            value: val,
            name: "已用",
            itemStyle: {
              color: getColor(val),
              borderRadius: 4
            }
          },
          {
            value: 100 - val,
            name: "剩余",
            itemStyle: {
              color: "#f0f0f0"
            }
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

watch(() => [props.value, props.subtitle], render, { deep: true });
</script>
