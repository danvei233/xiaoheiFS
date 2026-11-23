<template>
    <div class="login-page">
      <header class="auth-topbar">
        <router-link to="/" class="auth-topbar-brand">
          <span class="auth-topbar-mark" aria-hidden="true"></span>
          <span>{{ site.siteName }}</span>
        </router-link>
        <AppearanceSettings />
      </header>
      <div class="login-card">
        <h1 class="title">{{ t('auth.userLogin') }}</h1>
        <p class="subtitle">{{ t('auth.loginSubtitle') }}</p>

        <a-form :model="form" layout="vertical" @finish="onSubmit">
          <a-tabs v-model:activeKey="loginMode" size="small" style="margin-bottom: 8px">
            <a-tab-pane key="account" :tab="t('auth.accountTab')" />
            <a-tab-pane key="phone" :tab="t('auth.phoneTab')" />
          </a-tabs>

          <a-form-item v-if="loginMode === 'account'" :label="t('auth.account')" name="username" :rules="[{ required: true, message: t('auth.accountRequired') }]">
            <a-input v-model:value="form.username" :placeholder="t('auth.accountPlaceholder')" size="large" :maxlength="INPUT_LIMITS.EMAIL" />
          </a-form-item>

          <a-form-item
            v-else
            :label="t('common.phone')"
            name="phone"
            :rules="[{ required: true, message: t('auth.phoneRequired') }, { validator: validatePhoneLogin, trigger: 'blur' }]"
          >
            <a-input v-model:value="form.phone" :placeholder="t('auth.phonePlaceholder')" size="large" :maxlength="INPUT_LIMITS.PHONE" />
          </a-form-item>

          <a-form-item :label="t('common.password')" name="password" :rules="[{ required: true, message: t('auth.passwordRequired') }]">
            <a-input-password v-model:value="form.password" :placeholder="t('auth.passwordPlaceholder')" size="large" :maxlength="INPUT_LIMITS.PASSWORD" />
          </a-form-item>

          <a-form-item
            v-if="settings.login_captcha_enabled"
            :label="settings.captcha_provider === 'geetest' ? t('auth.behaviorCaptcha') : t('auth.imageCaptcha')"
            name="captcha_code"
            :rules="settings.captcha_provider === 'geetest' ? [] : [{ required: true, message: t('auth.captchaRequired') }]"
          >
            <div v-if="settings.captcha_provider !== 'geetest'" class="captcha-row">
              <a-input v-model:value="form.captcha_code" :placeholder="t('auth.captchaPlaceholder')" />
              <div class="captcha-img" @click="refreshCaptcha">
                <img v-if="captchaImage" :src="captchaImage" alt="captcha" />
                <span v-else>{{ t('auth.clickRefresh') }}</span>
              </div>
            </div>
            <div v-else class="captcha-geetest">
              <a-button @click="verifyGeeTest" :disabled="!geetest.ready" :loading="geetest.loading">
                {{ geetest.passed ? t('auth.geetestPassed') : t('auth.geetestClick') }}
              </a-button>
              <span class="captcha-geetest-status">{{ geetest.passed ? t('auth.geetestVerified') : t('auth.geetestUnverified') }}</span>
            </div>
          </a-form-item>

          <div class="actions-row">
            <router-link to="/forgot-password">{{ t('auth.forgot') }}</router-link>
            <router-link to="/register">{{ t('auth.registerNow') }}</router-link>
          </div>

          <a-button type="primary" html-type="submit" block size="large" :loading="auth.loading">
            {{ t('auth.loginButton') }}
          </a-button>
        </a-form>
      </div>
    </div>
</template>

<script setup>
import { reactive, ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { message } from "ant-design-vue";
import AppearanceSettings from "@/components/AppearanceSettings.vue";
import { useAuthStore } from "@/stores/auth";
import { useSiteStore } from "@/stores/site";
import { getAuthSettings, getCaptcha } from "@/services/user";
import { INPUT_LIMITS } from "@/constants/inputLimits";
import { t } from "@/locales";

const form = reactive({
  username: "",
  phone: "",
  password: "",
  captcha_code: ""
});
const loginMode = ref("account");

const auth = useAuthStore();
const site = useSiteStore();
const router = useRouter();
const route = useRoute();

const settings = reactive({
  login_captcha_enabled: false,
  captcha_provider: "image"
});

const captchaId = ref("");
const captchaImage = ref("");
const geetest = reactive({
  widget: null,
  ready: false,
  loading: false,
  passed: false,
  lot_number: "",
  captcha_output: "",
  pass_token: "",
  gen_time: ""
});

const resetGeeTestResult = () => {
  geetest.passed = false;
  geetest.lot_number = "";
  geetest.captcha_output = "";
  geetest.pass_token = "";
  geetest.gen_time = "";
};

const ensureGeeTestScript = async () => {
  if (window.initGeetest4) return;
  await new Promise((resolve, reject) => {
    const existed = document.querySelector("script[data-geetest='gt4']");
    if (existed) {
      existed.addEventListener("load", resolve, { once: true });
      existed.addEventListener("error", reject, { once: true });
      return;
    }
    const script = document.createElement("script");
    script.src = "https://static.geetest.com/v4/gt4.js";
    script.async = true;
    script.defer = true;
    script.dataset.geetest = "gt4";
    script.onload = resolve;
    script.onerror = reject;
    document.head.appendChild(script);
  });
};

const initGeeTest = async (captchaID) => {
  resetGeeTestResult();
  geetest.ready = false;
  geetest.widget = null;
  if (!captchaID) return;
  await ensureGeeTestScript();
  await new Promise((resolve, reject) => {
    window.initGeetest4(
      { captchaId: captchaID, product: "bind", language: "zho" },
      (captchaObj) => {
        geetest.widget = captchaObj;
        geetest.ready = true;
        captchaObj.onSuccess(() => {
          const result = captchaObj.getValidate ? captchaObj.getValidate() : null;
          geetest.lot_number = String(result?.lot_number || "");
          geetest.captcha_output = String(result?.captcha_output || "");
          geetest.pass_token = String(result?.pass_token || "");
          geetest.gen_time = String(result?.gen_time || "");
          geetest.passed = Boolean(
            geetest.lot_number && geetest.captcha_output && geetest.pass_token && geetest.gen_time
          );
        });
        captchaObj.onError(() => {
          resetGeeTestResult();
          message.error(t('auth.geetestInitFailed'));
        });
        resolve(true);
      }
    );
  }).catch(() => {
    message.error(t('auth.geetestScriptFailed'));
  });
};

const verifyGeeTest = async () => {
  if (!geetest.widget || !geetest.ready) {
    message.warning(t('auth.geetestNotReady'));
    return;
  }
  geetest.loading = true;
  try {
    resetGeeTestResult();
    geetest.widget.showCaptcha();
  } finally {
    geetest.loading = false;
  }
};

const refreshCaptcha = async () => {
  if (!settings.login_captcha_enabled) return;
  const res = await getCaptcha();
  const provider = String(res.data?.captcha_provider || settings.captcha_provider || "image").toLowerCase();
  settings.captcha_provider = provider === "geetest" ? "geetest" : "image";
  captchaId.value = String(res.data?.captcha_id || "");
  if (settings.captcha_provider === "geetest") {
    captchaImage.value = "";
    await initGeeTest(captchaId.value);
    return;
  }
  const base64 = String(res.data?.image_base64 || "");
  captchaImage.value = base64 ? `data:image/png;base64,${base64}` : "";
  resetGeeTestResult();
};

const loadSettings = async () => {
  try {
    const res = await getAuthSettings();
    Object.assign(settings, res.data || {});
  } catch (error) {
    console.error("Failed to fetch auth settings:", error);
  } finally {
    refreshCaptcha();
  }
};

const validatePhoneLogin = () => {
  const value = String(form.phone || "").trim();
  if (!value) return Promise.resolve();
  if (!/^[0-9+\-\s]{6,20}$/.test(value)) return Promise.reject(t('auth.invalidPhone'));
  return Promise.resolve();
};

const onSubmit = async () => {
  try {
    const loginAccount = loginMode.value === "phone" ? String(form.phone || "").trim() : String(form.username || "").trim();
    if (loginMode.value === "phone") {
      if (loginAccount.length > INPUT_LIMITS.PHONE) {
        message.error(t('auth.phoneTooLong', { n: INPUT_LIMITS.PHONE }));
        return;
      }
    } else {
      if (loginAccount.length > INPUT_LIMITS.EMAIL) {
        message.error(t('auth.accountTooLong', { n: INPUT_LIMITS.EMAIL }));
        return;
      }
    }
    if (String(form.password || "").length > INPUT_LIMITS.PASSWORD) {
      message.error(t('auth.passwordTooLong', { n: INPUT_LIMITS.PASSWORD }));
      return;
    }
    if (settings.login_captcha_enabled && settings.captcha_provider === "geetest" && !geetest.passed) {
      message.warning(t('auth.completeGeetest'));
      return;
    }
    const token = await auth.login({
      username: loginAccount,
      password: form.password,
      captcha_id: captchaId.value,
      captcha_code: form.captcha_code,
      lot_number: geetest.lot_number,
      captcha_output: geetest.captcha_output,
      pass_token: geetest.pass_token,
      gen_time: geetest.gen_time
    });
    if (!token) {
      message.error(t('auth.loginFailed'));
      return;
    }
    await auth.fetchMe();
    message.success(t('auth.loginSuccess'));
    router.replace(String(route.query.redirect || "/console"));
  } catch (error) {
    const msg =
      error?.response?.data?.error ||
      error?.response?.data?.message ||
      error?.message ||
      t('auth.loginFailed');
    message.error(msg);
    if (settings.login_captcha_enabled) {
      refreshCaptcha();
    }
  }
};

onMounted(() => {
  loadSettings();
});
</script>

<style scoped>
.login-page {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 88px 24px 40px;
  background: var(--theme-canvas, var(--bg-primary));
  color: var(--text-primary);
}

.login-card {
  width: 100%;
  max-width: 420px;
  background: var(--theme-surface, var(--card));
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  backdrop-filter: var(--theme-backdrop);
  padding: 24px;
}

.auth-topbar {
  position: absolute;
  inset-block-start: 0;
  inset-inline: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  height: 64px;
  padding-inline: clamp(16px, 4vw, 40px);
  border-bottom: 1px solid var(--border);
  background: color-mix(in srgb, var(--card) 86%, transparent);
  color: var(--text-primary);
  backdrop-filter: var(--theme-backdrop, blur(12px));
  -webkit-backdrop-filter: var(--theme-backdrop, blur(12px));
}

.auth-topbar-brand {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
  color: inherit;
  font-size: 15px;
  font-weight: 650;
  text-decoration: none;
}

.auth-topbar-brand > span:last-child {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.auth-topbar-mark {
  flex: 0 0 22px;
  width: 22px;
  height: 22px;
  border-radius: 7px;
  background: linear-gradient(140deg, var(--primary), color-mix(in srgb, var(--primary) 45%, #0b1220));
  box-shadow: inset 0 1px 0 #ffffff66;
}

.title {
  color: var(--text-primary);
  margin: 0;
}

.subtitle {
  color: var(--text-secondary);
  margin: 8px 0 18px;
}

.captcha-row {
  display: flex;
  gap: 10px;
}

.captcha-img {
  width: 120px;
  height: 40px;
  border: 1px dashed var(--border);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  overflow: hidden;
  color: var(--text-secondary);
}

.captcha-img img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.actions-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 14px;
}

.captcha-geetest {
  display: flex;
  align-items: center;
  gap: 10px;
}

.captcha-geetest-status {
  color: var(--text-secondary);
  font-size: 12px;
}
</style>
