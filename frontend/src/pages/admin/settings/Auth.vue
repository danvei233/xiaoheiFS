<template>
  <div class="auth-settings-page">
    <div class="page-header">
      <div>
        <div class="page-kicker">账号与安全</div>
        <h1 class="page-title">注册与登录设置</h1>
        <p class="page-subtitle">统一管理注册入口、密码规则、验证码与登录风控。</p>
      </div>
      <a-button type="primary" @click="handleSave" :loading="saving">保存设置</a-button>
    </div>

    <a-row :gutter="[18, 18]">
      <a-col :xs="24" :lg="14">
        <a-card :bordered="false" class="section-card">
          <div class="section-title">注册入口</div>
          <a-form layout="vertical">
            <a-form-item label="是否开启注册">
              <a-switch v-model:checked="form.register_enabled" />
              <span class="field-help">关闭后前台注册入口会提示禁止注册。</span>
            </a-form-item>

            <a-form-item label="必填字段">
              <a-checkbox-group v-model:value="form.register_required_fields" :options="requiredFieldOptions" />
              <div class="field-hint">
                用户名、密码为系统必填；邮箱必填请使用“邮箱必填”开关控制。
              </div>
            </a-form-item>
          </a-form>
        </a-card>

        <a-card :bordered="false" class="section-card">
          <div class="section-title">密码规则</div>
          <a-form layout="vertical">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="最小长度">
                  <a-input-number v-model:value="form.password_min_len" :min="6" :max="64" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="必须包含">
                  <div class="inline-switches">
                    <a-switch v-model:checked="form.password_require_upper" /> 大写
                    <a-switch v-model:checked="form.password_require_lower" /> 小写
                    <a-switch v-model:checked="form.password_require_number" /> 数字
                    <a-switch v-model:checked="form.password_require_symbol" /> 符号
                  </div>
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </a-card>
      </a-col>

      <a-col :xs="24" :lg="10">
        <a-card :bordered="false" class="section-card highlight">
          <div class="section-title">注册验证</div>
          <a-form layout="vertical">
            <a-form-item label="验证码类型">
              <a-checkbox-group v-model:value="form.register_verify_channels" :options="verifyChannelOptions" />
            </a-form-item>
            <a-form-item label="邮箱必填">
              <a-switch v-model:checked="form.register_email_required" />
            </a-form-item>
            <a-form-item label="验证码有效期（秒）">
              <a-input-number v-model:value="form.register_verify_ttl_sec" :min="60" :max="3600" style="width: 100%" />
            </a-form-item>
            <a-form-item label="启用图形验证码">
              <a-switch v-model:checked="form.register_captcha_enabled" />
            </a-form-item>
            <a-divider style="margin: 10px 0 14px">验证码策略</a-divider>
            <a-row :gutter="12">
              <a-col :span="12">
                <a-form-item label="短信长度">
                  <a-input-number v-model:value="form.auth_sms_code_len" :min="4" :max="12" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="短信复杂度">
                  <a-select v-model:value="form.auth_sms_code_complexity" :options="codeComplexityOptions" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="12">
              <a-col :span="12">
                <a-form-item label="邮箱长度">
                  <a-input-number v-model:value="form.auth_email_code_len" :min="4" :max="12" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="邮箱复杂度">
                  <a-select v-model:value="form.auth_email_code_complexity" :options="codeComplexityOptions" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="12">
              <a-col :span="12">
                <a-form-item label="图形长度">
                  <a-input-number v-model:value="form.auth_captcha_code_len" :min="4" :max="12" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="图形复杂度">
                  <a-select v-model:value="form.auth_captcha_code_complexity" :options="codeComplexityOptions" />
                </a-form-item>
              </a-col>
            </a-row>

            <a-alert
              type="info"
              show-icon
              message="模板维护入口：邮箱模板请在「系统设置-邮件设置」，短信模板请在「系统设置-短信设置」。"
            />
          </a-form>
        </a-card>

        <a-card :bordered="false" class="section-card">
          <div class="section-title">登录保护</div>
          <a-form layout="vertical">
            <a-form-item label="启用登录图形验证码">
              <a-switch v-model:checked="form.login_captcha_enabled" />
            </a-form-item>
            <a-form-item label="登录提醒开关">
              <a-switch v-model:checked="form.auth_login_notify_enabled" />
            </a-form-item>
            <a-form-item label="登录提醒触发">
              <a-checkbox-group v-model:value="form.auth_login_notify_events" :options="loginNotifyEventOptions" />
            </a-form-item>
            <a-form-item label="登录提醒渠道">
              <a-checkbox-group v-model:value="form.auth_login_notify_channels" :options="verifyChannelOptions" />
            </a-form-item>
            <a-form-item label="找回密码开关">
              <a-switch v-model:checked="form.auth_password_reset_enabled" />
            </a-form-item>
            <a-form-item label="找回密码渠道">
              <a-checkbox-group v-model:value="form.auth_password_reset_channels" :options="verifyChannelOptions" />
            </a-form-item>
            <a-form-item label="找回验证码有效期（秒）">
              <a-input-number v-model:value="form.auth_password_reset_verify_ttl_sec" :min="60" :max="3600" style="width: 100%" />
            </a-form-item>
            <a-form-item label="邮箱绑定功能">
              <a-switch v-model:checked="form.auth_email_bind_enabled" />
            </a-form-item>
            <a-form-item label="手机号绑定功能">
              <a-switch v-model:checked="form.auth_phone_bind_enabled" />
            </a-form-item>
            <a-form-item label="绑定验证码有效期（秒）">
              <a-input-number v-model:value="form.auth_contact_bind_verify_ttl_sec" :min="60" :max="3600" style="width: 100%" />
            </a-form-item>
            <a-form-item label="未开2FA时：首次绑定需密码">
              <a-switch v-model:checked="form.auth_bind_require_password_when_no_2fa" />
            </a-form-item>
            <a-form-item label="未开2FA时：换绑需密码">
              <a-switch v-model:checked="form.auth_rebind_require_password_when_no_2fa" />
            </a-form-item>
            <a-form-item label="2FA总开关">
              <a-switch v-model:checked="form.auth_2fa_enabled" />
            </a-form-item>
            <a-form-item label="2FA-绑定流程开关">
              <a-switch v-model:checked="form.auth_2fa_bind_enabled" />
            </a-form-item>
            <a-form-item label="2FA-换绑流程开关">
              <a-switch v-model:checked="form.auth_2fa_rebind_enabled" />
            </a-form-item>
            <a-form-item label="启用登录频率限制">
              <a-switch v-model:checked="form.login_rate_limit_enabled" />
            </a-form-item>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="窗口秒数">
                  <a-input-number v-model:value="form.login_rate_limit_window_sec" :min="60" :max="3600" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="最大次数">
                  <a-input-number v-model:value="form.login_rate_limit_max_attempts" :min="3" :max="30" style="width: 100%" />
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted, computed } from "vue";
import { message } from "ant-design-vue";
import { listSettings, updateSetting } from "@/services/admin";

const saving = ref(false);

const form = reactive({
  register_enabled: true,
  register_required_fields: ["username", "password"],
  register_email_required: true,
  password_min_len: 6,
  password_require_upper: false,
  password_require_lower: false,
  password_require_number: false,
  password_require_symbol: false,
  register_verify_type: "none",
  register_verify_channels: ["email"],
  register_verify_ttl_sec: 600,
  register_captcha_enabled: true,
  login_captcha_enabled: false,
  login_rate_limit_enabled: true,
  login_rate_limit_window_sec: 300,
  login_rate_limit_max_attempts: 5,
  auth_login_notify_enabled: true,
  auth_login_notify_events: ["first", "ip_change"],
  auth_login_notify_channels: ["email"],
  auth_password_reset_enabled: true,
  auth_password_reset_channels: ["email"],
  auth_password_reset_verify_ttl_sec: 600,
  auth_sms_code_len: 6,
  auth_sms_code_complexity: "digits",
  auth_email_code_len: 6,
  auth_email_code_complexity: "alnum",
  auth_captcha_code_len: 5,
  auth_captcha_code_complexity: "alnum",
  auth_email_bind_enabled: true,
  auth_phone_bind_enabled: true,
  auth_contact_bind_verify_ttl_sec: 600,
  auth_bind_require_password_when_no_2fa: false,
  auth_rebind_require_password_when_no_2fa: true,
  auth_2fa_enabled: true,
  auth_2fa_bind_enabled: true,
  auth_2fa_rebind_enabled: true
});

const requiredFieldOptions = computed(() => [
  { label: "用户名", value: "username", disabled: true },
  { label: "密码", value: "password", disabled: true },
  { label: "手机号", value: "phone" },
  { label: "QQ", value: "qq" }
]);
const verifyChannelOptions = [
  { label: "邮箱", value: "email" },
  { label: "短信", value: "sms" }
];
const codeComplexityOptions = [
  { label: "纯数字", value: "digits" },
  { label: "纯字母(大写)", value: "letters" },
  { label: "字母+数字", value: "alnum" }
];
const loginNotifyEventOptions = [
  { label: "首次登录", value: "first" },
  { label: "IP变化", value: "ip_change" }
];

const parseBool = (value: any, def = false) => {
  if (value === undefined || value === null || value === "") return def;
  return value === true || value === "true" || value === "1" || value === 1;
};

const parseIntValue = (value: any, def = 0) => {
  if (value === undefined || value === null || value === "") return def;
  const n = Number(value);
  return Number.isFinite(n) ? Math.floor(n) : def;
};

const parseList = (value: any, def: string[]) => {
  if (!value) return def;
  try {
    const parsed = typeof value === "string" ? JSON.parse(value) : value;
    return Array.isArray(parsed) ? parsed.map((v) => String(v)) : def;
  } catch {
    return def;
  }
};

const normalizeComplexity = (value: any, def: string) => {
  const v = String(value || "").trim().toLowerCase();
  if (v === "digits" || v === "letters" || v === "alnum") return v;
  return def;
};

const normalizeRequired = () => {
  const set = new Set(form.register_required_fields.map((v) => String(v).toLowerCase()));
  set.add("username");
  set.add("password");
  set.delete("email");
  form.register_required_fields = Array.from(set);
};

const fetchData = async () => {
  try {
    const res = await listSettings();
    const items = res.data?.items || [];
    const map = new Map<string, any>();
    items.forEach((item: any) => {
      map.set(item.key, item.value_json ?? item.value ?? "");
    });

    form.register_enabled = parseBool(map.get("auth_register_enabled"), true);
    form.register_required_fields = parseList(
      map.get("auth_register_required_fields"),
      ["username", "password"]
    );
    form.register_email_required = parseBool(map.get("auth_register_email_required"), true);
    form.password_min_len = parseIntValue(map.get("auth_password_min_len"), 6);
    form.password_require_upper = parseBool(map.get("auth_password_require_upper"), false);
    form.password_require_lower = parseBool(map.get("auth_password_require_lower"), false);
    form.password_require_number = parseBool(map.get("auth_password_require_number"), false);
    form.password_require_symbol = parseBool(map.get("auth_password_require_symbol"), false);
    form.register_verify_type = String(map.get("auth_register_verify_type") || "none");
    form.register_verify_channels = parseList(map.get("auth_register_verify_channels"), form.register_verify_type === "none" ? [] : [form.register_verify_type]);
    form.register_verify_ttl_sec = parseIntValue(map.get("auth_register_verify_ttl_sec"), 600);
    form.register_captcha_enabled = parseBool(map.get("auth_register_captcha_enabled"), true);
    form.login_captcha_enabled = parseBool(map.get("auth_login_captcha_enabled"), false);
    form.login_rate_limit_enabled = parseBool(map.get("auth_login_rate_limit_enabled"), true);
    form.login_rate_limit_window_sec = parseIntValue(map.get("auth_login_rate_limit_window_sec"), 300);
    form.login_rate_limit_max_attempts = parseIntValue(map.get("auth_login_rate_limit_max_attempts"), 5);
    form.auth_login_notify_enabled = parseBool(map.get("auth_login_notify_enabled"), true);
    form.auth_login_notify_channels = parseList(map.get("auth_login_notify_channels"), ["email"]);
    form.auth_login_notify_events = [];
    if (parseBool(map.get("auth_login_notify_on_first_login"), true)) form.auth_login_notify_events.push("first");
    if (parseBool(map.get("auth_login_notify_on_ip_change"), true)) form.auth_login_notify_events.push("ip_change");
    form.auth_password_reset_enabled = parseBool(map.get("auth_password_reset_enabled"), true);
    form.auth_password_reset_channels = parseList(map.get("auth_password_reset_channels"), ["email"]);
    form.auth_password_reset_verify_ttl_sec = parseIntValue(map.get("auth_password_reset_verify_ttl_sec"), 600);
    form.auth_sms_code_len = parseIntValue(map.get("auth_sms_code_len"), 6);
    form.auth_sms_code_complexity = normalizeComplexity(map.get("auth_sms_code_complexity"), "digits");
    form.auth_email_code_len = parseIntValue(map.get("auth_email_code_len"), 6);
    form.auth_email_code_complexity = normalizeComplexity(map.get("auth_email_code_complexity"), "alnum");
    form.auth_captcha_code_len = parseIntValue(map.get("auth_captcha_code_len"), 5);
    form.auth_captcha_code_complexity = normalizeComplexity(map.get("auth_captcha_code_complexity"), "alnum");
    form.auth_email_bind_enabled = parseBool(map.get("auth_email_bind_enabled"), true);
    form.auth_phone_bind_enabled = parseBool(map.get("auth_phone_bind_enabled"), true);
    form.auth_contact_bind_verify_ttl_sec = parseIntValue(map.get("auth_contact_bind_verify_ttl_sec"), 600);
    form.auth_bind_require_password_when_no_2fa = parseBool(map.get("auth_bind_require_password_when_no_2fa"), false);
    form.auth_rebind_require_password_when_no_2fa = parseBool(map.get("auth_rebind_require_password_when_no_2fa"), true);
    form.auth_2fa_enabled = parseBool(map.get("auth_2fa_enabled"), true);
    form.auth_2fa_bind_enabled = parseBool(map.get("auth_2fa_bind_enabled"), true);
    form.auth_2fa_rebind_enabled = parseBool(map.get("auth_2fa_rebind_enabled"), true);
    normalizeRequired();
  } catch (error) {
    console.error("Failed to fetch auth settings:", error);
  }
};

const handleSave = async () => {
  saving.value = true;
  try {
    normalizeRequired();
    if (!Array.isArray(form.register_verify_channels) || form.register_verify_channels.length === 0) {
      form.register_verify_type = "none";
    } else if (form.register_verify_channels.includes("email")) {
      form.register_verify_type = "email";
    } else {
      form.register_verify_type = "sms";
    }
    const items = [
      { key: "auth_register_enabled", value: form.register_enabled ? "true" : "false" },
      { key: "auth_register_required_fields", value: JSON.stringify(form.register_required_fields) },
      { key: "auth_register_email_required", value: form.register_email_required ? "true" : "false" },
      { key: "auth_password_min_len", value: String(form.password_min_len ?? 6) },
      { key: "auth_password_require_upper", value: form.password_require_upper ? "true" : "false" },
      { key: "auth_password_require_lower", value: form.password_require_lower ? "true" : "false" },
      { key: "auth_password_require_number", value: form.password_require_number ? "true" : "false" },
      { key: "auth_password_require_symbol", value: form.password_require_symbol ? "true" : "false" },
      { key: "auth_register_verify_type", value: form.register_verify_type },
      { key: "auth_register_verify_channels", value: JSON.stringify(form.register_verify_channels || []) },
      { key: "auth_register_verify_ttl_sec", value: String(form.register_verify_ttl_sec ?? 600) },
      { key: "auth_register_captcha_enabled", value: form.register_captcha_enabled ? "true" : "false" },
      { key: "auth_login_captcha_enabled", value: form.login_captcha_enabled ? "true" : "false" },
      { key: "auth_login_rate_limit_enabled", value: form.login_rate_limit_enabled ? "true" : "false" },
      { key: "auth_login_rate_limit_window_sec", value: String(form.login_rate_limit_window_sec ?? 300) },
      { key: "auth_login_rate_limit_max_attempts", value: String(form.login_rate_limit_max_attempts ?? 5) },
      { key: "auth_login_notify_enabled", value: form.auth_login_notify_enabled ? "true" : "false" },
      { key: "auth_login_notify_channels", value: JSON.stringify(form.auth_login_notify_channels || []) },
      { key: "auth_login_notify_on_first_login", value: form.auth_login_notify_events.includes("first") ? "true" : "false" },
      { key: "auth_login_notify_on_ip_change", value: form.auth_login_notify_events.includes("ip_change") ? "true" : "false" },
      { key: "auth_password_reset_enabled", value: form.auth_password_reset_enabled ? "true" : "false" },
      { key: "auth_password_reset_channels", value: JSON.stringify(form.auth_password_reset_channels || []) },
      { key: "auth_password_reset_verify_ttl_sec", value: String(form.auth_password_reset_verify_ttl_sec ?? 600) },
      { key: "auth_sms_code_len", value: String(form.auth_sms_code_len ?? 6) },
      { key: "auth_sms_code_complexity", value: normalizeComplexity(form.auth_sms_code_complexity, "digits") },
      { key: "auth_email_code_len", value: String(form.auth_email_code_len ?? 6) },
      { key: "auth_email_code_complexity", value: normalizeComplexity(form.auth_email_code_complexity, "alnum") },
      { key: "auth_captcha_code_len", value: String(form.auth_captcha_code_len ?? 5) },
      { key: "auth_captcha_code_complexity", value: normalizeComplexity(form.auth_captcha_code_complexity, "alnum") },
      { key: "auth_email_bind_enabled", value: form.auth_email_bind_enabled ? "true" : "false" },
      { key: "auth_phone_bind_enabled", value: form.auth_phone_bind_enabled ? "true" : "false" },
      { key: "auth_contact_bind_verify_ttl_sec", value: String(form.auth_contact_bind_verify_ttl_sec ?? 600) },
      { key: "auth_bind_require_password_when_no_2fa", value: form.auth_bind_require_password_when_no_2fa ? "true" : "false" },
      { key: "auth_rebind_require_password_when_no_2fa", value: form.auth_rebind_require_password_when_no_2fa ? "true" : "false" },
      { key: "auth_2fa_enabled", value: form.auth_2fa_enabled ? "true" : "false" },
      { key: "auth_2fa_bind_enabled", value: form.auth_2fa_bind_enabled ? "true" : "false" },
      { key: "auth_2fa_rebind_enabled", value: form.auth_2fa_rebind_enabled ? "true" : "false" }
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
.auth-settings-page {
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

.section-card.highlight {
  background: linear-gradient(145deg, #f8fafc, #eef6ff);
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 12px;
}

.field-help {
  margin-left: 10px;
  color: rgba(15, 23, 42, 0.45);
  font-size: 12px;
}

.field-hint {
  margin-top: 6px;
  font-size: 12px;
  color: rgba(15, 23, 42, 0.55);
}

.inline-switches {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}
</style>
