<template>
  <div class="forgot-page">
    <a-card class="forgot-card" title="找回密码">
      <a-steps :current="step - 1" size="small" style="margin-bottom: 16px">
        <a-step title="账号" />
        <a-step title="验证" />
        <a-step title="新密码" />
      </a-steps>

      <div v-if="step === 1">
        <a-form ref="step1Ref" layout="vertical" :model="step1Form" :rules="step1Rules" @finish="loadOptions">
          <a-form-item label="账户名/邮箱/手机号" name="account">
            <a-input v-model:value="step1Form.account" :maxlength="INPUT_LIMITS.EMAIL" />
          </a-form-item>
          <a-button type="primary" html-type="submit" :loading="loading" block>下一步</a-button>
        </a-form>
      </div>

      <div v-else-if="step === 2">
        <a-form ref="step2Ref" layout="vertical" :model="step2Form" :rules="step2Rules" @finish="verifyCode">
          <a-form-item label="重置方式">
            <a-radio-group v-model:value="step2Form.channel">
              <a-radio v-for="item in channels" :key="item" :value="item">{{ item === "email" ? "邮箱" : "手机号" }}</a-radio>
            </a-radio-group>
          </a-form-item>
          <a-alert
            v-if="step2Form.channel === 'sms' && smsRequiresPhoneFull"
            type="info"
            show-icon
            :message="`您的手机号是${maskedPhone || '已绑定号码'}，请补全后发送验证码`"
            style="margin-bottom: 12px"
          />
          <a-form-item v-if="step2Form.channel === 'sms' && smsRequiresPhoneFull" label="完整手机号（用于校验）" name="phone_full">
            <a-input v-model:value="step2Form.phone_full" placeholder="请输入完整手机号" :maxlength="INPUT_LIMITS.PHONE" />
          </a-form-item>
          <a-form-item label="验证码" name="code">
            <a-input v-model:value="step2Form.code" :maxlength="12" />
          </a-form-item>
          <div style="display:flex; gap:8px">
            <a-button @click="sendCode" :loading="sending">{{ sendText }}</a-button>
            <a-button type="primary" html-type="submit" :loading="loading">验证并继续</a-button>
          </div>
        </a-form>
      </div>

      <div v-else>
        <a-form ref="step3Ref" layout="vertical" :model="step3Form" :rules="step3Rules" @finish="submitReset">
          <a-form-item label="新密码" name="new_password">
            <a-input-password v-model:value="step3Form.new_password" :maxlength="INPUT_LIMITS.PASSWORD" />
          </a-form-item>
          <a-form-item label="确认密码" name="confirm_password">
            <a-input-password v-model:value="step3Form.confirm_password" :maxlength="INPUT_LIMITS.PASSWORD" />
          </a-form-item>
          <a-button type="primary" html-type="submit" :loading="loading" block>重置密码</a-button>
        </a-form>
      </div>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { message } from "ant-design-vue";
import { useRouter } from "vue-router";
import { confirmPasswordReset, getPasswordResetOptions, sendPasswordResetCode, verifyPasswordResetCode } from "@/services/user";
import { INPUT_LIMITS } from "@/constants/inputLimits";

const router = useRouter();
const step = ref(1);
const loading = ref(false);
const sending = ref(false);
const step1Ref = ref();
const step2Ref = ref();
const step3Ref = ref();
const step1Form = reactive({
  account: ""
});
const channels = ref<Array<"email" | "sms">>([]);
const maskedPhone = ref("");
const smsRequiresPhoneFull = ref(false);
const step2Form = reactive({
  channel: "email" as "email" | "sms",
  code: "",
  phone_full: ""
});
const resetTicket = ref("");
const step3Form = reactive({
  new_password: "",
  confirm_password: ""
});
const sendText = computed(() => (step2Form.channel === "email" ? "发送邮箱验证码" : "发送短信验证码"));

const step1Rules = {
  account: [
    {
      validator: () => {
        const value = String(step1Form.account || "").trim();
        if (!value) return Promise.reject("请输入账户名/邮箱/手机号");
        if (value.length > INPUT_LIMITS.EMAIL) return Promise.reject(`输入长度不能超过 ${INPUT_LIMITS.EMAIL} 个字符`);
        return Promise.resolve();
      },
      trigger: "blur"
    }
  ]
};
const step2Rules = {
  phone_full: [
    {
      validator: () => {
        if (step2Form.channel !== "sms" || !smsRequiresPhoneFull.value) return Promise.resolve();
        const v = String(step2Form.phone_full || "").trim();
        if (!v) return Promise.reject("请输入完整手机号");
        if (!/^[0-9+\-\s]{6,20}$/.test(v)) return Promise.reject("请输入有效手机号");
        return Promise.resolve();
      },
      trigger: "blur"
    }
  ],
  code: [
    { required: true, message: "请输入验证码", trigger: "blur" },
    { min: 4, max: 12, message: "验证码长度应为 4-12 位", trigger: "blur" }
  ]
};
const step3Rules = {
  new_password: [
    { required: true, message: "请输入新密码", trigger: "blur" },
    { min: 6, max: INPUT_LIMITS.PASSWORD, message: `密码长度应为 6-${INPUT_LIMITS.PASSWORD} 位`, trigger: "blur" }
  ],
  confirm_password: [
    { required: true, message: "请再次输入密码", trigger: "blur" },
    {
      validator: () => {
        if (step3Form.confirm_password !== step3Form.new_password) return Promise.reject("两次输入密码不一致");
        return Promise.resolve();
      },
      trigger: "blur"
    }
  ]
};

const loadOptions = async () => {
  try {
    await step1Ref.value?.validate();
  } catch {
    return;
  }
  loading.value = true;
  try {
    const res = await getPasswordResetOptions(step1Form.account.trim());
    const list = (res.data?.channels || []) as Array<"email" | "sms">;
    if (!list.length) {
      message.error("当前账号未绑定可用的找回方式");
      return;
    }
    channels.value = list;
    step2Form.channel = list[0];
    step2Form.code = "";
    step2Form.phone_full = "";
    maskedPhone.value = String(res.data?.masked_phone || "");
    smsRequiresPhoneFull.value = !!res.data?.sms_requires_phone_full;
    step.value = 2;
  } catch (e: any) {
    message.error(e?.response?.data?.error || "账号不存在或不可重置");
  } finally {
    loading.value = false;
  }
};

const sendCode = async () => {
  if (step2Form.channel === "sms" && smsRequiresPhoneFull.value) {
    try {
      await step2Ref.value?.validateFields(["phone_full"]);
    } catch {
      return;
    }
  }
  sending.value = true;
  try {
    await sendPasswordResetCode({
      account: step1Form.account.trim(),
      channel: step2Form.channel,
      phone_full: step2Form.phone_full.trim() || undefined
    });
    message.success("验证码已发送");
  } catch (e: any) {
    message.error(e?.response?.data?.error || "发送失败");
  } finally {
    sending.value = false;
  }
};

const verifyCode = async () => {
  try {
    await step2Ref.value?.validate();
  } catch {
    return;
  }
  loading.value = true;
  try {
    const res = await verifyPasswordResetCode({
      account: step1Form.account.trim(),
      channel: step2Form.channel,
      code: step2Form.code.trim()
    });
    resetTicket.value = String(res.data?.reset_ticket || "");
    if (!resetTicket.value) {
      message.error("获取重置票据失败");
      return;
    }
    step.value = 3;
  } catch (e: any) {
    message.error(e?.response?.data?.error || "验证码错误");
  } finally {
    loading.value = false;
  }
};

const submitReset = async () => {
  try {
    await step3Ref.value?.validate();
  } catch {
    return;
  }
  loading.value = true;
  try {
    await confirmPasswordReset({ reset_ticket: resetTicket.value, new_password: step3Form.new_password });
    message.success("密码重置成功，请登录");
    router.replace("/login");
  } catch (e: any) {
    message.error(e?.response?.data?.error || "重置失败");
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.forgot-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.forgot-card {
  width: 100%;
  max-width: 520px;
}
</style>
