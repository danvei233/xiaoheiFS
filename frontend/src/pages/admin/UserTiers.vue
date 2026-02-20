<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">用户等级</div>
        <div class="subtle">管理用户组、优惠策略与自动审批条件</div>
      </div>
      <a-space>
        <a-button @click="rebuildAll">重建全部缓存</a-button>
        <a-button type="primary" @click="openCreateGroup">创建用户组</a-button>
      </a-space>
    </div>

    <a-card title="用户组">
      <a-table :columns="groupColumns" :data-source="groups" :pagination="false" row-key="id" :loading="loading.groups">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <a-space>
              <span class="dot" :style="{ backgroundColor: record.color || '#1677ff' }"></span>
              <span>{{ record.name }}</span>
            </a-space>
          </template>
          <template v-else-if="column.key === 'auto_approve_enabled'">
            <a-tag :color="record.auto_approve_enabled ? 'green' : 'default'">{{ record.auto_approve_enabled ? "开启" : "关闭" }}</a-tag>
          </template>
          <template v-else-if="column.key === 'is_default'">
            <a-tag :color="record.is_default ? 'blue' : 'default'">{{ record.is_default ? "默认组" : "-" }}</a-tag>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" @click="selectGroup(record)">管理规则</a-button>
              <a-button type="link" @click="openEditGroup(record)">编辑</a-button>
              <a-button type="link" @click="rebuildGroup(record)">重建缓存</a-button>
              <a-button type="link" danger :disabled="record.is_default" @click="removeGroup(record)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-card class="rule-card" :title="`规则配置${selectedGroup ? ` - ${selectedGroup.name}` : ''}`">
      <a-empty v-if="!selectedGroup" description="请选择一个用户组管理规则" />
      <template v-else>
        <a-tabs v-model:activeKey="activeTab">
          <a-tab-pane key="discount" tab="优惠策略">
            <div class="toolbar">
              <a-button type="primary" @click="openCreateDiscount">新增优惠规则</a-button>
            </div>
            <a-table
              :columns="discountColumns"
              :data-source="discountRules"
              :pagination="false"
              row-key="id"
              :loading="loading.discounts"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'discount_permille'">
                  {{ renderPermille(record.discount_permille) }}
                </template>
                <template v-else-if="column.key === 'action'">
                  <a-space>
                    <a-button type="link" @click="openEditDiscount(record)">编辑</a-button>
                    <a-button type="link" danger @click="removeDiscount(record)">删除</a-button>
                  </a-space>
                </template>
              </template>
            </a-table>
          </a-tab-pane>
          <a-tab-pane key="auto" tab="自动审批条件">
            <div class="toolbar">
              <a-button type="primary" @click="openCreateAuto">新增审批条件</a-button>
            </div>
            <a-table :columns="autoColumns" :data-source="autoRules" :pagination="false" row-key="id" :loading="loading.autoRules">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'action'">
                  <a-space>
                    <a-button type="link" @click="openEditAuto(record)">编辑</a-button>
                    <a-button type="link" danger @click="removeAuto(record)">删除</a-button>
                  </a-space>
                </template>
              </template>
            </a-table>
          </a-tab-pane>
        </a-tabs>
      </template>
    </a-card>

    <a-modal v-model:open="groupModal.open" :title="groupModal.editing ? '编辑用户组' : '创建用户组'" @ok="saveGroup">
      <a-form layout="vertical">
        <a-form-item label="名称">
          <a-input v-model:value="groupForm.name" />
        </a-form-item>
        <a-form-item label="颜色">
          <a-input v-model:value="groupForm.color" placeholder="#1677ff" />
        </a-form-item>
        <a-form-item label="图标">
          <a-select v-model:value="groupForm.icon">
            <a-select-option value="badge">badge</a-select-option>
            <a-select-option value="star">star</a-select-option>
            <a-select-option value="crown">crown</a-select-option>
            <a-select-option value="rocket">rocket</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="优先级">
          <a-input-number v-model:value="groupForm.priority" style="width: 100%" :min="1" />
        </a-form-item>
        <a-form-item label="自动审批开关">
          <a-switch v-model:checked="groupForm.auto_approve_enabled" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="discountModal.open" :title="discountModal.editing ? '编辑优惠规则' : '新增优惠规则'" width="720px" @ok="saveDiscount">
      <a-form layout="vertical">
        <a-form-item label="对象范围">
          <a-select v-model:value="discountForm.scope" @change="onDiscountScopeChange">
            <a-select-option value="all">全部(不含附加项)</a-select-option>
            <a-select-option value="all_addons">全部附加项</a-select-option>
            <a-select-option value="goods_type">类型</a-select-option>
            <a-select-option value="goods_type_region">类型-地区</a-select-option>
            <a-select-option value="goods_type_region_plan_group">类型-地区-线路</a-select-option>
            <a-select-option value="package">套餐</a-select-option>
            <a-select-option value="addon_config">附加项配置</a-select-option>
          </a-select>
        </a-form-item>
        <a-row v-if="needsTargetSelection" :gutter="12">
          <a-col :span="6" v-if="needGoodsType">
            <a-form-item label="类型">
              <a-select
                v-model:value="discountForm.goods_type_id"
                placeholder="请选择类型"
                :options="goodsTypeOptions"
                allow-clear
                show-search
                option-filter-prop="label"
                @change="onDiscountGoodsTypeChange"
              />
            </a-form-item>
          </a-col>
          <a-col :span="6" v-if="needRegion">
            <a-form-item label="地区">
              <a-select
                v-model:value="discountForm.region_id"
                placeholder="请选择地区"
                :options="regionOptions"
                allow-clear
                show-search
                option-filter-prop="label"
                :disabled="!discountForm.goods_type_id"
                @change="onDiscountRegionChange"
              />
            </a-form-item>
          </a-col>
          <a-col :span="6" v-if="needPlanGroup">
            <a-form-item label="线路">
              <a-select
                v-model:value="discountForm.plan_group_id"
                placeholder="请选择线路"
                :options="planGroupOptions"
                allow-clear
                show-search
                option-filter-prop="label"
                :disabled="!discountForm.goods_type_id"
                @change="onDiscountPlanGroupChange"
              />
            </a-form-item>
          </a-col>
          <a-col :span="6" v-if="needPackage">
            <a-form-item label="套餐">
              <a-select
                v-model:value="discountForm.package_id"
                placeholder="请选择套餐"
                :options="packageOptions"
                allow-clear
                show-search
                option-filter-prop="label"
                :disabled="!discountForm.plan_group_id"
              />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8"><a-form-item label="折扣(‰)"><a-input-number v-model:value="discountForm.discount_permille" style="width: 100%" :min="0" :max="1000" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="固定价格(分,套餐)"><a-input-number v-model:value="discountForm.fixed_price" style="width: 100%" :min="0" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="核心附加折扣(‰)"><a-input-number v-model:value="discountForm.add_core_permille" style="width: 100%" :min="0" :max="1000" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8"><a-form-item label="内存附加折扣(‰)"><a-input-number v-model:value="discountForm.add_mem_permille" style="width: 100%" :min="0" :max="1000" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="磁盘附加折扣(‰)"><a-input-number v-model:value="discountForm.add_disk_permille" style="width: 100%" :min="0" :max="1000" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="带宽附加折扣(‰)"><a-input-number v-model:value="discountForm.add_bw_permille" style="width: 100%" :min="0" :max="1000" /></a-form-item></a-col>
        </a-row>
      </a-form>
    </a-modal>

    <a-modal v-model:open="autoModal.open" :title="autoModal.editing ? '编辑自动审批条件' : '新增自动审批条件'" width="680px" @ok="saveAuto">
      <a-form layout="vertical">
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="时长(天,-1无限)"><a-input-number v-model:value="autoForm.duration_days" style="width: 100%" :min="-1" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="排序"><a-input-number v-model:value="autoForm.sort_order" style="width: 100%" :min="0" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="条件(JSON)">
          <a-textarea v-model:value="autoForm.conditions_json" :rows="7" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from "vue";
import { Modal, message } from "ant-design-vue";
import {
  createUserTierAutoRule,
  createUserTierDiscountRule,
  createUserTierGroup,
  deleteUserTierAutoRule,
  deleteUserTierDiscountRule,
  deleteUserTierGroup,
  listUserTierAutoRules,
  listUserTierDiscountRules,
  listUserTierGroups,
  listGoodsTypes,
  listRegions,
  listPlanGroups,
  listPackages,
  rebuildUserTierCaches,
  updateUserTierAutoRule,
  updateUserTierDiscountRule,
  updateUserTierGroup
} from "@/services/admin";

const loading = reactive({
  groups: false,
  discounts: false,
  autoRules: false
});
const groups = ref([]);
const selectedGroup = ref(null);
const activeTab = ref("discount");
const discountRules = ref([]);
const autoRules = ref([]);
const goodsTypes = ref([]);
const regions = ref([]);
const planGroups = ref([]);
const packages = ref([]);

const groupColumns = [
  { title: "名称", key: "name" },
  { title: "图标", dataIndex: "icon", key: "icon", width: 90 },
  { title: "优先级", dataIndex: "priority", key: "priority", width: 90 },
  { title: "自动审批", key: "auto_approve_enabled", width: 100 },
  { title: "默认组", key: "is_default", width: 90 },
  { title: "操作", key: "action", width: 280 }
];

const discountColumns = [
  { title: "范围", dataIndex: "scope", key: "scope" },
  { title: "折扣", dataIndex: "discount_permille", key: "discount_permille", width: 110 },
  { title: "固定价(分)", dataIndex: "fixed_price", key: "fixed_price", width: 120 },
  { title: "类型/地区/线路/套餐", key: "obj", customRender: ({ record }) => renderRuleTarget(record) },
  { title: "操作", key: "action", width: 140 }
];

const autoColumns = [
  { title: "排序", dataIndex: "sort_order", key: "sort_order", width: 90 },
  { title: "时长(天)", dataIndex: "duration_days", key: "duration_days", width: 110 },
  { title: "条件(JSON)", dataIndex: "conditions_json", key: "conditions_json" },
  { title: "操作", key: "action", width: 140 }
];

const groupModal = reactive({ open: false, editing: null });
const discountModal = reactive({ open: false, editing: null });
const autoModal = reactive({ open: false, editing: null });

const groupForm = reactive({ name: "", color: "#1677ff", icon: "badge", priority: 10, auto_approve_enabled: true });
const discountForm = reactive({
  scope: "all",
  goods_type_id: 0,
  region_id: 0,
  plan_group_id: 0,
  package_id: 0,
  discount_permille: 1000,
  fixed_price: null,
  add_core_permille: 1000,
  add_mem_permille: 1000,
  add_disk_permille: 1000,
  add_bw_permille: 1000
});
const autoForm = reactive({ duration_days: -1, sort_order: 10, conditions_json: "[]" });

const renderPermille = (permille) => `${Number(permille || 0) / 10}%`;
const readItems = (res) => res?.data?.items || res?.items || [];
const toNumber = (v) => {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
};
const normalizeGoodsType = (row) => ({
  id: Number(row?.id ?? row?.ID ?? 0) || 0,
  code: row?.code ?? row?.Code ?? "",
  name: row?.name ?? row?.Name ?? ""
});
const normalizeRegion = (row) => ({
  id: Number(row?.id ?? row?.ID ?? 0) || 0,
  goods_type_id: Number(row?.goods_type_id ?? row?.GoodsTypeID ?? 0) || 0,
  name: row?.name ?? row?.Name ?? ""
});
const normalizePlanGroup = (row) => ({
  id: Number(row?.id ?? row?.ID ?? 0) || 0,
  goods_type_id: Number(row?.goods_type_id ?? row?.GoodsTypeID ?? 0) || 0,
  region_id: Number(row?.region_id ?? row?.RegionID ?? 0) || 0,
  name: row?.name ?? row?.Name ?? ""
});
const normalizePackage = (row) => ({
  id: Number(row?.id ?? row?.ID ?? 0) || 0,
  goods_type_id: Number(row?.goods_type_id ?? row?.GoodsTypeID ?? 0) || 0,
  plan_group_id: Number(row?.plan_group_id ?? row?.PlanGroupID ?? 0) || 0,
  name: row?.name ?? row?.Name ?? ""
});

const goodsTypeNameMap = computed(() => {
  const map = new Map();
  goodsTypes.value.forEach((item) => map.set(Number(item.id), item.name || `#${item.id}`));
  return map;
});
const regionNameMap = computed(() => {
  const map = new Map();
  regions.value.forEach((item) => map.set(Number(item.id), item.name || `#${item.id}`));
  return map;
});
const planGroupNameMap = computed(() => {
  const map = new Map();
  planGroups.value.forEach((item) => map.set(Number(item.id), item.name || `#${item.id}`));
  return map;
});
const packageNameMap = computed(() => {
  const map = new Map();
  packages.value.forEach((item) => map.set(Number(item.id), item.name || `#${item.id}`));
  return map;
});

const goodsTypeOptions = computed(() =>
  goodsTypes.value.map((item) => ({ label: item.name || `#${item.id}`, value: Number(item.id) }))
);
const regionOptions = computed(() => {
  const goodsTypeID = toNumber(discountForm.goods_type_id);
  return regions.value
    .filter((item) => !goodsTypeID || Number(item.goods_type_id) === goodsTypeID)
    .map((item) => ({ label: item.name || `#${item.id}`, value: Number(item.id) }));
});
const planGroupOptions = computed(() => {
  const goodsTypeID = toNumber(discountForm.goods_type_id);
  const regionID = toNumber(discountForm.region_id);
  return planGroups.value
    .filter((item) => !goodsTypeID || Number(item.goods_type_id) === goodsTypeID)
    .filter((item) => !regionID || Number(item.region_id) === regionID)
    .map((item) => ({ label: item.name || `#${item.id}`, value: Number(item.id) }));
});
const packageOptions = computed(() => {
  const goodsTypeID = toNumber(discountForm.goods_type_id);
  const planGroupID = toNumber(discountForm.plan_group_id);
  return packages.value
    .filter((item) => !goodsTypeID || Number(item.goods_type_id) === goodsTypeID)
    .filter((item) => !planGroupID || Number(item.plan_group_id) === planGroupID)
    .map((item) => ({ label: item.name || `#${item.id}`, value: Number(item.id) }));
});

const needGoodsType = computed(() =>
  ["goods_type", "goods_type_region", "goods_type_region_plan_group", "addon_config", "package"].includes(discountForm.scope)
);
const needRegion = computed(() =>
  ["goods_type_region", "goods_type_region_plan_group", "addon_config", "package"].includes(discountForm.scope)
);
const needPlanGroup = computed(() => ["goods_type_region_plan_group", "addon_config", "package"].includes(discountForm.scope));
const needPackage = computed(() => discountForm.scope === "package");
const needsTargetSelection = computed(() => needGoodsType.value || needRegion.value || needPlanGroup.value || needPackage.value);

const renderRuleTarget = (record) => {
  const gtID = Number(record.goods_type_id || 0);
  const regionID = Number(record.region_id || 0);
  const planID = Number(record.plan_group_id || 0);
  const pkgID = Number(record.package_id || 0);
  const gt = gtID > 0 ? goodsTypeNameMap.value.get(gtID) || `#${gtID}` : "-";
  const region = regionID > 0 ? regionNameMap.value.get(regionID) || `#${regionID}` : "-";
  const plan = planID > 0 ? planGroupNameMap.value.get(planID) || `#${planID}` : "-";
  const pkg = pkgID > 0 ? packageNameMap.value.get(pkgID) || `#${pkgID}` : "-";
  return `${gt}/${region}/${plan}/${pkg}`;
};

const loadGoodsTypes = async () => {
  const res = await listGoodsTypes();
  goodsTypes.value = readItems(res).map(normalizeGoodsType).filter((item) => item.id > 0);
};
const loadRegions = async (goodsTypeId) => {
  const params = goodsTypeId ? { goods_type_id: goodsTypeId } : undefined;
  const res = await listRegions(params);
  regions.value = readItems(res).map(normalizeRegion).filter((item) => item.id > 0);
};
const loadPlanGroups = async (goodsTypeId, regionId) => {
  const params = goodsTypeId ? { goods_type_id: goodsTypeId } : undefined;
  const res = await listPlanGroups(params);
  const items = readItems(res).map(normalizePlanGroup).filter((item) => item.id > 0);
  planGroups.value = regionId ? items.filter((it) => Number(it.region_id) === Number(regionId)) : items;
};
const loadPackages = async (goodsTypeId, planGroupId) => {
  const params = {};
  if (goodsTypeId) {
    params.goods_type_id = goodsTypeId;
  }
  if (planGroupId) {
    params.plan_group_id = planGroupId;
  }
  const res = await listPackages(Object.keys(params).length ? params : undefined);
  packages.value = readItems(res).map(normalizePackage).filter((item) => item.id > 0);
};

const normalizeDiscountTargetByScope = () => {
  if (!needGoodsType.value) {
    discountForm.goods_type_id = 0;
  }
  if (!needRegion.value) {
    discountForm.region_id = 0;
  }
  if (!needPlanGroup.value) {
    discountForm.plan_group_id = 0;
  }
  if (!needPackage.value) {
    discountForm.package_id = 0;
  }
};

const onDiscountScopeChange = async () => {
  normalizeDiscountTargetByScope();
  if (needGoodsType.value && goodsTypes.value.length === 0) {
    await loadGoodsTypes();
  }
  if (needRegion.value) {
    await loadRegions(discountForm.goods_type_id);
  } else {
    regions.value = [];
  }
  if (needPlanGroup.value) {
    await loadPlanGroups(discountForm.goods_type_id, discountForm.region_id);
  } else {
    planGroups.value = [];
  }
  if (needPackage.value) {
    await loadPackages(discountForm.goods_type_id, discountForm.plan_group_id);
  } else {
    packages.value = [];
  }
};

const onDiscountGoodsTypeChange = async () => {
  discountForm.region_id = 0;
  discountForm.plan_group_id = 0;
  discountForm.package_id = 0;
  await Promise.all([
    needRegion.value ? loadRegions(discountForm.goods_type_id) : Promise.resolve()
  ]);
  planGroups.value = [];
  packages.value = [];
};

const onDiscountRegionChange = async () => {
  discountForm.plan_group_id = 0;
  discountForm.package_id = 0;
  if (needPlanGroup.value) {
    await loadPlanGroups(discountForm.goods_type_id, discountForm.region_id);
  }
  packages.value = [];
};

const onDiscountPlanGroupChange = async () => {
  discountForm.package_id = 0;
  if (needPackage.value) {
    await loadPackages(discountForm.goods_type_id, discountForm.plan_group_id);
  }
};

const fetchGroups = async () => {
  loading.groups = true;
  try {
    const res = await listUserTierGroups();
    groups.value = res.data?.items || [];
    if (!selectedGroup.value && groups.value.length) {
      selectGroup(groups.value[0]);
    } else if (selectedGroup.value) {
      const next = groups.value.find((g) => g.id === selectedGroup.value.id);
      selectedGroup.value = next || null;
    }
  } finally {
    loading.groups = false;
  }
};

const fetchDiscounts = async () => {
  if (!selectedGroup.value?.id) return;
  loading.discounts = true;
  try {
    const res = await listUserTierDiscountRules(selectedGroup.value.id);
    discountRules.value = res.data?.items || [];
  } finally {
    loading.discounts = false;
  }
};

const fetchAutoRules = async () => {
  if (!selectedGroup.value?.id) return;
  loading.autoRules = true;
  try {
    const res = await listUserTierAutoRules(selectedGroup.value.id);
    autoRules.value = res.data?.items || [];
  } finally {
    loading.autoRules = false;
  }
};

const selectGroup = async (group) => {
  selectedGroup.value = group;
  await Promise.all([fetchDiscounts(), fetchAutoRules()]);
};

const openCreateGroup = () => {
  Object.assign(groupForm, { name: "", color: "#1677ff", icon: "badge", priority: 10, auto_approve_enabled: true });
  groupModal.editing = null;
  groupModal.open = true;
};

const openEditGroup = (record) => {
  Object.assign(groupForm, {
    name: record.name || "",
    color: record.color || "#1677ff",
    icon: record.icon || "badge",
    priority: record.priority || 10,
    auto_approve_enabled: !!record.auto_approve_enabled
  });
  groupModal.editing = record;
  groupModal.open = true;
};

const saveGroup = async () => {
  try {
    if (groupModal.editing?.id) {
      await updateUserTierGroup(groupModal.editing.id, { ...groupForm });
      message.success("用户组已更新");
    } else {
      await createUserTierGroup({ ...groupForm });
      message.success("用户组已创建");
    }
    groupModal.open = false;
    await fetchGroups();
  } catch (e) {
    message.error(e.response?.data?.error || "保存失败");
  }
};

const removeGroup = (record) => {
  Modal.confirm({
    title: "确认删除",
    content: `确定删除用户组 ${record.name} 吗？`,
    onOk: async () => {
      try {
        await deleteUserTierGroup(record.id);
        message.success("已删除");
        await fetchGroups();
      } catch (e) {
        message.error(e.response?.data?.error || "删除失败");
      }
    }
  });
};

const rebuildAll = async () => {
  await rebuildUserTierCaches();
  message.success("已触发全量缓存重建");
};

const rebuildGroup = async (record) => {
  await rebuildUserTierCaches(record.id);
  message.success("已触发分组缓存重建");
};

const openCreateDiscount = () => {
  Object.assign(discountForm, {
    scope: "all",
    goods_type_id: 0,
    region_id: 0,
    plan_group_id: 0,
    package_id: 0,
    discount_permille: 1000,
    fixed_price: null,
    add_core_permille: 1000,
    add_mem_permille: 1000,
    add_disk_permille: 1000,
    add_bw_permille: 1000
  });
  regions.value = [];
  planGroups.value = [];
  packages.value = [];
  discountModal.editing = null;
  discountModal.open = true;
};

const openEditDiscount = async (record) => {
  Object.assign(discountForm, {
    scope: record.scope || "all",
    goods_type_id: record.goods_type_id || 0,
    region_id: record.region_id || 0,
    plan_group_id: record.plan_group_id || 0,
    package_id: record.package_id || 0,
    discount_permille: record.discount_permille ?? 1000,
    fixed_price: record.fixed_price ?? null,
    add_core_permille: record.add_core_permille ?? 1000,
    add_mem_permille: record.add_mem_permille ?? 1000,
    add_disk_permille: record.add_disk_permille ?? 1000,
    add_bw_permille: record.add_bw_permille ?? 1000
  });
  await onDiscountScopeChange();
  if (needPlanGroup.value) {
    await loadPlanGroups(discountForm.goods_type_id, discountForm.region_id);
  }
  if (needPackage.value) {
    await loadPackages(discountForm.goods_type_id, discountForm.plan_group_id);
  }
  discountModal.editing = record;
  discountModal.open = true;
};

const saveDiscount = async () => {
  try {
    if (!selectedGroup.value?.id) return;
    if (needGoodsType.value && !toNumber(discountForm.goods_type_id)) {
      message.error("请选择类型");
      return;
    }
    if (needRegion.value && !toNumber(discountForm.region_id)) {
      message.error("请选择地区");
      return;
    }
    if (needPlanGroup.value && !toNumber(discountForm.plan_group_id)) {
      message.error("请选择线路");
      return;
    }
    if (needPackage.value && !toNumber(discountForm.package_id)) {
      message.error("请选择套餐");
      return;
    }
    normalizeDiscountTargetByScope();
    if (discountModal.editing?.id) {
      await updateUserTierDiscountRule(selectedGroup.value.id, discountModal.editing.id, { ...discountForm });
      message.success("优惠规则已更新");
    } else {
      await createUserTierDiscountRule(selectedGroup.value.id, { ...discountForm });
      message.success("优惠规则已创建");
    }
    discountModal.open = false;
    await fetchDiscounts();
  } catch (e) {
    message.error(e.response?.data?.error || "保存失败");
  }
};

const removeDiscount = (record) => {
  Modal.confirm({
    title: "确认删除",
    content: "确定删除该优惠规则吗？",
    onOk: async () => {
      try {
        await deleteUserTierDiscountRule(selectedGroup.value.id, record.id);
        message.success("已删除");
        await fetchDiscounts();
      } catch (e) {
        message.error(e.response?.data?.error || "删除失败");
      }
    }
  });
};

const openCreateAuto = () => {
  Object.assign(autoForm, { duration_days: -1, sort_order: 10, conditions_json: "[]" });
  autoModal.editing = null;
  autoModal.open = true;
};

const openEditAuto = (record) => {
  Object.assign(autoForm, {
    duration_days: record.duration_days ?? -1,
    sort_order: record.sort_order ?? 10,
    conditions_json: record.conditions_json || "[]"
  });
  autoModal.editing = record;
  autoModal.open = true;
};

const saveAuto = async () => {
  try {
    if (!selectedGroup.value?.id) return;
    if (autoModal.editing?.id) {
      await updateUserTierAutoRule(selectedGroup.value.id, autoModal.editing.id, { ...autoForm });
      message.success("审批规则已更新");
    } else {
      await createUserTierAutoRule(selectedGroup.value.id, { ...autoForm });
      message.success("审批规则已创建");
    }
    autoModal.open = false;
    await fetchAutoRules();
  } catch (e) {
    message.error(e.response?.data?.error || "保存失败");
  }
};

const removeAuto = (record) => {
  Modal.confirm({
    title: "确认删除",
    content: "确定删除该审批规则吗？",
    onOk: async () => {
      try {
        await deleteUserTierAutoRule(selectedGroup.value.id, record.id);
        message.success("已删除");
        await fetchAutoRules();
      } catch (e) {
        message.error(e.response?.data?.error || "删除失败");
      }
    }
  });
};

fetchGroups();
loadGoodsTypes();
</script>

<style scoped>
.rule-card {
  margin-top: 16px;
}

.toolbar {
  margin-bottom: 12px;
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 999px;
  display: inline-block;
}
</style>
