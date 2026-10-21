<template>
  <NotFound v-if="showNotFound" />
  <div v-else class="install-wizard">
    <div class="wizard-container">
      <!-- 步骤指示器 -->
      <div class="steps-indicator">
        <div 
          v-for="(step, index) in steps" 
          :key="index"
          class="step-item"
          :class="{ active: currentStep === index, completed: currentStep > index }"
        >
          <div class="step-number">
            <span v-if="currentStep > index">✓</span>
            <span v-else>{{ index + 1 }}</span>
          </div>
          <div class="step-label">{{ step.label }}</div>
        </div>
      </div>

      <!-- 步骤内容 -->
      <div class="step-content">
        <!-- 步骤 1: 数据库配置 -->
        <DbStep v-if="currentStep === 0" @next="handleDbNext" />
        
        <!-- 步骤 2: 站点信息 -->
        <SiteStep v-if="currentStep === 1" @next="handleSiteNext" @back="currentStep = 0" />
        
        <!-- 步骤 3: 管理员设置 -->
        <AdminStep v-if="currentStep === 2" @next="handleAdminNext" @back="currentStep = 1" />
        
        <!-- 步骤 4: 完成 -->
        <DoneStep 
          v-if="currentStep === 3" 
          :admin-path="doneData.adminPath"
          :restart="doneData.restart"
          :config-file="doneData.configFile"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { useInstallStore } from "@/stores/install";
import NotFound from "@/pages/public/NotFound.vue";
import DbStep from "./DbStep.vue";
import SiteStep from "./SiteStep.vue";
import AdminStep from "./AdminStep.vue";
import DoneStep from "./DoneStep.vue";

const install = useInstallStore();
const currentStep = ref(0);
const showNotFound = ref(false);

const steps = [
  { label: "数据库配置" },
  { label: "站点信息" },
  { label: "管理员设置" },
  { label: "完成" }
];

const doneData = reactive({
  adminPath: "admin",
  restart: false,
  configFile: ""
});

const handleDbNext = () => {
  currentStep.value = 1;
};

const handleSiteNext = () => {
  currentStep.value = 2;
};

const handleAdminNext = (adminPath: string, restart: boolean, configFile: string) => {
  doneData.adminPath = adminPath;
  doneData.restart = restart;
  doneData.configFile = configFile;
  currentStep.value = 3;
};

onMounted(async () => {
  // 检查是否已安装
  if (!install.loaded) {
    await install.fetchStatus();
  }
  
  // 如果已安装，显示 404
  if (install.installed) {
    showNotFound.value = true;
  }
});
</script>

<style scoped>
.install-wizard {
  min-height: 100vh;
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
  padding: 40px 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.wizard-container {
  width: 100%;
  max-width: 900px;
}

/* 步骤指示器 */
.steps-indicator {
  display: flex;
  justify-content: space-between;
  margin-bottom: 40px;
  position: relative;
}

.steps-indicator::before {
  content: "";
  position: absolute;
  top: 20px;
  left: 0;
  right: 0;
  height: 2px;
  background: rgba(71, 85, 105, 0.3);
  z-index: 0;
}

.step-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  position: relative;
  z-index: 1;
  flex: 1;
}

.step-number {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(30, 41, 59, 0.8);
  border: 2px solid rgba(71, 85, 105, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.5);
  transition: all 0.3s ease;
}

.step-item.active .step-number {
  background: linear-gradient(135deg, #22d3ee 0%, #06b6d4 100%);
  border-color: #22d3ee;
  color: #0f172a;
  box-shadow: 0 4px 20px rgba(34, 211, 238, 0.4);
}

.step-item.completed .step-number {
  background: rgba(34, 211, 238, 0.2);
  border-color: #22d3ee;
  color: #22d3ee;
}

.step-label {
  font-size: 13px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.5);
  text-align: center;
  transition: color 0.3s ease;
}

.step-item.active .step-label {
  color: #22d3ee;
}

.step-item.completed .step-label {
  color: rgba(34, 211, 238, 0.8);
}

/* 步骤内容 */
.step-content {
  background: rgba(15, 23, 42, 0.6);
  border-radius: 12px;
  border: 1px solid rgba(71, 85, 105, 0.3);
  padding: 32px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
}

/* 确保子组件的表单样式正确 */
:deep(.ant-form) {
  color: #ffffff;
}

:deep(.ant-form-item-label > label) {
  color: rgba(255, 255, 255, 0.7);
}

:deep(.ant-input),
:deep(.ant-input-password),
:deep(.ant-input-number) {
  background: rgba(15, 23, 42, 0.6);
  border-color: rgba(71, 85, 105, 0.5);
  color: #ffffff;
}

:deep(.ant-input::placeholder) {
  color: rgba(255, 255, 255, 0.3);
}

:deep(.ant-btn) {
  font-weight: 600;
}

/* 响应式 */
@media (max-width: 768px) {
  .install-wizard {
    padding: 20px 12px;
  }

  .steps-indicator {
    margin-bottom: 24px;
  }

  .step-number {
    width: 32px;
    height: 32px;
    font-size: 14px;
  }

  .step-label {
    font-size: 11px;
  }

  .step-content {
    padding: 20px;
  }
}
</style>

