import { defineStore } from "pinia";
import { getCatalog } from "@/services/user";

export const useCatalogStore = defineStore("catalog", {
  state: () => ({
    goodsTypes: [],
    regions: [],
    lines: [],
    planGroups: [],
    packages: [],
    systemImages: [],
    billingCycles: [],
    loading: false
  }),
  actions: {
    async fetchCatalog() {
      this.loading = true;
      try {
        const res = await getCatalog();
        const data = res.data || {};
        const rawGoodsTypes = data.goods_types || [];
        const rawRegions = data.regions || [];
        const rawLines = data.lines || [];
        const rawGroups = data.plan_groups || [];
        const rawPackages = data.packages || [];
        const rawImages = data.system_images || [];
        const rawCycles = data.billing_cycles || [];

        this.goodsTypes = rawGoodsTypes.map((gt) => ({
          id: gt.id ?? gt.ID,
          code: gt.code ?? gt.Code,
          name: gt.name ?? gt.Name,
          active: gt.active ?? gt.Active,
          sort_order: gt.sort_order ?? gt.SortOrder,
          automation_category: gt.automation_category ?? gt.AutomationCategory,
          automation_plugin_id: gt.automation_plugin_id ?? gt.AutomationPluginID,
          automation_instance_id: gt.automation_instance_id ?? gt.AutomationInstanceID
        }));

        this.regions = rawRegions.map((region) => ({
          id: region.id ?? region.ID,
          goods_type_id: region.goods_type_id ?? region.GoodsTypeID,
          name: region.name ?? region.Name,
          code: region.code ?? region.Code,
          active: region.active ?? region.Active
        }));

        const lineSource = rawLines.length ? rawLines : rawGroups;
        this.lines = lineSource.map((line) => ({
          id: line.id ?? line.ID,
          goods_type_id: line.goods_type_id ?? line.GoodsTypeID,
          region_id: line.region_id ?? line.RegionID,
          name: line.name ?? line.Name ?? line.line_name ?? line.LineName,
          line_id: line.line_id ?? line.LineID,
          unit_core: line.unit_core ?? line.UnitCore,
          unit_mem: line.unit_mem ?? line.UnitMem,
          unit_disk: line.unit_disk ?? line.UnitDisk,
          unit_bw: line.unit_bw ?? line.UnitBW,
          add_core_min: line.add_core_min ?? line.AddCoreMin,
          add_core_max: line.add_core_max ?? line.AddCoreMax,
          add_core_step: line.add_core_step ?? line.AddCoreStep,
          add_mem_min: line.add_mem_min ?? line.AddMemMin,
          add_mem_max: line.add_mem_max ?? line.AddMemMax,
          add_mem_step: line.add_mem_step ?? line.AddMemStep,
          add_disk_min: line.add_disk_min ?? line.AddDiskMin,
          add_disk_max: line.add_disk_max ?? line.AddDiskMax,
          add_disk_step: line.add_disk_step ?? line.AddDiskStep,
          add_bw_min: line.add_bw_min ?? line.AddBwMin,
          add_bw_max: line.add_bw_max ?? line.AddBwMax,
          add_bw_step: line.add_bw_step ?? line.AddBwStep,
          active: line.active ?? line.Active,
          visible: line.visible ?? line.Visible,
          capacity_remaining: line.capacity_remaining ?? line.CapacityRemaining,
          sort_order: line.sort_order ?? line.SortOrder
        }));

        this.planGroups = rawGroups.map((group) => ({
          id: group.id ?? group.ID,
          goods_type_id: group.goods_type_id ?? group.GoodsTypeID,
          region_id: group.region_id ?? group.RegionID,
          line_id: group.line_id ?? group.LineID,
          name: group.name ?? group.Name ?? group.line_name ?? group.LineName,
          unit_core: group.unit_core ?? group.UnitCore,
          unit_mem: group.unit_mem ?? group.UnitMem,
          unit_disk: group.unit_disk ?? group.UnitDisk,
          unit_bw: group.unit_bw ?? group.UnitBW,
          active: group.active ?? group.Active,
          visible: group.visible ?? group.Visible,
          capacity_remaining: group.capacity_remaining ?? group.CapacityRemaining
        }));

        this.packages = rawPackages.map((pkg) => ({
          id: pkg.id ?? pkg.ID,
          goods_type_id: pkg.goods_type_id ?? pkg.GoodsTypeID,
          product_id: pkg.product_id ?? pkg.ProductID,
          plan_group_id: pkg.plan_group_id ?? pkg.PlanGroupID,
          name: pkg.name ?? pkg.Name,
          cores: pkg.cores ?? pkg.Cores,
          memory_gb: pkg.memory_gb ?? pkg.MemoryGB,
          disk_gb: pkg.disk_gb ?? pkg.DiskGB,
          bandwidth_mbps: pkg.bandwidth_mbps ?? pkg.BandwidthMB,
          cpu_model: pkg.cpu_model ?? pkg.CPUModel,
          monthly_price: pkg.monthly_price ?? pkg.Monthly,
          port_num: pkg.port_num ?? pkg.PortNum,
          active: pkg.active ?? pkg.Active,
          visible: pkg.visible ?? pkg.Visible,
          capacity_remaining: pkg.capacity_remaining ?? pkg.CapacityRemaining
        }));

        this.systemImages = rawImages.map((img) => ({
          id: img.id ?? img.ID,
          line_id: img.line_id ?? img.LineID,
          plan_group_id: img.plan_group_id ?? img.PlanGroupID,
          image_id: img.image_id ?? img.ImageID,
          name: img.name ?? img.Name,
          type: img.type ?? img.Type,
          enabled: img.enabled ?? img.Enabled
        }));

        this.billingCycles = rawCycles.map((cycle) => ({
          id: cycle.id ?? cycle.ID,
          name: cycle.name ?? cycle.Name,
          months: cycle.months ?? cycle.Months,
          multiplier: cycle.multiplier ?? cycle.Multiplier,
          min_qty: cycle.min_qty ?? cycle.MinQty,
          max_qty: cycle.max_qty ?? cycle.MaxQty,
          active: cycle.active ?? cycle.Active,
          sort_order: cycle.sort_order ?? cycle.SortOrder
        }));
      } finally {
        this.loading = false;
      }
    }
  }
});
