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

        <a-form-item label="版权信息" required>
          <a-input 
            v-model:value="form.copyright_text" 
            placeholder="2023-2016 xx云服务. All rights reserved." 
            :maxlength="200"
            show-count
          />
          <div style="margin-top: 8px; font-size: 12px; color: rgba(0, 0, 0, 0.45);">
            用于设置网站底部版权文字，最多 200 字
          </div>
        </a-form-item>

        <a-form-item label="备案信息">
          <div class="beian-info-list">
            <div v-for="(beian, index) in beianInfoList" :key="index" class="beian-info-item">
              <a-button 
                v-if="beianInfoList.length > 1"
                type="text" 
                danger 
                size="small"
                @click="handleRemoveBeianInfo(index)"
                style="float: right;"
              >
                <template #icon>
                  <DeleteOutlined />
                </template>
                删除
              </a-button>
              <a-row :gutter="12">
                <a-col :span="24">
                  <div class="form-label">备案号 <span class="required-mark">*</span></div>
                  <a-input 
                    v-model:value="beian.number" 
                    placeholder="例如：京ICP备12345678号"
                  />
                </a-col>
              </a-row>
              <a-row :gutter="12" style="margin-top: 12px;">
                <a-col :span="12">
                  <div class="form-label">备案图标 URL</div>
                  <a-input 
                    v-model:value="beian.icon_url" 
                    placeholder="例如：https://example.com/icon.png"
                  />
                </a-col>
                <a-col :span="12">
                  <div class="form-label">备案跳转链接</div>
                  <a-input 
                    v-model:value="beian.link_url" 
                    placeholder="例如：https://beian.miit.gov.cn/"
                  />
                </a-col>
              </a-row>
            </div>
          </div>
          <a-button 
            type="dashed" 
            block 
            @click="handleAddBeianInfo"
            style="margin-top: 12px;"
          >
            <template #icon>
              <PlusOutlined />
            </template>
            添加备案信息
          </a-button>
        </a-form-item>

        <a-divider>安全设置</a-divider>

        <a-form-item label="管理端路径">
          <a-input-group compact>
            <a-input 
              v-model:value="form.admin_path" 
              placeholder="admin" 
              style="width: calc(100% - 32px)"
            />
            <a-tooltip title="随机生成">
              <a-button @click="handleRefreshAdminPath" :loading="refreshing">
                <template #icon>
                  <ReloadOutlined />
                </template>
              </a-button>
            </a-tooltip>
          </a-input-group>
          <div style="margin-top: 8px; font-size: 12px; color: rgba(0, 0, 0, 0.45);">
            自定义管理后台访问路径，修改后将自动跳转到新路径（仅支持字母和数字）
          </div>
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
import { ReloadOutlined, PlusOutlined, DeleteOutlined } from "@ant-design/icons-vue";
import { useRouter } from "vue-router";
import { listSettings, updateSetting } from "@/services/admin";
import { useSiteStore } from "@/stores/site";
import { clearAdminPathCache, getCachedAdminPath } from "@/services/adminPath";
import DefaultLogoMark from "@/components/brand/DefaultLogoMark.vue";

const saving = ref(false);
const refreshing = ref(false);
const site = useSiteStore();
const router = useRouter();

// 当前年份
const currentYear = new Date().getFullYear();

// 记录原始的 admin_path 值，用于检测变化
const originalAdminPath = ref("");

// 系统预设的配置项（不可删除）
const systemSettings = [
  { key: "icp_number", label: "ICP备案号", placeholder: "如：京ICP备12345678号" },
  { key: "psbe_number", label: "公安备案号", placeholder: "如：京公网安备11010802012345号" },
  { key: "maintenance_mode", label: "维护模式", placeholder: "true/false" },
  { key: "maintenance_message", label: "维护提示信息", placeholder: "系统维护中，请稍后再试" }
];

// 动态配置项列表
const customSettings = ref<Array<{
  key: string;
  label: string;
  value: string;
  placeholder?: string;
  isSystem?: boolean;
}>>([]);

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
  analytics_code: "",
  admin_path: "",
  copyright_text: ""
});

// 备案信息列表
const beianInfoList = ref<Array<{
  number: string;
  icon_url: string;
  link_url: string;
}>>([
  {
    number: "",
    icon_url: "",
    link_url: ""
  }
]);

// 添加备案信息
const handleAddBeianInfo = () => {
  beianInfoList.value.push({
    number: "",
    icon_url: "",
    link_url: ""
  });
};

// 删除备案信息
const handleRemoveBeianInfo = (index: number) => {
  beianInfoList.value.splice(index, 1);
};

// 生成随机管理路径（与后端逻辑一致）
const generateRandomAdminPath = (): string => {
  const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
  const length = 12;
  let result = "";
  
  // 使用 crypto.getRandomValues 生成随机数
  const randomValues = new Uint8Array(length);
  crypto.getRandomValues(randomValues);
  
  for (let i = 0; i < length; i++) {
    result += charset[randomValues[i] % charset.length];
  }
  
  return result;
};

const handleRefreshAdminPath = () => {
  refreshing.value = true;
  try {
    const newPath = generateRandomAdminPath();
    form.admin_path = newPath;
    message.success("已生成新的管理路径");
  } finally {
    refreshing.value = false;
  }
};

const fetchData = async () => {
  try {
    const res = await listSettings();
    const items = res.data?.items || [];
    
    // 用于存储已处理的键
    const processedKeys = new Set<string>();
    
    // 处理表单字段
    items.forEach((item: any) => {
      if (item.key in form) {
        form[item.key] = item.value || "";
        processedKeys.add(item.key);
      }
    });
    
    // 处理备案信息列表
    const beianInfoItem = items.find((i: any) => i.key === "beian_info_list");
    if (beianInfoItem && beianInfoItem.value) {
      try {
        const parsed = JSON.parse(beianInfoItem.value);
        if (Array.isArray(parsed) && parsed.length > 0) {
          beianInfoList.value = parsed;
        }
      } catch (e) {
        console.error("Failed to parse beian_info_list:", e);
      }
    }
    processedKeys.add("beian_info_list");
    
    // 记录原始的 admin_path 值
    originalAdminPath.value = form.admin_path || "admin";
  } catch (error) {
    console.error("Failed to fetch settings:", error);
  }
};

const handleSave = async () => {
  // 验证版权信息
  if (!form.copyright_text || form.copyright_text.trim() === "") {
    message.error("版权信息不能为空");
    return;
  }
  
  if (form.copyright_text.length > 200) {
    message.error("版权信息不能超过 200 字");
    return;
  }
  
  // 验证备案信息（如果有填写的话，备案号不能为空）
  for (let i = 0; i < beianInfoList.value.length; i++) {
    const beian = beianInfoList.value[i];
    // 如果填写了任何一个字段，则备案号必填
    if ((beian.icon_url || beian.link_url) && (!beian.number || beian.number.trim() === "")) {
      message.error(`备案信息 ${i + 1} 的备案号不能为空`);
      return;
    }
  }
  
  saving.value = true;
  try {
    // 合并表单数据和自定义配置项
    const items = [
      // 表单字段
      ...Object.entries(form).map(([key, value]) => ({
        key,
        value: String(value ?? "")
      })),
      // 备案信息列表（过滤掉空的备案信息）
      {
        key: "beian_info_list",
        value: JSON.stringify(
          beianInfoList.value.filter(beian => 
            beian.number && beian.number.trim() !== ""
          )
        )
      }
    ];
    
    await updateSetting({ items });
    await site.fetchSettings();
    
    // 检测 admin_path 是否发生变化
    const newAdminPath = form.admin_path || "admin";
    const oldAdminPath = originalAdminPath.value || "admin";
    
    if (newAdminPath !== oldAdminPath) {
      // 管理端路径发生变化
      message.success("保存成功，管理路径已更改，正在跳转到新路径...", 2);
      
      // 清除旧的缓存
      clearAdminPathCache();
      
      // 缓存新路径
      try {
        localStorage.setItem("admin_path_cache", newAdminPath);
        localStorage.setItem("admin_path_validated", "true");
      } catch (e) {
        console.error("Failed to cache new admin path:", e);
      }
      
      // 延迟跳转，让用户看到成功提示
      setTimeout(() => {
        // 跳转到新的管理路径
        router.replace(`/${newAdminPath}/settings/site`);
      }, 2000);
    } else {
      // 路径没有变化，正常提示
      message.success("保存成功");
    }
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

.beian-info-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.beian-info-item {
  padding: 16px;
  background: #fafafa;
  border-radius: 8px;
  border: 1px solid #e8e8e8;
  transition: all 0.3s;
}

.beian-info-item:hover {
  background: #f5f5f5;
  border-color: #d9d9d9;
}

.form-label {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.85);
  margin-bottom: 8px;
}

.required-mark {
  color: #ff4d4f;
  margin-left: 4px;
}

</style>
