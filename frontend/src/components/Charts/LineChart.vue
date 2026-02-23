<template>
  <div ref="el" :style="{ width: '100%', height: normalizedHeight }"></div>
</template>

<script setup>
import { computed, nextTick, onMounted, onBeforeUnmount, ref, watch } from "vue";
import { echarts } from "@/lib/echarts";

const props = defineProps({
  data: { type: Object, default: () => ({ labels: [], values: [] }) },
  color: { type: String, default: "#1677ff" },
  height: { type: [String, Number], default: 260 },
  yAxisValueFormatter: { type: Function, default: undefined },
  tooltipValueFormatter: { type: Function, default: undefined },
  smooth: { type: Boolean, default: true }
});

const el = ref(null);
let chart;
let resizeObserver;

const normalizedHeight = computed(() => {
  const raw = props.height;
  if (typeof raw === "number") return `${raw}px`;
  return raw || "260px";
});

const formatNumber = (value) => {
  const num = Number(value || 0);
  if (!Number.isFinite(num)) return "0";
  return num.toLocaleString("zh-CN", { maximumFractionDigits: 2 });
};

const formatYAxisValue = (value) => {
  if (typeof props.yAxisValueFormatter === "function") {
    return props.yAxisValueFormatter(value);
  }
  return formatNumber(value);
};

const formatTooltipValue = (value) => {
  if (typeof props.tooltipValueFormatter === "function") {
    return props.tooltipValueFormatter(value);
  }
  return formatNumber(value);
};

const render = () => {
  if (!el.value) return;
  chart = chart || echarts.init(el.value);
  const labels = Array.isArray(props.data?.labels) ? props.data.labels : [];
  const values = Array.isArray(props.data?.values) ? props.data.values : [];
  const pointCount = labels.length;
  chart.setOption({
    animationDuration: 350,
    animationDurationUpdate: 260,
    tooltip: {
      trigger: "axis",
      confine: true,
      borderWidth: 0,
      backgroundColor: "rgba(17, 24, 39, 0.92)",
      textStyle: { color: "#fff", fontSize: 12 },
      axisPointer: { type: "cross" },
      formatter: (params = []) => {
        const first = params[0];
        if (!first) return "";
        const title = first.axisValueLabel || first.axisValue || "";
        const line = `${first.marker || ""} ${first.seriesName || "数值"}: ${formatTooltipValue(first.value)}`;
        return `${title}<br/>${line}`;
      }
    },
    grid: { left: 46, right: 26, top: 20, bottom: pointCount > 18 ? 56 : 36 },
    xAxis: {
      type: "category",
      boundaryGap: false,
      data: labels,
      axisPointer: { show: true, snap: true },
      axisLabel: { color: "#6b7280", hideOverlap: true, interval: pointCount > 20 ? "auto" : 0 },
      axisLine: { lineStyle: { color: "#e5e7eb" } }
    },
    yAxis: {
      type: "value",
      axisLabel: {
        color: "#6b7280",
        formatter: (value) => formatYAxisValue(value)
      },
      axisPointer: {
        show: true,
        label: {
          show: true,
          formatter: ({ value }) => formatYAxisValue(value)
        }
      },
      splitLine: { lineStyle: { color: "#f1f5f9" } }
    },
    dataZoom: pointCount > 18 ? [{ type: "inside", zoomOnMouseWheel: true, moveOnMouseMove: true, moveOnMouseWheel: true }] : [],
    series: [
      {
        name: "数值",
        type: "line",
        smooth: props.smooth,
        showSymbol: pointCount <= 32,
        symbol: "circle",
        symbolSize: 6,
        data: values,
        areaStyle: {
          color: {
            type: "linear",
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: `${props.color}33` },
              { offset: 1, color: `${props.color}08` }
            ]
          }
        },
        lineStyle: { color: props.color, width: 2 },
        itemStyle: { color: props.color },
        emphasis: { focus: "series" }
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
  () => [props.data, props.color, props.height, props.smooth],
  async () => {
    await nextTick();
    render();
    resize();
  },
  { deep: true }
);
</script>
