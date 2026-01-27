<template>
  <div class="lifecycle-settings-page">
    <div class="page-header">
      <h1 class="page-title">生命周期设置</h1>
      <a-button type="primary" @click="handleSave" :loading="saving">保存更改</a-button>
    </div>

    <a-row :gutter="16">
      <a-col :span="12">
        <a-card title="到期提醒" :bordered="false">
          <a-form :model="form" layout="vertical">
            <a-form-item label="邮件提醒">
              <a-switch v-model:checked="form.email_expire_enabled" />
              <span style="margin-left: 8px">开启后会在实例到期前发送提醒</span>
            </a-form-item>

            <a-form-item label="提前提醒天数">
              <a-input-number v-model:value="form.expire_reminder_days" :min="0" style="width: 100%" />
              <div class="form-tip">例如 7 表示到期前 7 天发送提醒</div>
            </a-form-item>
          </a-form>
        </a-card>

        <a-card title="VPS 到期删除策略" :bordered="false" style="margin-top: 16px">
          <a-form :model="form" layout="vertical">
            <a-form-item label="是否自动删除">
              <a-switch v-model:checked="form.auto_delete_enabled" />
              <span style="margin-left: 8px">开启后将自动回收实例</span>
            </a-form-item>

            <a-form-item label="到期多少天后自动删除">
              <a-input-number
                v-model:value="form.auto_delete_days"
                :min="0"
                :disabled="!form.auto_delete_enabled"
                style="width: 100%"
              />
              <div class="form-tip">到期超过 N 天后，会由定时任务执行删除/回收（默认每天 03:00 执行）。</div>
            </a-form-item>
          </a-form>
        </a-card>
      </a-col>

      <a-col :span="12">
        <a-card title="紧急续费" :bordered="false">
          <a-form :model="form" layout="vertical">
            <a-form-item label="是否允许紧急续费">
              <a-switch v-model:checked="form.emergency_renew_enabled" />
              <span style="margin-left: 8px">允许用户在到期前窗口内触发紧急续费</span>
            </a-form-item>

            <a-form-item label="可紧急续费窗口（到期前）">
              <a-input-number
                v-model:value="form.emergency_renew_window_days"
                :min="0"
                :disabled="!form.emergency_renew_enabled"
                style="width: 100%"
              />
              <div class="form-tip">0 表示不限制窗口（只要未到期都允许）。</div>
            </a-form-item>

            <a-form-item label="每次紧急续费延长天数">
              <a-input-number
                v-model:value="form.emergency_renew_days"
                :min="1"
                :disabled="!form.emergency_renew_enabled"
                style="width: 100%"
              />
              <div class="form-tip">每次紧急续费会将到期时间延长 N 天。</div>
            </a-form-item>

            <a-form-item label="紧急续费间隔（小时）">
              <a-input-number
                v-model:value="form.emergency_renew_interval_hours"
                :min="1"
                :disabled="!form.emergency_renew_enabled"
                style="width: 100%"
              />
              <div class="form-tip">两次紧急续费之间的最小间隔时间（小时）。</div>
            </a-form-item>
          </a-form>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { message } from "ant-design-vue";
import { listSettings, updateSetting } from "@/services/admin";

const saving = ref(false);

const toBool = (raw: unknown, fallback: boolean) => {
  const val = String(raw ?? "").trim().toLowerCase();
  if (val === "true") return true;
  if (val === "false") return false;
  return fallback;
};

const toInt = (raw: unknown, fallback: number) => {
  const s = String(raw ?? "").trim();
  if (!s) return fallback;
  const first = s
    .split(",")
    .map((x) => x.trim())
    .find(Boolean);
  const n = Number.parseInt(first ?? "", 10);
  return Number.isFinite(n) ? n : fallback;
};

const form = reactive({
  email_expire_enabled: false,
  expire_reminder_days: 7,

  auto_delete_enabled: false,
  auto_delete_days: 7,

  emergency_renew_enabled: true,
  emergency_renew_window_days: 7,
  emergency_renew_days: 1,
  emergency_renew_interval_hours: 720
});

const fetchData = async () => {
  try {
    const res = await listSettings();
    const items = res.data?.items || [];
    const data: Record<string, string> = {};
    items.forEach((item: any) => {
      data[item.key] = item.value;
    });

    form.email_expire_enabled = toBool(data.email_expire_enabled, false);
    form.expire_reminder_days = Math.max(0, toInt(data.expire_reminder_days, 7));

    form.auto_delete_enabled = toBool(data.auto_delete_enabled, false);
    form.auto_delete_days = Math.max(0, toInt(data.auto_delete_days, 7));

    form.emergency_renew_enabled = toBool(data.emergency_renew_enabled, true);
    form.emergency_renew_window_days = Math.max(0, toInt(data.emergency_renew_window_days, 7));
    form.emergency_renew_days = Math.max(1, toInt(data.emergency_renew_days, 1));
    form.emergency_renew_interval_hours = Math.max(1, toInt(data.emergency_renew_interval_hours, 720));
  } catch (error) {
    console.error("Failed to fetch settings:", error);
  }
};

const handleSave = async () => {
  saving.value = true;
  try {
    const items = [
      { key: "email_expire_enabled", value: String(form.email_expire_enabled) },
      { key: "expire_reminder_days", value: String(Math.max(0, form.expire_reminder_days)) },

      { key: "auto_delete_enabled", value: String(form.auto_delete_enabled) },
      { key: "auto_delete_days", value: String(Math.max(0, form.auto_delete_days)) },

      { key: "emergency_renew_enabled", value: String(form.emergency_renew_enabled) },
      { key: "emergency_renew_window_days", value: String(Math.max(0, form.emergency_renew_window_days)) },
      { key: "emergency_renew_days", value: String(Math.max(1, form.emergency_renew_days)) },
      { key: "emergency_renew_interval_hours", value: String(Math.max(1, form.emergency_renew_interval_hours)) }
    ];
    await updateSetting({ items });
    message.success("保存成功");
  } catch (error: any) {
    message.error(error.response?.data?.error || "保存失败");
  } finally {
    saving.value = false;
  }
};

onMounted(() => {
  fetchData();
});
</script>

<style scoped>
.lifecycle-settings-page {
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
}
</style>
