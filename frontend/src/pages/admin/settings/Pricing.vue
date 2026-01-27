<template>
  <div class="pricing-settings-page">
    <div class="page-header">
      <h1 class="page-title">价格与退款</h1>
      <a-button type="primary" @click="handleSave" :loading="saving">保存修改</a-button>
    </div>

    <a-row :gutter="16">
      <a-col :span="12">
        <a-card title="退款时间规则" :bordered="false">
          <a-form :model="form" layout="vertical">
            <a-form-item label="全额退款（天）">
              <a-input-number v-model:value="form.refund_full_days" :min="0" style="width: 100%" addon-after="天" />
              <div class="form-tip">按“天”生效；若下方“全额退款（小时）”大于 0，则优先生效。</div>
            </a-form-item>

            <a-form-item label="按比例退款（天）">
              <a-input-number
                v-model:value="form.refund_prorate_days"
                :min="0"
                style="width: 100%"
                addon-after="天"
              />
              <div class="form-tip">按“天”线性递减；若下方“按比例退款（小时）”大于 0，则优先生效。</div>
            </a-form-item>

            <a-form-item label="不再退款（天）">
              <a-input-number
                v-model:value="form.refund_no_refund_days"
                :min="0"
                style="width: 100%"
                addon-after="天"
              />
              <div class="form-tip">超过该天数不再允许退款；若下方“不再退款（小时）”大于 0，则优先生效。</div>
            </a-form-item>

            <a-divider style="margin: 12px 0" />

            <a-form-item label="全额退款（小时）">
              <a-input-number v-model:value="form.refund_full_hours" :min="0" style="width: 100%" addon-after="小时" />
              <div class="form-tip">可选：大于 0 时覆盖“全额退款（天）”。</div>
            </a-form-item>

            <a-form-item label="按比例退款（小时）">
              <a-input-number
                v-model:value="form.refund_prorate_hours"
                :min="0"
                style="width: 100%"
                addon-after="小时"
              />
              <div class="form-tip">可选：大于 0 时覆盖“按比例退款（天）”。</div>
            </a-form-item>

            <a-form-item label="不再退款（小时）">
              <a-input-number
                v-model:value="form.refund_no_refund_hours"
                :min="0"
                style="width: 100%"
                addon-after="小时"
              />
              <div class="form-tip">可选：大于 0 时覆盖“不再退款（天）”。</div>
            </a-form-item>

            <a-divider style="margin: 12px 0" />

            <a-form-item label="退款需要审核">
              <a-switch v-model:checked="form.refund_requires_approval" />
              <span style="margin-left: 8px">开启后，用户退款申请进入待审核</span>
            </a-form-item>

            <a-form-item label="管理员删除实例时自动退款">
              <a-switch v-model:checked="form.refund_on_admin_delete" />
              <span style="margin-left: 8px">开启后，管理员删除实例会触发自动退款（按当前规则计算）</span>
            </a-form-item>
          </a-form>
        </a-card>
      </a-col>

      <a-col :span="12">
        <a-card title="退款曲线（可选）" :bordered="false">
          <p class="form-tip">
            若配置退款曲线，将优先按曲线计算（覆盖左侧天/小时规则）。曲线的 X 轴为“已使用周期百分比”
            <code>(0-100)</code>，Y 轴为退款系数 <code>(0-1)</code>，后端按点之间线性插值。
          </p>
          <a-textarea
            v-model:value="form.refund_curve_json"
            :rows="14"
            placeholder='[
  { "percent": 0, "ratio": 1 },
  { "percent": 10, "ratio": 0.9 },
  { "percent": 100, "ratio": 0 }
]'
          />
          <div class="form-tip">字段说明：<code>percent</code>（已使用百分比 0-100），<code>ratio</code>（退款系数 0-1）。</div>
        </a-card>
      </a-col>
    </a-row>

    <a-card title="扩容/缩配定价（可选）" :bordered="false" style="margin-top: 16px">
      <a-form :model="form" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="扩容退款系数（0-1）">
              <a-input-number
                v-model:value="form.resize_refund_ratio"
                :min="0"
                :max="1"
                :step="0.01"
                style="width: 100%"
              />
              <div class="form-tip">当缩配产生退款时，按该系数折算入钱包金额。</div>
            </a-form-item>
          </a-col>

          <a-col :span="8">
            <a-form-item label="最小补差价（元）">
              <a-input-number v-model:value="form.resize_min_charge" :min="0" :step="0.01" style="width: 100%" />
              <div class="form-tip">小于该金额的补差价将提升到该值。</div>
            </a-form-item>
          </a-col>

          <a-col :span="8">
            <a-form-item label="最小退款（元）">
              <a-input-number v-model:value="form.resize_min_refund" :min="0" :step="0.01" style="width: 100%" />
              <div class="form-tip">小于该金额的退款将被视为 0。</div>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="计算方式（预留）">
              <a-select v-model:value="form.resize_price_mode" style="width: 100%">
                <a-select-option value="remaining">按剩余周期比例</a-select-option>
              </a-select>
              <div class="form-tip">当前后端仅实现“按剩余周期比例”。</div>
            </a-form-item>
          </a-col>

          <a-col :span="8">
            <a-form-item label="舍入方式">
              <a-select v-model:value="form.resize_rounding" style="width: 100%">
                <a-select-option value="round">四舍五入（默认）</a-select-option>
                <a-select-option value="ceil">向上取整</a-select-option>
                <a-select-option value="floor">向下取整</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>

          <a-col :span="8">
            <a-form-item label="退款入钱包">
              <a-switch v-model:checked="form.resize_refund_to_wallet" />
              <span style="margin-left: 8px">缩配退款自动入钱包</span>
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { message } from "ant-design-vue";
import { listSettings, updateSetting } from "@/services/admin";

const saving = ref(false);

const form = reactive({
  refund_full_days: 1,
  refund_prorate_days: 7,
  refund_no_refund_days: 30,
  refund_full_hours: 0,
  refund_prorate_hours: 0,
  refund_no_refund_hours: 0,
  refund_requires_approval: true,
  refund_on_admin_delete: true,
  refund_curve_json: "[]",
  resize_price_mode: "remaining",
  resize_refund_ratio: 1,
  resize_rounding: "round",
  resize_min_charge: 0,
  resize_min_refund: 0,
  resize_refund_to_wallet: true,
});

const toInt = (raw: string | undefined, fallback: number) => {
  const val = parseInt(String(raw ?? ""), 10);
  return Number.isFinite(val) ? val : fallback;
};

const toFloat = (raw: string | undefined, fallback: number) => {
  const val = parseFloat(String(raw ?? ""));
  return Number.isFinite(val) ? val : fallback;
};

const normalizeCurveJSON = (raw: string, label: string) => {
  const text = String(raw ?? "").trim();
  if (text === "") return "[]";
  let parsed: any;
  try {
    parsed = JSON.parse(text);
  } catch {
    throw new Error(`${label} 不是合法 JSON`);
  }
  if (!Array.isArray(parsed)) {
    throw new Error(`${label} 必须是数组`);
  }
  const normalized = parsed.map((item, idx) => {
    const percent = Number(item?.percent ?? item?.hours);
    const ratio = Number(item?.ratio);
    if (!Number.isFinite(percent) || percent < 0 || percent > 100) {
      throw new Error(`${label} 第 ${idx + 1} 项 percent 必须在 0-100 之间`);
    }
    if (!Number.isFinite(ratio) || ratio < 0 || ratio > 1) {
      throw new Error(`${label} 第 ${idx + 1} 项 ratio 必须在 0-1 之间`);
    }
    return { percent: Math.round(percent), ratio };
  });
  normalized.sort((a, b) => a.percent - b.percent);
  return JSON.stringify(normalized, null, 2);
};

const fetchData = async () => {
  const res = await listSettings();
  const items = res.data?.items || [];
  const data: Record<string, string> = {};
  items.forEach((item: any) => {
    data[item.key] = item.value;
  });

  form.refund_full_days = toInt(data.refund_full_days, 1);
  form.refund_prorate_days = toInt(data.refund_prorate_days, 7);
  form.refund_no_refund_days = toInt(data.refund_no_refund_days, 30);
  form.refund_full_hours = toInt(data.refund_full_hours, 0);
  form.refund_prorate_hours = toInt(data.refund_prorate_hours, 0);
  form.refund_no_refund_hours = toInt(data.refund_no_refund_hours, 0);
  form.refund_requires_approval = data.refund_requires_approval === "true";
  form.refund_on_admin_delete = data.refund_on_admin_delete === "true";
  form.refund_curve_json = data.refund_curve_json?.trim() || "[]";

  form.resize_price_mode = (data.resize_price_mode || "remaining").trim() || "remaining";
  form.resize_refund_ratio = toFloat(data.resize_refund_ratio, 1);
  form.resize_rounding = (data.resize_rounding || "round").trim() || "round";
  form.resize_min_charge = toFloat(data.resize_min_charge, 0);
  form.resize_min_refund = toFloat(data.resize_min_refund, 0);
  form.resize_refund_to_wallet = data.resize_refund_to_wallet !== "false";
};

const handleSave = async () => {
  saving.value = true;
  try {
    form.refund_curve_json = normalizeCurveJSON(form.refund_curve_json, "退款曲线");

    const items = [
      { key: "refund_full_days", value: String(form.refund_full_days) },
      { key: "refund_prorate_days", value: String(form.refund_prorate_days) },
      { key: "refund_no_refund_days", value: String(form.refund_no_refund_days) },
      { key: "refund_full_hours", value: String(form.refund_full_hours) },
      { key: "refund_prorate_hours", value: String(form.refund_prorate_hours) },
      { key: "refund_no_refund_hours", value: String(form.refund_no_refund_hours) },
      { key: "refund_requires_approval", value: String(form.refund_requires_approval) },
      { key: "refund_on_admin_delete", value: String(form.refund_on_admin_delete) },
      { key: "refund_curve_json", value: form.refund_curve_json },
      { key: "resize_price_mode", value: String(form.resize_price_mode) },
      { key: "resize_refund_ratio", value: String(form.resize_refund_ratio) },
      { key: "resize_rounding", value: String(form.resize_rounding) },
      { key: "resize_min_charge", value: String(form.resize_min_charge) },
      { key: "resize_min_refund", value: String(form.resize_min_refund) },
      { key: "resize_refund_to_wallet", value: String(form.resize_refund_to_wallet) },
    ];
    await updateSetting({ items });
    message.success("保存成功");
  } catch (error: any) {
    message.error(error?.message || error?.response?.data?.error || "保存失败");
  } finally {
    saving.value = false;
  }
};

onMounted(() => {
  fetchData().catch(err => console.error("Failed to fetch settings:", err));
});
</script>

<style scoped>
.pricing-settings-page {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
}

.form-tip {
  color: var(--text2);
  font-size: 12px;
  margin-top: 4px;
  line-height: 1.4;
}
</style>
