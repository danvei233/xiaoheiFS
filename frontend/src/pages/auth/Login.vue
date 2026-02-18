<template>
  <a-config-provider :theme="{ algorithm: theme.darkAlgorithm }">
    <div class="login-page">
      <div class="login-card">
        <h1 class="title">用户登录</h1>
        <p class="subtitle">登录以访问控制台</p>

        <a-form :model="form" layout="vertical" @finish="onSubmit">
          <a-tabs v-model:activeKey="loginMode" size="small" style="margin-bottom: 8px">
            <a-tab-pane key="account" tab="账号登录" />
            <a-tab-pane key="phone" tab="手机号登录" />
          </a-tabs>

          <a-form-item v-if="loginMode === 'account'" label="账号" name="username" :rules="[{ required: true, message: '请输入账号' }]">
            <a-input v-model:value="form.username" placeholder="请输入用户名/邮箱" size="large" :maxlength="INPUT_LIMITS.EMAIL" />
          </a-form-item>

          <a-form-item
            v-else
            label="手机号"
            name="phone"
            :rules="[{ required: true, message: '请输入手机号' }, { validator: validatePhoneLogin, trigger: 'blur' }]"
          >
            <a-input v-model:value="form.phone" placeholder="请输入手机号" size="large" :maxlength="INPUT_LIMITS.PHONE" />
          </a-form-item>

          <a-form-item label="密码" name="password" :rules="[{ required: true, message: '请输入密码' }]">
            <a-input-password v-model:value="form.password" placeholder="请输入密码" size="large" :maxlength="INPUT_LIMITS.PASSWORD" />
          </a-form-item>

          <a-form-item
            v-if="settings.login_captcha_enabled"
            :label="settings.captcha_provider === 'geetest' ? '行为验证码' : '图形验证码'"
            name="captcha_code"
            :rules="settings.captcha_provider === 'geetest' ? [] : [{ required: true, message: '请输入验证码' }]"
          >
            <div v-if="settings.captcha_provider !== 'geetest'" class="captcha-row">
              <a-input v-model:value="form.captcha_code" placeholder="验证码" />
              <div class="captcha-img" @click="refreshCaptcha">
                <img v-if="captchaImage" :src="captchaImage" alt="captcha" />
                <span v-else>点击刷新</span>
              </div>
            </div>
            <div v-else class="captcha-geetest">
              <a-button @click="verifyGeeTest" :disabled="!geetest.ready" :loading="geetest.loading">
                {{ geetest.passed ? "已通过验证，点击重试" : "点击完成极验验证" }}
              </a-button>
              <span class="captcha-geetest-status">{{ geetest.passed ? "验证通过" : "未验证" }}</span>
            </div>
          </a-form-item>

          <div class="actions-row">
            <router-link to="/forgot-password">忘记密码？</router-link>
            <router-link to="/register">立即注册</router-link>
          </div>

          <a-button type="primary" html-type="submit" block size="large" :loading="auth.loading">
            登录
          </a-button>
        </a-form>
      </div>
    </div>
  </a-config-provider>
</template>

<script setup>
import { reactive, ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { message, theme } from "ant-design-vue";
import { useAuthStore } from "@/stores/auth";
import { getAuthSettings, getCaptcha } from "@/services/user";
import { INPUT_LIMITS } from "@/constants/inputLimits";

const form = reactive({
  username: "",
  phone: "",
  password: "",
  captcha_code: ""
});
const loginMode = ref("account");

const auth = useAuthStore();
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
          message.error("极验初始化失败");
        });
        resolve(true);
      }
    );
  }).catch(() => {
    message.error("极验脚本加载失败");
  });
};

const verifyGeeTest = async () => {
  if (!geetest.widget || !geetest.ready) {
    message.warning("极验尚未就绪，请稍后");
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
  if (!/^[0-9+\-\s]{6,20}$/.test(value)) return Promise.reject("请输入有效手机号");
  return Promise.resolve();
};

const onSubmit = async () => {
  try {
    const loginAccount = loginMode.value === "phone" ? String(form.phone || "").trim() : String(form.username || "").trim();
    if (loginMode.value === "phone") {
      if (loginAccount.length > INPUT_LIMITS.PHONE) {
        message.error(`手机号长度不能超过 ${INPUT_LIMITS.PHONE} 个字符`);
        return;
      }
    } else {
      if (loginAccount.length > INPUT_LIMITS.EMAIL) {
        message.error(`账号长度不能超过 ${INPUT_LIMITS.EMAIL} 个字符`);
        return;
      }
    }
    if (String(form.password || "").length > INPUT_LIMITS.PASSWORD) {
      message.error(`密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`);
      return;
    }
    if (settings.login_captcha_enabled && settings.captcha_provider === "geetest" && !geetest.passed) {
      message.warning("请先完成极验验证");
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
      message.error("登录失败");
      return;
    }
    await auth.fetchMe();
    message.success("登录成功");
    router.replace(String(route.query.redirect || "/console"));
  } catch (error) {
    const msg =
      error?.response?.data?.error ||
      error?.response?.data?.message ||
      error?.message ||
      "登录失败";
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
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: #0f172a;
}

.login-card {
  width: 100%;
  max-width: 420px;
  background: #111827;
  border: 1px solid #1f2937;
  border-radius: 14px;
  padding: 24px;
}

.title {
  color: #f8fafc;
  margin: 0;
}

.subtitle {
  color: #94a3b8;
  margin: 8px 0 18px;
}

.captcha-row {
  display: flex;
  gap: 10px;
}

.captcha-img {
  width: 120px;
  height: 40px;
  border: 1px dashed #334155;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  overflow: hidden;
  color: #94a3b8;
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
  color: #94a3b8;
  font-size: 12px;
}
</style>
