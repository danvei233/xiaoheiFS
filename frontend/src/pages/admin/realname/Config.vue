<template>
  <div class="realname-config-page">
    <div class="page-header">
      <h1 class="page-title">实名认证设置</h1>
      <a-button type="primary" @click="handleSave" :loading="saving">
        保存更改
      </a-button>
    </div>

    <a-card :bordered="false">
      <a-form :model="form" layout="vertical">
        <a-form-item label="启用实名认证">
          <a-switch v-model:checked="form.enabled" />
          <span style="margin-left: 8px">开启后用户需要进行实名认证才能使用某些功能</span>
        </a-form-item>

        <a-form-item label="认证服务商">
          <a-select v-model:value="form.provider" placeholder="选择服务商">
            <a-select-option
              v-for="provider in providerOptions"
              :key="provider.key"
              :value="provider.key"
            >
              {{ provider.name }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="限制的操作">
          <a-checkbox-group v-model:value="form.block_actions">
            <a-checkbox value="create_order">创建订单</a-checkbox>
            <a-checkbox value="renew_vps">续费VPS</a-checkbox>
            <a-checkbox value="resize_vps">扩容VPS</a-checkbox>
            <a-checkbox value="wallet_recharge">钱包充值</a-checkbox>
            <a-checkbox value="wallet_withdraw">钱包提现</a-checkbox>
          </a-checkbox-group>
          <div class="form-tip">选择需要实名认证后才能进行的操作</div>
        </a-form-item>
      </a-form>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { message } from "ant-design-vue";
import { getRealNameConfig, updateRealNameConfig, listRealNameProviders } from "@/services/admin";

const saving = ref(false);
const providerOptions = ref<any[]>([]);

const form = reactive({
  enabled: false,
  provider: "",
  block_actions: [] as string[]
});

const fetchData = async () => {
  try {
    const [configRes, providersRes] = await Promise.all([
      getRealNameConfig(),
      listRealNameProviders()
    ]);

    const config = configRes.data || {};
    form.enabled = config.enabled || false;
    form.provider = config.provider || "";
    form.block_actions = config.block_actions || [];

    providerOptions.value = providersRes.data?.items || [];
  } catch (error) {
    console.error("Failed to fetch config:", error);
  }
};

const handleSave = async () => {
  saving.value = true;
  try {
    await updateRealNameConfig({
      enabled: form.enabled,
      provider: form.provider,
      block_actions: form.block_actions
    });
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
.realname-config-page {
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
