<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">邮件与模板</div>
        <div class="subtle">SMTP 配置与模板管理</div>
      </div>
      <div class="page-header-actions">
        <a-button type="primary" @click="save">保存配置</a-button>
      </div>
    </div>

    <a-card class="card">
      <a-form layout="vertical">
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="SMTP Host"><a-input v-model:value="form.smtp_host" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="SMTP Port"><a-input v-model:value="form.smtp_port" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="SMTP User"><a-input v-model:value="form.smtp_user" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="SMTP Pass"><a-input v-model:value="form.smtp_pass" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="SMTP From"><a-input v-model:value="form.smtp_from" /></a-form-item>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="启用 SMTP"><a-switch v-model:checked="form.smtp_enabled" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="启用邮件"><a-switch v-model:checked="form.email_enabled" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="到期提醒邮件"><a-switch v-model:checked="form.email_expire_enabled" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="到期提醒天数"><a-input-number v-model:value="form.expire_reminder_days" :min="1" /></a-form-item></a-col>
        </a-row>
        <a-divider />
        <a-form-item label="SMTP 测试">
          <div class="test-row">
            <a-input v-model:value="smtpTestTo" placeholder="接收人邮箱" />
            <a-button type="primary" @click="sendSmtpTest">测试发送</a-button>
          </div>
          <div class="subtle" style="margin-top: 6px">
            未启用模板时将发送默认文案；如有启用模板，将优先发送列表中的第一个启用模板。
          </div>
        </a-form-item>
      </a-form>
    </a-card>

    <a-card class="card" style="margin-top: 16px">
      <div class="page-header" style="margin-bottom: 12px">
        <div class="section-title">邮件模板</div>
        <a-button type="primary" @click="openTemplate">新增模板</a-button>
      </div>
      <a-table :columns="columns" :data-source="templates" row-key="id" :pagination="false">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'enabled'">
            <a-tag :color="record.enabled ? 'green' : 'red'">{{ record.enabled ? '启用' : '停用' }}</a-tag>
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

    <a-modal v-model:open="templateOpen" title="邮件模板" width="80vw" :style="{ maxWidth: '1000px' }" @ok="saveTemplate">
      <a-form layout="vertical">
        <div class="template-toolbar">
          <a-space>
            <a-select v-model:value="defaultTemplateKey" allow-clear placeholder="默认模板" style="width: 220px">
              <a-select-option v-for="item in defaultTemplateOptions" :key="item.key" :value="item.key">
                {{ item.label }}
              </a-select-option>
            </a-select>
            <a-button @click="applyDefaultTemplate">填充默认</a-button>
          </a-space>
          <a-space>
            <a-button @click="openPreview">预览</a-button>
            <a-button type="link" size="small" @click="openHelp">
              <template #icon><span style="font-size: 14px;">?</span></template>
              帮助
            </a-button>
          </a-space>
        </div>
        <a-row :gutter="12">
          <a-col :span="16"><a-form-item label="名称"><a-input v-model:value="templateForm.name" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="启用"><a-switch v-model:checked="templateForm.enabled" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="主题"><a-input v-model:value="templateForm.subject" /></a-form-item>
        <a-form-item label="内容">
          <div class="editor-hint">
            <span>提示：点击工具栏的 <span style="font-family: monospace; background: #f0f0f0; padding: 2px 6px; border-radius: 3px;">&lt;/&gt;</span> 按钮切换可视化/HTML 模式</span>
          </div>
          <RichTextEditor v-model="templateForm.body" :height="400" />
        </a-form-item>
        <a-collapse ghost>
          <a-collapse-panel key="vars" header="模板变量与 Mock 数据">
            <a-alert type="info" show-icon>
              <template #message>变量示例</template>
              <template #description>
                <div class="template-vars">
                  <div><code v-pre>{{ .user.id }}</code>：{{ mockData.user.id }}</div>
                  <div><code v-pre>{{ .user.username }}</code>：{{ mockData.user.username }}</div>
                  <div><code v-pre>{{ .user.email }}</code>：{{ mockData.user.email }}</div>
                  <div><code v-pre>{{ .user.qq }}</code>：{{ mockData.user.qq }}</div>
                  <div><code v-pre>{{ .order.no }}</code>：{{ mockData.order.no }}</div>
                  <div><code v-pre>{{ .vps.name }}</code>：{{ mockData.vps.name }}</div>
                  <div><code v-pre>{{ .vps.expire_at }}</code>：{{ mockData.vps.expire_at }}</div>
                  <div><code v-pre>{{ .message }}</code>：{{ mockData.message }}</div>
                </div>
              </template>
            </a-alert>
            <div style="margin-top: 12px">
              <div class="subtle">渲染数据：</div>
              <pre class="mock-data">{{ mockDataJson }}</pre>
            </div>
          </a-collapse-panel>
          <a-collapse-panel key="test" header="模板测试">
            <div class="test-row">
              <a-input v-model:value="templateTestTo" placeholder="test@example.com" />
              <a-button type="primary" @click="sendTest">发送测试</a-button>
            </div>
            <div class="subtle" style="margin-top: 6px">测试会使用上方模板内容 + mock 数据渲染变量</div>
          </a-collapse-panel>
        </a-collapse>
      </a-form>
    </a-modal>

    <a-modal v-model:open="previewOpen" title="模板预览" width="80vw" :style="{ maxWidth: '760px' }" :footer="null">
      <div class="preview-block">
        <div class="preview-title">主题</div>
        <div class="preview-subject">{{ previewSubject }}</div>
      </div>
      <div class="preview-block">
        <div class="preview-title">正文</div>
        <div v-if="previewIsHtml" class="preview-body" v-html="previewBody" />
        <pre v-else class="preview-text">{{ previewBody }}</pre>
      </div>
      <a-divider />
      <div class="preview-title">Mock 数据</div>
      <pre class="mock-data">{{ mockDataJson }}</pre>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import RichTextEditor from "@/components/RichTextEditor/RichTextEditor.vue";
import {
  listSettings,
  updateSetting,
  listEmailTemplates,
  upsertEmailTemplate,
  updateEmailTemplate,
  deleteEmailTemplate,
  getSmtpConfig,
  updateSmtpConfig,
  testSmtpConfig
} from "@/services/admin";
import { message, Modal } from "ant-design-vue";

const form = reactive({
  smtp_host: "",
  smtp_port: "",
  smtp_user: "",
  smtp_pass: "",
  smtp_from: "",
  smtp_enabled: false,
  email_enabled: false,
  email_expire_enabled: false,
  expire_reminder_days: 7,
});

const templates = ref([]);
const templateOpen = ref(false);
const templateForm = reactive({ id: null, name: "", subject: "", body: "", enabled: true });
const templateTestTo = ref("");
const smtpTestTo = ref("");
const previewOpen = ref(false);
const previewSubject = ref("");
const previewBody = ref("");
const previewIsHtml = ref(false);
const defaultTemplateKey = ref(null);


const columns = [
  { title: "模板 ID", dataIndex: "id", key: "id" },
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "主题", dataIndex: "subject", key: "subject" },
  { title: "启用", dataIndex: "enabled", key: "enabled" },
  { title: "操作", key: "action" }
];


const DEFAULT_TEMPLATES = [
  {
    key: "provision_success",
    label: "开通成功 (provision_success)",
    subject: "VPS Provisioned: Order {{.order.no}}",
    body: `<!DOCTYPE html>
<html>
<body style="margin:0; padding:24px; background:#f4f6fb; font-family: Arial, sans-serif; color:#1f2329;">
  <div style="max-width:640px; margin:0 auto;">
    <div style="font-size:12px; color:#6b7280;">Provision Notice</div>
    <div style="font-size:20px; font-weight:700;">VPS Provisioned</div>
    <div style="height:12px;"></div>
    <div style="background:#ffffff; border-radius:12px; box-shadow:0 8px 20px rgba(15,23,42,0.08); padding:24px;">
      <div style="display:inline-block; padding:6px 10px; background:#eef2ff; color:#4338ca; border-radius:999px; font-size:12px; font-weight:600;">Active</div>
      <h2 style="margin:12px 0 8px; font-size:18px;">Hi {{.user.username}},</h2>
      <p style="margin:0 0 12px;">Your VPS for order <strong>{{.order.no}}</strong> is now active.</p>
      <div style="background:#f8fafc; border-radius:10px; padding:12px;">
        <div style="font-size:12px; color:#6b7280;">Next step</div>
        <div style="font-size:14px; font-weight:600; padding-top:4px;">Log in to the console to manage your instance.</div>
      </div>
      <p style="margin:16px 0 0; font-size:13px; color:#6b7280;">If you have any questions, reply to this email.</p>
    </div>
    <div style="padding-top:12px; font-size:12px; color:#94a3b8;">This is an automated message.</div>
  </div>
</body>
</html>`
  },
  {
    key: "expire_reminder",
    label: "到期提醒 (expire_reminder)",
    subject: "VPS Expiration Reminder: {{.vps.name}}",
    body: `<!DOCTYPE html>
<html>
<body style="margin:0; padding:24px; background:#fff7ed; font-family: Arial, sans-serif; color:#1f2329;">
  <div style="max-width:640px; margin:0 auto;">
    <div style="font-size:12px; color:#9a3412;">Reminder</div>
    <div style="font-size:20px; font-weight:700;">VPS Expiration Alert</div>
    <div style="height:12px;"></div>
    <div style="background:#ffffff; border-radius:12px; box-shadow:0 8px 20px rgba(180,83,9,0.08); padding:24px;">
      <div style="display:inline-block; padding:6px 10px; background:#ffedd5; color:#9a3412; border-radius:999px; font-size:12px; font-weight:600;">Action Required</div>
      <h2 style="margin:12px 0 8px; font-size:18px;">Hi {{.user.username}},</h2>
      <p style="margin:0 0 12px;">Your VPS <strong>{{.vps.name}}</strong> will expire on <strong>{{.vps.expire_at}}</strong>.</p>
      <div style="background:#fff7ed; border-radius:10px; padding:12px;">
        <div style="font-size:12px; color:#9a3412;">Recommendation</div>
        <div style="font-size:14px; font-weight:600; padding-top:4px;">Renew early to avoid service interruption.</div>
      </div>
      <p style="margin:16px 0 0; font-size:13px; color:#9a3412;">If you have questions, contact support.</p>
    </div>
    <div style="padding-top:12px; font-size:12px; color:#c2410c;">This is an automated message.</div>
  </div>
</body>
</html>`
  },
  {
    key: "order_approved",
    label: "订单通过 (order_approved)",
    subject: "Order Approved: {{.order.no}}",
    body: `<!DOCTYPE html>
<html>
<body style="margin:0; padding:24px; background:#ecfeff; font-family: Arial, sans-serif; color:#1f2329;">
  <div style="max-width:640px; margin:0 auto;">
    <div style="font-size:12px; color:#0e7490;">Order Update</div>
    <div style="font-size:20px; font-weight:700;">Order Approved</div>
    <div style="height:12px;"></div>
    <div style="background:#ffffff; border-radius:12px; box-shadow:0 8px 20px rgba(14,116,144,0.08); padding:24px;">
      <div style="display:inline-block; padding:6px 10px; background:#cffafe; color:#0e7490; border-radius:999px; font-size:12px; font-weight:600;">Approved</div>
      <h2 style="margin:12px 0 8px; font-size:18px;">Hi {{.user.username}},</h2>
      <p style="margin:0 0 12px;">Your order <strong>{{.order.no}}</strong> has been approved.</p>
      <div style="background:#f0fdfa; border-radius:10px; padding:12px; font-size:14px;">
        {{.message}}
      </div>
      <p style="margin:16px 0 0; font-size:13px; color:#0e7490;">You will receive another email when provisioning is complete.</p>
    </div>
    <div style="padding-top:12px; font-size:12px; color:#0891b2;">This is an automated message.</div>
  </div>
</body>
</html>`
  },
  {
    key: "order_rejected",
    label: "订单驳回 (order_rejected)",
    subject: "Order Rejected: {{.order.no}}",
    body: `<!DOCTYPE html>
<html>
<body style="margin:0; padding:24px; background:#fef2f2; font-family: Arial, sans-serif; color:#1f2329;">
  <div style="max-width:640px; margin:0 auto;">
    <div style="font-size:12px; color:#b91c1c;">Order Update</div>
    <div style="font-size:20px; font-weight:700;">Order Rejected</div>
    <div style="height:12px;"></div>
    <div style="background:#ffffff; border-radius:12px; box-shadow:0 8px 20px rgba(185,28,28,0.08); padding:24px;">
      <div style="display:inline-block; padding:6px 10px; background:#fee2e2; color:#b91c1c; border-radius:999px; font-size:12px; font-weight:600;">Rejected</div>
      <h2 style="margin:12px 0 8px; font-size:18px;">Hi {{.user.username}},</h2>
      <p style="margin:0 0 12px;">Your order <strong>{{.order.no}}</strong> has been rejected.</p>
      <div style="background:#fef2f2; border-radius:10px; padding:12px; font-size:14px;">
        Reason: {{.message}}
      </div>
      <p style="margin:16px 0 0; font-size:13px; color:#b91c1c;">You can reply to this email if you need help.</p>
    </div>
    <div style="padding-top:12px; font-size:12px; color:#ef4444;">This is an automated message.</div>
  </div>
</body>
</html>`
  }
];

const defaultTemplateOptions = DEFAULT_TEMPLATES.map((item) => ({ key: item.key, label: item.label }));

const mockData = reactive({
  user: { id: 1001, username: "demo_user", email: "demo@example.com", qq: "123456" },
  order: { no: "ORD-20240501-0001" },
  vps: { name: "vps-001", expire_at: "2024-12-31" },
  message: "This is a mock message.",
  now: ""
});

const mockDataJson = computed(() => JSON.stringify(mockData, null, 2));

const refreshMockData = () => {
  mockData.now = new Date().toISOString();
};

const isHtmlContent = (value) => /<\/?[a-z][\s\S]*>/i.test(String(value || ""));

const resolvePath = (obj, path) =>
  path.split(".").reduce((acc, key) => (acc && acc[key] !== undefined ? acc[key] : undefined), obj);

const renderWithMock = (input) => {
  if (!input) return "";
  const text = String(input);
  return text.replace(/{{\s*\.([a-zA-Z0-9_.]+)\s*}}/g, (_match, key) => {
    const val = resolvePath(mockData, key);
    return val === undefined || val === null ? "" : String(val);
  });
};

const applyDefaultTemplate = () => {
  const match = DEFAULT_TEMPLATES.find((item) => item.key === defaultTemplateKey.value);
  if (!match) {
    message.warning("请选择默认模板");
    return;
  }
  if (!templateForm.name) templateForm.name = match.key;
  templateForm.subject = match.subject;
  templateForm.body = match.body;
};

const openHelp = () => {
  Modal.info({
    title: "编辑器使用帮助",
    width: 600,
    content: `
      <div style="line-height: 1.8;">
        <h4>编辑模式切换</h4>
        <p>点击工具栏右上角的 <code>&lt;/&gt;</code> 按钮可在可视化模式和 HTML 源码模式之间切换。</p>

        <h4>插入模板变量</h4>
        <p>点击工具栏的 <code>插入变量</code> 按钮，从下拉菜单中选择要插入的变量（如 {{.user.username}}）。</p>

        <h4>模板变量保护</h4>
        <p>插入的变量会被自动保护，整体可删除但不可修改内容。变量会以渐变色背景显示。</p>

        <h4>快捷键</h4>
        <ul>
          <li><code>Ctrl+B</code> - 加粗</li>
          <li><code>Ctrl+I</code> - 斜体</li>
          <li><code>Ctrl+U</code> - 下划线</li>
          <li><code>Ctrl+Z</code> - 撤销</li>
          <li><code>Ctrl+Y</code> - 重做</li>
          <li><code>Ctrl+Shift+S</code> - 切换编辑模式</li>
          <li><code>Esc</code> - 退出全屏</li>
        </ul>

        <h4>右键菜单</h4>
        <p>在编辑器中右键可快速访问撤销、重做、剪切、复制、粘贴和清除格式等功能。</p>
      </div>
    `,
  });
};

const openPreview = () => {
  refreshMockData();
  previewSubject.value = renderWithMock(templateForm.subject) || "(无主题)";
  previewBody.value = renderWithMock(templateForm.body) || "(无内容)";
  previewIsHtml.value = isHtmlContent(previewBody.value);
  previewOpen.value = true;
};

const sendTest = async () => {
  if (!templateTestTo.value) {
    message.error("请输入测试收件人");
    return;
  }
  if (!templateForm.body) {
    message.error("模板内容不能为空");
    return;
  }
  refreshMockData();
  const variables = JSON.parse(JSON.stringify(mockData));
  await testSmtpConfig({
    to: templateTestTo.value,
    subject: templateForm.subject,
    body: templateForm.body,
    variables,
    html: isHtmlContent(templateForm.body)
  });
  message.success("已发送测试邮件");
};

const sendSmtpTest = async () => {
  if (!smtpTestTo.value) {
    message.error("请输入接收人邮箱");
    return;
  }
  refreshMockData();
  const variables = JSON.parse(JSON.stringify(mockData));
  const enabledTemplate = templates.value.find((item) => item.enabled ?? item.Enabled);
  if (enabledTemplate?.name || enabledTemplate?.Name) {
    await testSmtpConfig({
      to: smtpTestTo.value,
      template_name: enabledTemplate.name ?? enabledTemplate.Name,
      variables
    });
  } else {
    await testSmtpConfig({
      to: smtpTestTo.value,
      subject: "SMTP Test",
      body: "如果您收到此邮件，则说明邮箱成功配置！",
      variables,
      html: false
    });
  }
  message.success("测试邮件已发送");
};

const loadSettings = async () => {
  const [settingsRes, smtpRes] = await Promise.all([listSettings(), getSmtpConfig()]);
  const items = settingsRes.data?.items || [];
  const map = new Map(items.map((i) => [i.key ?? i.Key, i.value ?? i.Value ?? i.ValueJSON]));
  const smtp = smtpRes.data || {};
  const smtpHost = smtp.host ?? map.get("smtp_host");
  const smtpPort = smtp.port ?? map.get("smtp_port");
  const smtpUser = smtp.user ?? map.get("smtp_user");
  const smtpPass = smtp.pass ?? map.get("smtp_pass");
  const smtpFrom = smtp.from ?? map.get("smtp_from");
  form.smtp_host = smtpHost || "";
  form.smtp_port = smtpPort || "";
  form.smtp_user = smtpUser || "";
  form.smtp_pass = smtpPass || "";
  form.smtp_from = smtpFrom || "";
  const settingSmtpEnabled = map.get("smtp_enabled") === "true" || map.get("smtp_enabled") === true;
  form.smtp_enabled = smtp.enabled !== undefined && smtp.enabled !== null ? smtp.enabled : settingSmtpEnabled;
  form.email_enabled = map.get("email_enabled") === "true" || map.get("email_enabled") === true;
  form.email_expire_enabled = map.get("email_expire_enabled") === "true" || map.get("email_expire_enabled") === true;
  form.expire_reminder_days = Number(map.get("expire_reminder_days") || 7);
};

const save = async () => {
  await updateSmtpConfig({
    host: form.smtp_host,
    port: form.smtp_port,
    user: form.smtp_user,
    pass: form.smtp_pass,
    from: form.smtp_from,
    enabled: form.smtp_enabled
  });
  await updateSetting({ key: "email_enabled", value: String(form.email_enabled) });
  await updateSetting({ key: "email_expire_enabled", value: String(form.email_expire_enabled) });
  await updateSetting({ key: "expire_reminder_days", value: String(form.expire_reminder_days) });
  message.success("已保存配置");
};

const loadTemplates = async () => {
  const res = await listEmailTemplates();
  templates.value = res.data?.items || [];
};

const openTemplate = (record) => {
  console.log('[DEBUG] openTemplate 被调用', { record, timestamp: new Date().toISOString() });

  if (record) {
    templateForm.id = record.id ?? record.ID ?? null;
    templateForm.name = record.name ?? record.Name ?? "";
    templateForm.subject = record.subject ?? record.Subject ?? "";
    templateForm.body = record.body ?? record.Body ?? record.content ?? record.Content ?? "";
    templateForm.enabled = record.enabled ?? record.Enabled ?? true;
  } else {
    Object.assign(templateForm, { id: null, name: "", subject: "", body: "", enabled: true });
  }
  defaultTemplateKey.value = null;
  templateTestTo.value = "";

  console.log('[DEBUG] 准备打开 Modal', {
    templateOpen: templateOpen.value,
    bodyLength: templateForm.body?.length
  });

  templateOpen.value = true;
};

const saveTemplate = async () => {
  const payload = {
    name: templateForm.name,
    subject: templateForm.subject,
    body: templateForm.body,
    enabled: templateForm.enabled
  };
  if (templateForm.id) {
    await updateEmailTemplate(templateForm.id, payload);
  } else {
    await upsertEmailTemplate(payload);
  }
  message.success("模板已保存");
  templateOpen.value = false;
  loadTemplates();
};

const removeTemplate = (record) => {
  Modal.confirm({
    title: "确认删除该模板?",
    onOk: async () => {
      await deleteEmailTemplate(record.id);
      message.success("已删除");
      loadTemplates();
    }
  });
};

onMounted(() => {
  loadSettings();
  loadTemplates();
});
</script>

<style scoped>
.template-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  gap: 8px;
  flex-wrap: wrap;
}

.editor-hint {
  padding: 8px 12px;
  background: #e6f7ff;
  border: 1px solid #91d5ff;
  border-radius: 4px;
  margin-bottom: 8px;
  font-size: 13px;
  color: #096dd9;
}

.template-vars {
  display: grid;
  gap: 4px;
}

.test-row {
  display: flex;
  gap: 12px;
}

.test-row :deep(.ant-input) {
  flex: 1;
}

.preview-block {
  margin-bottom: 16px;
}

.preview-title {
  font-weight: 600;
  margin-bottom: 6px;
}

.preview-subject {
  padding: 8px 12px;
  background: #f5f6f8;
  border-radius: 6px;
}

.preview-body {
  padding: 12px;
  background: #f5f6f8;
  border-radius: 6px;
}

.preview-text,
.mock-data {
  padding: 12px;
  background: #f5f6f8;
  border-radius: 6px;
  white-space: pre-wrap;
}
</style>
