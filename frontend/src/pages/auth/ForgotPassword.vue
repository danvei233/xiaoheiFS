<template>
  <div class="forgot-page">
    <a-card class="forgot-card" :title="t('auth.forgotTitle')">
      <a-steps :current="step - 1" size="small" style="margin-bottom: 16px">
        <a-step :title="t('auth.stepAccount')" />
        <a-step :title="t('auth.stepVerify')" />
        <a-step :title="t('auth.stepPassword')" />
      </a-steps>

      <div v-if="step === 1">
        <a-form ref="step1Ref" layout="vertical" :model="step1Form" :rules="step1Rules" @finish="loadOptions">
          <a-form-item :label="t('auth.accountLabel')" name="account">
            <a-input v-model:value="step1Form.account" :maxlength="INPUT_LIMITS.EMAIL" />
          </a-form-item>
          <a-button type="primary" html-type="submit" :loading="loading" block>{{ t('common.next') }}</a-button>
        </a-form>
      </div>

      <div v-else-if="step === 2">
        <a-form ref="step2Ref" layout="vertical" :model="step2Form" :rules="step2Rules" @finish="verifyCode">
          <a-form-item :label="t('auth.resetMethod')">
            <a-radio-group v-model:value="step2Form.channel">
              <a-radio v-for="item in channels" :key="item" :value="item">{{ item === "email" ? t('common.email') : t('common.phone') }}</a-radio>
            </a-radio-group>
          </a-form-item>
          <a-alert
            v-if="step2Form.channel === 'sms' && smsRequiresPhoneFull"
            type="info"
            show-icon
            :message="t('auth.phoneMaskedHint', { masked: maskedPhone || t('auth.boundPhone') })"
            style="margin-bottom: 12px"
          />
          <a-form-item v-if="step2Form.channel === 'sms' && smsRequiresPhoneFull" :label="t('auth.phoneFull')" name="phone_full">
            <a-input v-model:value="step2Form.phone_full" :placeholder="t('auth.phoneFullPlaceholder')" :maxlength="INPUT_LIMITS.PHONE" />
          </a-form-item>
          <a-form-item :label="t('common.code')" name="code">
            <a-input v-model:value="step2Form.code" :maxlength="12" />
          </a-form-item>
          <div style="display:flex; gap:8px">
            <a-button @click="sendCode" :loading="sending">{{ sendText }}</a-button>
            <a-button type="primary" html-type="submit" :loading="loading">{{ t('auth.verifyAndContinue') }}</a-button>
          </div>
        </a-form>
      </div>

      <div v-else>
        <a-form ref="step3Ref" layout="vertical" :model="step3Form" :rules="step3Rules" @finish="submitReset">
          <a-form-item :label="t('auth.newPassword')" name="new_password">
            <a-input-password v-model:value="step3Form.new_password" :maxlength="INPUT_LIMITS.PASSWORD" />
          </a-form-item>
          <a-form-item :label="t('auth.confirmPassword')" name="confirm_password">
            <a-input-password v-model:value="step3Form.confirm_password" :maxlength="INPUT_LIMITS.PASSWORD" />
          </a-form-item>
          <a-button type="primary" html-type="submit" :loading="loading" block>{{ t('auth.resetPassword') }}</a-button>
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
import { t } from "@/locales";

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
const sendText = computed(() => (step2Form.channel === "email" ? t('auth.sendEmailCode') : t('auth.sendSmsCode')));

const step1Rules = {
  account: [
    {
      validator: () => {
        const value = String(step1Form.account || "").trim();
        if (!value) return Promise.reject(t('auth.accountRequiredMsg'));
        if (value.length > INPUT_LIMITS.EMAIL) return Promise.reject(t('auth.inputTooLong', { n: INPUT_LIMITS.EMAIL }));
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
        if (!v) return Promise.reject(t('auth.phoneFullPlaceholder'));
        if (!/^[0-9+\-\s]{6,20}$/.test(v)) return Promise.reject(t('auth.invalidPhone'));
        return Promise.resolve();
      },
      trigger: "blur"
    }
  ],
  code: [
    { required: true, message: t('auth.captchaRequired'), trigger: "blur" },
    { min: 4, max: 12, message: t('auth.codeLength'), trigger: "blur" }
  ]
};
const step3Rules = {
  new_password: [
    { required: true, message: t('auth.newPasswordRequired'), trigger: "blur" },
    { min: 6, max: INPUT_LIMITS.PASSWORD, message: t('auth.passwordRange', { n: INPUT_LIMITS.PASSWORD }), trigger: "blur" }
  ],
  confirm_password: [
    { required: true, message: t('auth.confirmRequired'), trigger: "blur" },
    {
      validator: () => {
        if (step3Form.confirm_password !== step3Form.new_password) return Promise.reject(t('auth.passwordMismatch'));
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
    message.error(e?.response?.data?.error || t('auth.accountNotFound'));
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
    message.success(t('auth.codeSent'));
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('auth.sendFailedShort'));
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
      message.error(t('auth.resetTicketFailed'));
      return;
    }
    step.value = 3;
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('auth.codeIncorrect'));
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
    message.success(t('auth.resetSuccess'));
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
