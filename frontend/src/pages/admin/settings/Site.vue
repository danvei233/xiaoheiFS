<template>
  <div class="site-settings-page">
    <div class="page-header">
      <h1 class="page-title">站点设置</h1>
      <a-button type="primary" @click="handleSave" :loading="saving">
        保存更改
      </a-button>
    </div>

    <a-card :bordered="false">
      <a-form :model="form" layout="vertical">
        <a-row :gutter="24">
          <a-col :span="12">
            <a-form-item label="站点名称">
              <a-input v-model:value="form.site_name" placeholder="小黑云控制台" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="网站URL">
              <a-input v-model:value="form.site_url" placeholder="https://example.com" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="24">
          <a-col :span="12">
            <a-form-item label="Logo URL">
              <a-input v-model:value="form.logo_url" placeholder="https://example.com/logo.png" />
              <div class="logo-preview">
                <div class="logo-preview-badge" aria-hidden="true">
                  <img v-if="form.logo_url" :src="form.logo_url" alt="logo" />
                  <DefaultLogoMark v-else :size="18" />
                </div>
                <span class="logo-preview-tip">留空则使用默认 Logo</span>
              </div>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="Favicon URL">
              <a-input v-model:value="form.favicon_url" placeholder="https://example.com/favicon.ico" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="网站描述">
          <a-textarea v-model:value="form.site_description" :rows="3" placeholder="专业的云服务提供商" />
        </a-form-item>

        <a-form-item label="关键词">
          <a-input v-model:value="form.site_keywords" placeholder="云服务器,VPS,云主机" />
        </a-form-item>

        <a-divider>联系方式</a-divider>

        <a-row :gutter="24">
          <a-col :span="12">
            <a-form-item label="公司名称">
              <a-input v-model:value="form.company_name" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="联系电话">
              <a-input v-model:value="form.contact_phone" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="24">
          <a-col :span="12">
            <a-form-item label="联系邮箱">
              <a-input v-model:value="form.contact_email" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="QQ号码">
              <a-input v-model:value="form.contact_qq" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="微信二维码">
          <a-input v-model:value="form.wechat_qrcode" placeholder="二维码图片URL" />
        </a-form-item>

        <a-divider>其他设置</a-divider>

        <a-row :gutter="24">
          <a-col :span="12">
            <a-form-item label="ICP备案号">
              <a-input v-model:value="form.icp_number" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="公安备案号">
              <a-input v-model:value="form.psbe_number" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="维护模式">
          <a-switch v-model:checked="form.maintenance_mode" />
          <span style="margin-left: 8px">启用后前台将显示维护页面</span>
        </a-form-item>

        <a-form-item label="维护提示信息">
          <a-textarea v-model:value="form.maintenance_message" :rows="2" placeholder="系统维护中，请稍后再试" />
        </a-form-item>

        <a-form-item label="统计代码">
          <a-textarea v-model:value="form.analytics_code" :rows="4" placeholder="&lt;script&gt;...&lt;/script&gt;" />
        </a-form-item>
      </a-form>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { message } from "ant-design-vue";
import { listSettings, updateSetting } from "@/services/admin";
import { useSiteStore } from "@/stores/site";
import DefaultLogoMark from "@/components/brand/DefaultLogoMark.vue";

const saving = ref(false);
const site = useSiteStore();

const form = reactive({
  site_name: "",
  site_url: "",
  logo_url: "",
  favicon_url: "",
  site_description: "",
  site_keywords: "",
  company_name: "",
  contact_phone: "",
  contact_email: "",
  contact_qq: "",
  wechat_qrcode: "",
  icp_number: "",
  psbe_number: "",
  maintenance_mode: false,
  maintenance_message: "",
  analytics_code: ""
});

const fetchData = async () => {
  try {
    const res = await listSettings();
    const items = res.data?.items || [];
    items.forEach((item: any) => {
      if (item.key in form) {
        if (item.key === "maintenance_mode") {
          form[item.key] = item.value === "true";
        } else {
          form[item.key] = item.value || "";
        }
      }
    });
  } catch (error) {
    console.error("Failed to fetch settings:", error);
  }
};

const handleSave = async () => {
  saving.value = true;
  try {
    const items = Object.entries(form).map(([key, value]) => ({
      key,
      value: typeof value === "boolean" ? (value ? "true" : "false") : String(value ?? "")
    }));
    await updateSetting({ items });
    await site.fetchSettings();
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
.site-settings-page {
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

.logo-preview {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
}

.logo-preview-badge {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  background: linear-gradient(135deg, #0ea5e9 0%, #0284c7 50%, #10b981 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  box-shadow: 0 6px 18px rgba(14, 165, 233, 0.25);
  overflow: hidden;
}

.logo-preview-badge img {
  width: 18px;
  height: 18px;
  object-fit: contain;
  display: block;
}

.logo-preview-tip {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}
</style>
