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
            <a-checkbox value="purchase_vps">购买VPS</a-checkbox>
            <a-checkbox value="renew_vps">续费VPS</a-checkbox>
            <a-checkbox value="resize_vps">扩容VPS</a-checkbox>
          </a-checkbox-group>
          <div class="form-tip">选择需要实名认证后才能进行的操作</div>
        </a-form-item>

        <template v-if="form.provider === 'mangzhu_realname'">
          <a-divider />
          <a-form-item label="芒竹 API 地址">
            <a-input v-model:value="form.mangzhu.base_url" placeholder="https://e.mangzhuyun.cn" />
          </a-form-item>
          <a-form-item label="认证模式">
            <a-select v-model:value="form.mangzhu.auth_mode">
              <a-select-option value="two_factor">二要素（姓名+身份证）</a-select-option>
              <a-select-option value="three_factor">三要素（姓名+身份证+手机）</a-select-option>
              <a-select-option value="face">面容流程（百度/微信）</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item v-if="form.mangzhu.auth_mode === 'face'" label="面容提供商">
            <a-select v-model:value="form.mangzhu.face_provider">
              <a-select-option value="baidu">百度</a-select-option>
              <a-select-option value="wechat">微信</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="API Key">
            <a-input-password v-model:value="form.mangzhu.key" placeholder="留空表示保持不变" />
            <div class="form-tip" v-if="form.mangzhu.key_set">当前已配置 Key（已脱敏）</div>
          </a-form-item>
          <a-form-item label="请求超时（秒）">
            <a-input-number v-model:value="form.mangzhu.timeout_sec" :min="1" :max="60" />
          </a-form-item>
        </template>
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
  block_actions: [] as string[],
  mangzhu: {
    base_url: "https://e.mangzhuyun.cn",
    auth_mode: "three_factor",
    face_provider: "baidu",
    key: "",
    timeout_sec: 10,
    key_set: false
  }
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
    form.mangzhu.base_url = config.mangzhu?.base_url || "https://e.mangzhuyun.cn";
    form.mangzhu.auth_mode = config.mangzhu?.auth_mode || "three_factor";
    form.mangzhu.face_provider = config.mangzhu?.face_provider || "baidu";
    form.mangzhu.timeout_sec = config.mangzhu?.timeout_sec || 10;
    form.mangzhu.key_set = !!config.mangzhu?.key_set;
    form.mangzhu.key = "";

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
      block_actions: form.block_actions,
      mangzhu: {
        base_url: form.mangzhu.base_url,
        auth_mode: form.mangzhu.auth_mode,
        face_provider: form.mangzhu.face_provider,
        key: form.mangzhu.key,
        timeout_sec: form.mangzhu.timeout_sec
      }
    });
    message.success("保存成功");
    form.mangzhu.key = "";
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
