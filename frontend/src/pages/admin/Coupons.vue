<template>
  <div class="coupon-page">
    <a-card :bordered="false" class="panel">
      <a-tabs v-model:activeKey="activeTab">
        <a-tab-pane key="groups" tab="商品组">
          <div class="toolbar">
            <a-button type="primary" @click="openGroup()">新增商品组</a-button>
          </div>
          <a-table :columns="groupColumns" :data-source="groups" row-key="id" :pagination="false">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'rules'">
                {{ (record.rules || []).length }} 条
              </template>
              <template v-else-if="column.key === 'rule_preview'">
                {{ renderRulePreview(record.rules || []) }}
              </template>
              <template v-else-if="column.key === 'actions'">
                <a-space>
                  <a-button type="link" @click="openGroup(record)">编辑</a-button>
                  <a-button type="link" danger @click="removeGroup(record)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-tab-pane>

        <a-tab-pane key="coupons" tab="优惠码">
          <div class="toolbar">
            <a-space>
              <a-button type="primary" @click="openCoupon()">新增优惠码</a-button>
              <a-button @click="batchOpen = true">批量生成</a-button>
            </a-space>
          </div>
          <a-table :columns="couponColumns" :data-source="coupons" row-key="id" :pagination="false">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'discount'">{{ ((record.discount_permille || 0) / 10).toFixed(1) }}%</template>
              <template v-else-if="column.key === 'actions'">
                <a-space>
                  <a-button type="link" @click="openCoupon(record)">编辑</a-button>
                  <a-button type="link" danger @click="removeCoupon(record)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-tab-pane>
      </a-tabs>
    </a-card>

    <a-modal v-model:open="groupOpen" :title="groupForm.id ? '编辑商品组' : '新增商品组'" width="980px" @ok="saveGroup">
      <a-form layout="vertical">
        <a-form-item label="名称">
          <a-input v-model:value="groupForm.name" />
        </a-form-item>

        <div class="rule-header">
          <span>商品策略</span>
          <a-button size="small" @click="addRule">新增策略</a-button>
        </div>

        <div v-for="(rule, idx) in groupForm.rules" :key="`rule-${idx}`" class="rule-card">
          <div class="rule-card-header">
            <span>策略 {{ idx + 1 }}</span>
            <a-button size="small" danger @click="removeRule(idx)" :disabled="groupForm.rules.length <= 1">删除</a-button>
          </div>
          <a-row :gutter="10">
            <a-col :span="8">
              <a-form-item label="范围">
                <a-select v-model:value="rule.scope" :options="scopeOptions" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="类型">
                <a-select
                  v-model:value="rule.goods_type_id"
                  allow-clear
                  :options="goodsTypeOptions"
                  :disabled="!needGoodsType(rule.scope)"
                />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="地区">
                <a-select
                  v-model:value="rule.region_id"
                  allow-clear
                  :options="regionOptions(rule)"
                  :disabled="!needRegion(rule.scope)"
                />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="线路">
                <a-select
                  v-model:value="rule.plan_group_id"
                  allow-clear
                  :options="planGroupOptions(rule)"
                  :disabled="!needPlanGroup(rule.scope)"
                />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="套餐">
                <a-select
                  v-model:value="rule.package_id"
                  allow-clear
                  :options="packageOptions(rule)"
                  :disabled="!needPackage(rule.scope)"
                />
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="10" v-if="rule.scope === 'addon_config'">
            <a-col :span="6"><a-form-item label="附加CPU"><a-switch v-model:checked="rule.addon_core_enabled" /></a-form-item></a-col>
            <a-col :span="6"><a-form-item label="附加内存"><a-switch v-model:checked="rule.addon_mem_enabled" /></a-form-item></a-col>
            <a-col :span="6"><a-form-item label="附加磁盘"><a-switch v-model:checked="rule.addon_disk_enabled" /></a-form-item></a-col>
            <a-col :span="6"><a-form-item label="附加带宽"><a-switch v-model:checked="rule.addon_bw_enabled" /></a-form-item></a-col>
          </a-row>
        </div>
      </a-form>
    </a-modal>

    <a-modal v-model:open="couponOpen" :title="couponForm.id ? '编辑优惠码' : '新增优惠码'" @ok="saveCoupon">
      <a-form layout="vertical">
        <a-row :gutter="10">
          <a-col :span="12"><a-form-item label="优惠码"><a-input v-model:value="couponForm.code" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="折扣(‰)"><a-input-number v-model:value="couponForm.discount_permille" style="width:100%" :min="1" :max="1000" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="商品组"><a-select v-model:value="couponForm.product_group_id" :options="groupOptions" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="总次数(-1无限)"><a-input-number v-model:value="couponForm.total_limit" style="width:100%" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="单用户(-1无限)"><a-input-number v-model:value="couponForm.per_user_limit" style="width:100%" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="仅新用户"><a-switch v-model:checked="couponForm.new_user_only" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="启用"><a-switch v-model:checked="couponForm.active" /></a-form-item></a-col>
        </a-row>
      </a-form>
    </a-modal>

    <a-modal v-model:open="batchOpen" title="批量生成优惠码" @ok="saveBatch">
      <a-form layout="vertical">
        <a-row :gutter="10">
          <a-col :span="8"><a-form-item label="前缀"><a-input v-model:value="batchForm.prefix" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="数量"><a-input-number v-model:value="batchForm.count" style="width:100%" :min="1" :max="1000" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="随机长度"><a-input-number v-model:value="batchForm.length" style="width:100%" :min="4" :max="16" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="折扣(‰)"><a-input-number v-model:value="batchForm.discount_permille" style="width:100%" :min="1" :max="1000" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="商品组"><a-select v-model:value="batchForm.product_group_id" :options="groupOptions" /></a-form-item></a-col>
        </a-row>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { message, Modal } from "ant-design-vue";
import {
  listCouponGroups,
  createCouponGroup,
  updateCouponGroup,
  deleteCouponGroup,
  listCoupons,
  createCoupon,
  updateCoupon,
  deleteCoupon,
  batchGenerateCoupons,
  listGoodsTypes,
  listRegions,
  listPlanGroups,
  listPackages
} from "@/services/admin";

const activeTab = ref("groups");
const groups = ref([]);
const coupons = ref([]);
const goodsTypes = ref([]);
const regions = ref([]);
const planGroups = ref([]);
const packages = ref([]);

const scopeOptions = [
  { label: "全部(不含附加)", value: "all" },
  { label: "全部附加项", value: "all_addons" },
  { label: "类型", value: "goods_type" },
  { label: "类型-地区", value: "goods_type_region" },
  { label: "类型-地区-线路", value: "plan_group" },
  { label: "类型-地区-线路-套餐", value: "package" },
  { label: "附加项配置(在线路下)", value: "addon_config" }
];

const groupColumns = [
  { title: "ID", dataIndex: "id", width: 80 },
  { title: "名称", dataIndex: "name", width: 220 },
  { title: "策略数", key: "rules", width: 100 },
  { title: "策略预览", key: "rule_preview" },
  { title: "操作", key: "actions", width: 150 }
];
const couponColumns = [
  { title: "ID", dataIndex: "id", width: 80 },
  { title: "优惠码", dataIndex: "code", width: 180 },
  { title: "折扣", key: "discount", width: 120 },
  { title: "商品组", dataIndex: "product_group_id", width: 120 },
  { title: "策略", key: "policy", customRender: ({ record }) => `总:${record.total_limit} 单:${record.per_user_limit}` },
  { title: "操作", key: "actions", width: 150 }
];

const groupOpen = ref(false);
const couponOpen = ref(false);
const batchOpen = ref(false);

const createEmptyRule = () => ({
  scope: "all",
  goods_type_id: undefined,
  region_id: undefined,
  plan_group_id: undefined,
  package_id: undefined,
  addon_core_enabled: false,
  addon_mem_enabled: false,
  addon_disk_enabled: false,
  addon_bw_enabled: false
});

const groupForm = reactive({
  id: null,
  name: "",
  rules: [createEmptyRule()]
});

const couponForm = reactive({
  id: null,
  code: "",
  discount_permille: 900,
  product_group_id: undefined,
  total_limit: -1,
  per_user_limit: -1,
  new_user_only: false,
  active: true,
  note: ""
});

const batchForm = reactive({
  prefix: "CP",
  count: 20,
  length: 8,
  discount_permille: 900,
  product_group_id: undefined,
  total_limit: -1,
  per_user_limit: -1,
  new_user_only: false,
  active: true,
  note: ""
});

const groupOptions = computed(() => (groups.value || []).map((g) => ({ label: `${g.name} (#${g.id})`, value: g.id })));
const goodsTypeOptions = computed(() => (goodsTypes.value || []).map((item) => ({ label: item.name, value: item.id })));

const needGoodsType = (scope) => ["goods_type", "goods_type_region", "plan_group", "package", "addon_config"].includes(scope);
const needRegion = (scope) => ["goods_type_region", "plan_group"].includes(scope);
const needPlanGroup = (scope) => ["plan_group", "package", "addon_config"].includes(scope);
const needPackage = (scope) => scope === "package";

const regionOptions = (rule) => {
  const gid = Number(rule.goods_type_id || 0);
  return (regions.value || [])
    .filter((item) => !gid || Number(item.goods_type_id) === gid)
    .map((item) => ({ label: item.name, value: item.id }));
};

const planGroupOptions = (rule) => {
  const gid = Number(rule.goods_type_id || 0);
  const rid = Number(rule.region_id || 0);
  return (planGroups.value || [])
    .filter((item) => (!gid || Number(item.goods_type_id) === gid) && (!rid || Number(item.region_id) === rid))
    .map((item) => ({ label: item.name, value: item.id }));
};

const packageOptions = (rule) => {
  const pid = Number(rule.plan_group_id || 0);
  return (packages.value || [])
    .filter((item) => !pid || Number(item.plan_group_id) === pid)
    .map((item) => ({ label: item.name, value: item.id }));
};

const scopeLabel = (scope) => scopeOptions.find((item) => item.value === scope)?.label || scope || "-";

const renderRulePreview = (rules) => {
  if (!rules.length) return "-";
  const first = rules[0];
  const text = scopeLabel(first.scope);
  if (rules.length === 1) return text;
  return `${text} 等 ${rules.length} 条`;
};

const normalizeRule = (rule) => {
  const out = { ...createEmptyRule(), ...(rule || {}) };
  if (!needGoodsType(out.scope)) out.goods_type_id = undefined;
  if (!needRegion(out.scope)) out.region_id = undefined;
  if (!needPlanGroup(out.scope)) out.plan_group_id = undefined;
  if (!needPackage(out.scope)) out.package_id = undefined;
  if (out.scope !== "addon_config") {
    out.addon_core_enabled = false;
    out.addon_mem_enabled = false;
    out.addon_disk_enabled = false;
    out.addon_bw_enabled = false;
  }
  return out;
};

const normalizeRules = (rules) => {
  const list = Array.isArray(rules) ? rules.map(normalizeRule) : [];
  return list.length ? list : [createEmptyRule()];
};

const validateRule = (rule) => {
  if (needGoodsType(rule.scope) && !Number(rule.goods_type_id || 0)) return "请选择类型";
  if (needRegion(rule.scope) && !Number(rule.region_id || 0)) return "请选择地区";
  if (needPlanGroup(rule.scope) && !Number(rule.plan_group_id || 0)) return "请选择线路";
  if (needPackage(rule.scope) && !Number(rule.package_id || 0)) return "请选择套餐";
  return "";
};

const fetchAll = async () => {
  const [g, c, gt, r, pg, p] = await Promise.all([
    listCouponGroups(),
    listCoupons({ limit: 200, offset: 0 }),
    listGoodsTypes(),
    listRegions({ limit: 1000, offset: 0 }),
    listPlanGroups({ limit: 2000, offset: 0 }),
    listPackages({ limit: 3000, offset: 0 })
  ]);
  groups.value = (g.data?.items || []).map((item) => ({ ...item, rules: normalizeRules(item.rules) }));
  coupons.value = c.data?.items || [];
  goodsTypes.value = gt.data?.items || [];
  regions.value = r.data?.items || [];
  planGroups.value = pg.data?.items || [];
  packages.value = p.data?.items || [];
};

const addRule = () => {
  groupForm.rules.push(createEmptyRule());
};

const removeRule = (idx) => {
  if (groupForm.rules.length <= 1) return;
  groupForm.rules.splice(idx, 1);
};

const openGroup = (record) => {
  if (!record) {
    Object.assign(groupForm, { id: null, name: "", rules: [createEmptyRule()] });
  } else {
    Object.assign(groupForm, {
      id: record.id,
      name: record.name || "",
      rules: normalizeRules(record.rules)
    });
  }
  groupOpen.value = true;
};

const saveGroup = async () => {
  const rules = normalizeRules(groupForm.rules);
  for (const rule of rules) {
    const errText = validateRule(rule);
    if (errText) {
      message.error(errText);
      return;
    }
  }
  const payload = {
    id: groupForm.id,
    name: (groupForm.name || "").trim(),
    rules
  };
  if (!payload.name) {
    message.error("请输入商品组名称");
    return;
  }
  if (groupForm.id) await updateCouponGroup(groupForm.id, payload);
  else await createCouponGroup(payload);
  groupOpen.value = false;
  message.success("保存成功");
  await fetchAll();
};

const removeGroup = (record) =>
  Modal.confirm({
    title: "确认删除商品组？",
    onOk: async () => {
      await deleteCouponGroup(record.id);
      message.success("删除成功");
      await fetchAll();
    }
  });

const openCoupon = (record) => {
  Object.assign(couponForm, record || {
    id: null,
    code: "",
    discount_permille: 900,
    product_group_id: undefined,
    total_limit: -1,
    per_user_limit: -1,
    new_user_only: false,
    active: true,
    note: ""
  });
  couponOpen.value = true;
};

const saveCoupon = async () => {
  const payload = { ...couponForm };
  if (couponForm.id) await updateCoupon(couponForm.id, payload);
  else await createCoupon(payload);
  couponOpen.value = false;
  message.success("保存成功");
  await fetchAll();
};

const removeCoupon = (record) =>
  Modal.confirm({
    title: "确认删除优惠码？",
    onOk: async () => {
      await deleteCoupon(record.id);
      message.success("删除成功");
      await fetchAll();
    }
  });

const saveBatch = async () => {
  await batchGenerateCoupons({ ...batchForm });
  batchOpen.value = false;
  message.success("批量生成完成");
  await fetchAll();
};

onMounted(fetchAll);
</script>

<style scoped>
.coupon-page { padding: 4px; }
.panel { border-radius: 12px; }
.toolbar { margin-bottom: 12px; display: flex; justify-content: flex-end; }
.rule-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  font-weight: 600;
}
.rule-card {
  border: 1px solid var(--border, #e5e7eb);
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 10px;
}
.rule-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
</style>
