<template>
  <div class="captcha-settings-page">
    <div class="page-header">
      <div>
        <div class="page-kicker">SECURITY</div>
        <h1 class="page-title">验证码设置</h1>
        <p class="page-subtitle">独立管理登录/注册验证码与极验方案。</p>
      </div>
      <a-button type="primary" :loading="saving" @click="handleSave">保存设置</a-button>
    </div>

    <a-row :gutter="[18, 18]">
      <a-col :xs="24" :lg="12">
        <a-card :bordered="false" class="section-card">
          <div class="section-title">基础开关</div>
          <a-form layout="vertical">
            <a-form-item label="注册启用验证码">
              <a-switch v-model:checked="form.register_captcha_enabled" />
            </a-form-item>
            <a-form-item label="登录启用验证码">
              <a-switch v-model:checked="form.login_captcha_enabled" />
            </a-form-item>
            <a-form-item label="验证码方案">
              <a-radio-group v-model:value="form.auth_captcha_provider">
                <a-radio value="image">图形验证码</a-radio>
                <a-radio value="geetest">极验（GeeTest）</a-radio>
              </a-radio-group>
            </a-form-item>
            <a-alert type="info" show-icon message="切换到极验后，登录/注册会改为极验行为验证。" />
          </a-form>
        </a-card>
      </a-col>

      <a-col :xs="24" :lg="12">
        <a-card :bordered="false" class="section-card">
          <div class="section-title">图形验证码参数</div>
          <a-form layout="vertical">
            <a-row :gutter="12">
              <a-col :span="12">
                <a-form-item label="长度">
                  <a-input-number v-model:value="form.auth_captcha_code_len" :min="4" :max="12" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="复杂度">
                  <a-select v-model:value="form.auth_captcha_code_complexity" :options="codeComplexityOptions" />
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </a-card>

        <a-card :bordered="false" class="section-card">
          <div class="section-title">极验参数</div>
          <a-form layout="vertical">
            <a-form-item label="Captcha ID">
              <a-input v-model:value="form.auth_geetest_captcha_id" placeholder="请输入极验 captcha_id" />
            </a-form-item>
            <a-form-item label="Captcha Key">
              <a-input-password v-model:value="form.auth_geetest_captcha_key" placeholder="请输入极验 captcha_key" />
            </a-form-item>
            <a-form-item label="API Server">
              <a-input v-model:value="form.auth_geetest_api_server" placeholder="https://gcaptcha4.geetest.com" />
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

const form = reactive({
  register_captcha_enabled: true,
  login_captcha_enabled: false,
  auth_captcha_provider: "image",
  auth_captcha_code_len: 5,
  auth_captcha_code_complexity: "alnum",
  auth_geetest_captcha_id: "",
  auth_geetest_captcha_key: "",
  auth_geetest_api_server: "https://gcaptcha4.geetest.com"
});

const codeComplexityOptions = [
  { label: "纯数字", value: "digits" },
  { label: "纯字母(大写)", value: "letters" },
  { label: "字母+数字", value: "alnum" }
];

const parseBool = (value: unknown, def = false) => {
  if (value === undefined || value === null || value === "") return def;
  return value === true || value === "true" || value === "1" || value === 1;
};

const parseIntValue = (value: unknown, def = 0) => {
  if (value === undefined || value === null || value === "") return def;
  const n = Number(value);
  return Number.isFinite(n) ? Math.floor(n) : def;
};

const normalizeComplexity = (value: unknown, def: "digits" | "letters" | "alnum") => {
  const v = String(value || "").trim().toLowerCase();
  if (v === "digits" || v === "letters" || v === "alnum") return v as "digits" | "letters" | "alnum";
  return def;
};

const normalizeProvider = (value: unknown) => {
  const v = String(value || "").trim().toLowerCase();
  return v === "geetest" ? "geetest" : "image";
};

const fetchData = async () => {
  try {
    const res = await listSettings();
    const items = res.data?.items || [];
    const map = new Map<string, unknown>();
    items.forEach((item: any) => {
      map.set(item.key, item.value_json ?? item.value ?? "");
    });
    form.register_captcha_enabled = parseBool(map.get("auth_register_captcha_enabled"), true);
    form.login_captcha_enabled = parseBool(map.get("auth_login_captcha_enabled"), false);
    form.auth_captcha_provider = normalizeProvider(map.get("auth_captcha_provider"));
    form.auth_captcha_code_len = parseIntValue(map.get("auth_captcha_code_len"), 5);
    form.auth_captcha_code_complexity = normalizeComplexity(map.get("auth_captcha_code_complexity"), "alnum");
    form.auth_geetest_captcha_id = String(map.get("auth_geetest_captcha_id") || "");
    form.auth_geetest_captcha_key = String(map.get("auth_geetest_captcha_key") || "");
    form.auth_geetest_api_server = String(map.get("auth_geetest_api_server") || "https://gcaptcha4.geetest.com");
  } catch (error) {
    console.error("Failed to fetch captcha settings:", error);
  }
};

const handleSave = async () => {
  saving.value = true;
  try {
    const items = [
      { key: "auth_register_captcha_enabled", value: form.register_captcha_enabled ? "true" : "false" },
      { key: "auth_login_captcha_enabled", value: form.login_captcha_enabled ? "true" : "false" },
      { key: "auth_captcha_provider", value: normalizeProvider(form.auth_captcha_provider) },
      { key: "auth_captcha_code_len", value: String(form.auth_captcha_code_len || 5) },
      { key: "auth_captcha_code_complexity", value: normalizeComplexity(form.auth_captcha_code_complexity, "alnum") },
      { key: "auth_geetest_captcha_id", value: String(form.auth_geetest_captcha_id || "").trim() },
      { key: "auth_geetest_captcha_key", value: String(form.auth_geetest_captcha_key || "").trim() },
      { key: "auth_geetest_api_server", value: String(form.auth_geetest_api_server || "https://gcaptcha4.geetest.com").trim() }
    ];
    await updateSetting({ items });
    message.success("保存成功");
  } catch (error: any) {
    message.error(error?.response?.data?.error || "保存失败");
  } finally {
    saving.value = false;
  }
};

onMounted(fetchData);
</script>

<style scoped>
.captcha-settings-page {
  padding: 24px;
  background: radial-gradient(circle at top left, rgba(14, 165, 233, 0.08), transparent 45%);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}

.page-kicker {
  text-transform: uppercase;
  font-size: 12px;
  letter-spacing: 0.18em;
  color: rgba(15, 23, 42, 0.45);
}

.page-title {
  margin: 4px 0;
  font-size: 22px;
  font-weight: 700;
}

.page-subtitle {
  margin: 0;
  color: rgba(15, 23, 42, 0.55);
}

.section-card {
  border-radius: 16px;
  box-shadow: 0 12px 30px rgba(15, 23, 42, 0.08);
  padding: 6px 6px 10px;
}

.section-card + .section-card {
  margin-top: 18px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 12px;
}
</style>
