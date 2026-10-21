<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">Webhook</div>
        <div class="subtle">机器人通知与测试（支持多条配置）</div>
      </div>
      <div class="page-header-actions">
        <a-space>
          <a-button @click="addWebhook">
            <template #icon>
              <PlusOutlined />
            </template>
            新增 Webhook
          </a-button>
          <a-button @click="testWebhook">发送测试</a-button>
          <a-button type="primary" :loading="saving" @click="save">保存</a-button>
        </a-space>
      </div>
    </div>

    <a-alert
      class="hint"
      type="info"
      show-icon
      message="事件说明"
      description="events 为空代表全事件。你可以留空或填写特定事件。"
    />

    <a-card class="card">
      <a-table :columns="columns" :data-source="webhooks" :pagination="false" row-key="_key" size="middle">
        <template #bodyCell="{ column, record, index }">
          <template v-if="column.key === 'name'">
            <a-input v-model:value="record.name" placeholder="Webhook 名称" />
          </template>
          <template v-else-if="column.key === 'url'">
            <a-input v-model:value="record.url" placeholder="https://..." />
          </template>
          <template v-else-if="column.key === 'secret'">
            <a-input v-model:value="record.secret" placeholder="签名密钥（可选）" />
          </template>
          <template v-else-if="column.key === 'events'">
            <a-select
              v-model:value="record.events"
              mode="tags"
              placeholder="空表示全事件"
              style="min-width: 180px"
              :options="eventOptions"
            />
          </template>
          <template v-else-if="column.key === 'enabled'">
            <a-switch v-model:checked="record.enabled" />
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="text" size="small" danger @click="removeWebhook(index)">移除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
      <div v-if="!webhooks.length" class="empty">
        <a-empty description="暂无配置" />
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { getRobotConfig, updateRobotConfig, testRobotWebhook } from "@/services/admin";
import { message } from "ant-design-vue";
import { PlusOutlined } from "@ant-design/icons-vue";

const saving = ref(false);
const webhooks = ref([]);

const columns = [
  { title: "名称", key: "name", width: 160 },
  { title: "Webhook URL", key: "url" },
  { title: "签名密钥", key: "secret", width: 200 },
  { title: "事件", key: "events", width: 220 },
  { title: "启用", key: "enabled", width: 90 },
  { title: "操作", key: "action", width: 90 }
];

const eventOptions = [
  { label: "订单：待支付", value: "order.pending_payment" },
  { label: "订单：待审核", value: "order.pending_review" },
  { label: "订单：已通过", value: "order.approved" },
  { label: "订单：已驳回", value: "order.rejected" },
  { label: "订单：已取消", value: "order.canceled" },
  { label: "订单：开通中", value: "order.provisioning" },
  { label: "订单：已完成", value: "order.completed" },
  { label: "订单项：开通成功", value: "order.item.active" },
  { label: "订单项：开通失败", value: "order.item.failed" },
  { label: "支付：创建", value: "payment.created" },
  { label: "支付：已确认", value: "payment.confirmed" },
  { label: "支付：已通过", value: "payment.approved" },
  { label: "测试", value: "webhook.test" }
];

const createKey = () => `${Date.now()}-${Math.random().toString(16).slice(2)}`;

const normalizeWebhooks = (items) =>
  (items || []).map((item) => ({
    _key: createKey(),
    name: item.name || "Webhook",
    url: item.url || "",
    secret: item.secret || "",
    enabled: item.enabled ?? true,
    events: Array.isArray(item.events) ? item.events : []
  }));

const load = async () => {
  const res = await getRobotConfig();
  const data = res.data || {};
  const list = normalizeWebhooks(data.webhooks || []);
  webhooks.value = list;
};

const addWebhook = () => {
  webhooks.value.push({
    _key: createKey(),
    name: `Webhook ${webhooks.value.length + 1}`,
    url: "",
    secret: "",
    enabled: true,
    events: []
  });
};

const removeWebhook = (index) => {
  webhooks.value.splice(index, 1);
};

const save = async () => {
  saving.value = true;
  try {
    const payload = webhooks.value.map((item) => ({
      name: String(item.name || "").trim() || "Webhook",
      url: String(item.url || "").trim(),
      secret: item.secret || "",
      enabled: item.enabled ?? false,
      events: Array.isArray(item.events) ? item.events.filter(Boolean) : []
    }));
    await updateRobotConfig({ webhooks: payload });
    message.success("已保存");
  } finally {
    saving.value = false;
  }
};

const testWebhook = async () => {
  await testRobotWebhook({
    event: "webhook.test",
    data: { text: "测试 Webhook", sender: "console", timestamp: Math.floor(Date.now() / 1000) }
  });
  message.success("已发送测试请求");
};

onMounted(load);
</script>

<style scoped>
.hint {
  margin-bottom: 16px;
  border-radius: 10px;
}

.empty {
  padding: 16px 0;
}
</style>
