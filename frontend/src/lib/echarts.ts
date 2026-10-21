import * as echarts from "echarts/core";
import { BarChart, GaugeChart, LineChart, PieChart } from "echarts/charts";
import { DataZoomComponent, GridComponent, LegendComponent, TooltipComponent } from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";

echarts.use([LineChart, BarChart, PieChart, GaugeChart, GridComponent, TooltipComponent, LegendComponent, DataZoomComponent, CanvasRenderer]);

export { echarts };
