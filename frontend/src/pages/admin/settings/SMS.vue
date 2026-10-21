<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">短信设置</div>
        <div class="subtle">短信插件选择、模板管理、预览与测试发送</div>
      </div>
      <div class="page-header-actions">
        <a-button type="primary" @click="saveConfig" :loading="saving">保存配置</a-button>
      </div>
    </div>

    <a-card class="card">
      <a-form layout="vertical">
        <a-row :gutter="12">
          <a-col :span="8">
            <a-form-item label="启用短信模块">
              <a-switch v-model:checked="config.enabled" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="短信插件实例">
              <a-select v-model:value="selectedBinding" allow-clear placeholder="请选择短信插件实例" @change="onBindingChange">
                <a-select-option v-for="opt in smsPluginOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="供应商模板ID（可选）">
              <a-input v-model:value="config.provider_template_id" placeholder="如阿里云 TemplateCode" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8">
            <a-form-item label="默认内容模板">
              <a-select v-model:value="config.default_template_id" allow-clear placeholder="选择默认模板">
                <a-select-option v-for="item in templates" :key="String(item.id)" :value="String(item.id)">
                  {{ item.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="16">
            <a-form-item label="快速测试手机号（支持逗号分隔）">
              <div class="test-row">
                <a-input v-model:value="quickTestPhone" placeholder="13800138000" />
                <a-button @click="quickTest" type="primary">发送测试</a-button>
              </div>
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-card>

    <a-card class="card" style="margin-top: 16px">
      <div class="page-header" style="margin-bottom: 12px">
        <div class="section-title">短信模板</div>
        <a-button type="primary" @click="openTemplate()">新增模板</a-button>
      </div>
      <a-table :columns="columns" :data-source="templates" row-key="id" :pagination="false" :scroll="{ x: 980 }">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'enabled'">
            <a-tag :color="record.enabled ? 'green' : 'red'">{{ record.enabled ? '启用' : '停用' }}</a-tag>
          </template>
          <template v-else-if="column.key === 'content'">
            <div class="ellipsis">{{ record.content }}</div>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button size="small" @click="openTemplate(record)">编辑</a-button>
              <a-button size="small" danger @click="removeTemplate(record)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:open="templateOpen" title="短信模板" width="860px" @ok="saveTemplate">
      <a-form layout="vertical">
        <a-row :gutter="12">
          <a-col :span="16"><a-form-item label="名称"><a-input v-model:value="templateForm.name" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="启用"><a-switch v-model:checked="templateForm.enabled" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="内容模板">
          <a-textarea v-model:value="templateForm.content" :rows="6" placeholder="例如：您的验证码是 {{code}}，请勿泄露。" />
          <div class="subtle" style="margin-top: 6px">支持变量：{{code}} / {{phone}} / {{now}}</div>
        </a-form-item>

        <a-collapse ghost>
          <a-collapse-panel key="preview" header="预览">
            <div class="test-row">
              <a-input v-model:value="previewPhone" placeholder="用于预览的手机号" />
              <a-input v-model:value="previewCode" placeholder="用于预览的验证码" />
              <a-button @click="previewTemplate">生成预览</a-button>
            </div>
            <pre class="preview-block">{{ previewContent }}</pre>
          </a-collapse-panel>
          <a-collapse-panel key="test" header="测试发送">
            <div class="test-row">
              <a-input v-model:value="templateTestPhone" placeholder="测试手机号，支持逗号分隔" />
              <a-button type="primary" @click="testTemplate">发送测试</a-button>
            </div>
          </a-collapse-panel>
        </a-collapse>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { message, Modal } from "ant-design-vue";
import {
  deleteSmsTemplate,
  getSmsConfig,
  listAdminPlugins,
  listSmsTemplates,
  previewSmsConfig,
  testSmsConfig,
  updateSmsConfig,
  updateSmsTemplate,
  upsertSmsTemplate,
} from "@/services/admin";

const saving = ref(false);
const selectedBinding = ref<string | undefined>(undefined);
const quickTestPhone = ref("");

const config = reactive({
  enabled: true,
  plugin_id: "",
  instance_id: "default",
  default_template_id: "",
  provider_template_id: "",
});

const templates = ref<Array<Record<string, any>>>([]);
const templateOpen = ref(false);
const templateForm = reactive({ id: null as number | null, name: "", content: "", enabled: true });
const previewPhone = ref("13800138000");
const previewCode = ref("123456");
const previewContent = ref("");
const templateTestPhone = ref("");

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 70 },
  { title: "名称", dataIndex: "name", key: "name", width: 200 },
  { title: "内容", dataIndex: "content", key: "content", width: 470, ellipsis: true },
  { title: "启用", dataIndex: "enabled", key: "enabled", width: 90 },
  { title: "操作", key: "action", width: 150 },
];

const pluginItems = ref<Array<Record<string, any>>>([]);
const smsPluginOptions = computed(() => {
  return pluginItems.value
    .filter((item) => (item.category || "") === "sms" && item.enabled === true && item.loaded === true)
    .map((item) => {
      const pluginId = String(item.plugin_id || "");
      const instanceId = String(item.instance_id || "default");
      const name = String(item.name || pluginId);
      return {
        value: `${pluginId}::${instanceId}`,
        label: `${name} (${pluginId}/${instanceId})`,
      };
    });
});

const onBindingChange = (value?: string) => {
  if (!value) {
    config.plugin_id = "";
    config.instance_id = "";
    return;
  }
  const [pluginId, instanceId] = String(value).split("::");
  config.plugin_id = pluginId || "";
  config.instance_id = instanceId || "default";
};

const loadConfig = async () => {
  const [configRes, pluginRes] = await Promise.all([getSmsConfig(), listAdminPlugins()]);
  const data = configRes.data || {};
  config.enabled = data.enabled !== false;
  config.plugin_id = String(data.plugin_id || "");
  config.instance_id = String(data.instance_id || "default");
  config.default_template_id = String(data.default_template_id || "");
  config.provider_template_id = String(data.provider_template_id || "");
  pluginItems.value = pluginRes.data?.items || [];
  if (config.plugin_id) {
    selectedBinding.value = `${config.plugin_id}::${config.instance_id || "default"}`;
  }
};

const loadTemplates = async () => {
  const res = await listSmsTemplates();
  templates.value = res.data?.items || [];
};

const saveConfig = async () => {
  saving.value = true;
  try {
    await updateSmsConfig({
      enabled: config.enabled,
      plugin_id: config.plugin_id,
      instance_id: config.instance_id || "default",
      default_template_id: config.default_template_id,
      provider_template_id: config.provider_template_id,
    });
    message.success("短信配置已保存");
  } finally {
    saving.value = false;
  }
};

const openTemplate = (row?: Record<string, any>) => {
  if (row) {
    templateForm.id = Number(row.id || 0) || null;
    templateForm.name = String(row.name || "");
    templateForm.content = String(row.content || "");
    templateForm.enabled = row.enabled !== false;
  } else {
    templateForm.id = null;
    templateForm.name = "";
    templateForm.content = "";
    templateForm.enabled = true;
  }
  previewContent.value = "";
  templateTestPhone.value = "";
  templateOpen.value = true;
};

const saveTemplate = async () => {
  const payload = {
    name: templateForm.name,
    content: templateForm.content,
    enabled: templateForm.enabled,
  };
  if (templateForm.id) {
    await updateSmsTemplate(templateForm.id, payload);
  } else {
    await upsertSmsTemplate(payload);
  }
  message.success("模板已保存");
  templateOpen.value = false;
  await loadTemplates();
};

const removeTemplate = (row: Record<string, any>) => {
  Modal.confirm({
    title: "确认删除该短信模板？",
    onOk: async () => {
      await deleteSmsTemplate(row.id);
      message.success("已删除");
      await loadTemplates();
    },
  });
};

const previewTemplate = async () => {
  if (!templateForm.content.trim()) {
    message.error("请输入模板内容");
    return;
  }
  const res = await previewSmsConfig({
    content: templateForm.content,
    variables: {
      code: previewCode.value,
      phone: previewPhone.value,
    },
  });
  previewContent.value = res.data?.content || "";
};

const testTemplate = async () => {
  const phone = templateTestPhone.value.trim();
  if (!phone) {
    message.error("请输入测试手机号");
    return;
  }
  const payload: Record<string, any> = {
    phone,
    plugin_id: config.plugin_id,
    instance_id: config.instance_id || "default",
    provider_template_id: config.provider_template_id,
  };
  if (templateForm.id) {
    payload.template_id = templateForm.id;
  } else {
    const content = templateForm.content.trim();
    if (!content) {
      message.error("请先填写模板内容或先保存模板");
      return;
    }
    payload.content = content;
  }
  payload.variables = {
    code: previewCode.value || "123456",
    phone: previewPhone.value || phone.split(",")[0],
  };
  await testSmsConfig(payload);
  message.success("测试短信已发送");
};

const quickTest = async () => {
  const phone = quickTestPhone.value.trim();
  if (!phone) {
    message.error("请输入测试手机号");
    return;
  }
  const selectedTemplateID = Number(config.default_template_id || 0) || 0;
  const fallbackTemplate = templates.value.find((item) => item.enabled !== false);
  const fallbackTemplateID = Number(fallbackTemplate?.id || 0) || 0;
  const templateID = selectedTemplateID || fallbackTemplateID;
  if (!templateID) {
    message.error("请先在短信设置中选择默认模板，或先新增并启用一个模板");
    return;
  }
  await testSmsConfig({
    phone,
    template_id: templateID,
    plugin_id: config.plugin_id,
    instance_id: config.instance_id || "default",
    provider_template_id: config.provider_template_id,
  });
  message.success("测试短信已发送");
};

onMounted(async () => {
  await Promise.all([loadConfig(), loadTemplates()]);
});
</script>

<style scoped>
.test-row {
  display: flex;
  gap: 12px;
}

.test-row :deep(.ant-input) {
  flex: 1;
}

.preview-block {
  margin-top: 12px;
  padding: 10px;
  border-radius: 6px;
  background: #f5f5f5;
  white-space: pre-wrap;
}

.ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
