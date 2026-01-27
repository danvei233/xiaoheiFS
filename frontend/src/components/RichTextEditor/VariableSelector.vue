<template>
  <div class="variable-selector">
    <a-dropdown :trigger="['click']" placement="bottomLeft">
      <a-button type="text" size="small" class="variable-btn">
        <span class="variable-icon">{{ }}</span>
        <span>插入变量</span>
      </a-button>
      <template #overlay>
        <a-menu @click="handleSelect">
          <a-menu-item-group v-for="(group, category) in variableGroups" :key="category" :title="category">
            <a-menu-item v-for="item in group" :key="item.key">
              <span class="variable-label">{{ item.label }}</span>
              <span class="variable-code">{{ item.code }}</span>
            </a-menu-item>
          </a-menu-item-group>
        </a-menu>
      </template>
    </a-dropdown>
  </div>
</template>

<script setup lang="ts">

const emit = defineEmits<{
  insert: [variable: string];
}>();

const variableGroups = {
  '用户信息': [
    { key: 'user.id', label: '用户 ID', code: '{{ .user.id }}' },
    { key: 'user.username', label: '用户名', code: '{{ .user.username }}' },
    { key: 'user.email', label: '邮箱', code: '{{ .user.email }}' },
    { key: 'user.qq', label: 'QQ', code: '{{ .user.qq }}' },
  ],
  '订单信息': [
    { key: 'order.no', label: '订单号', code: '{{ .order.no }}' },
    { key: 'order.amount', label: '订单金额', code: '{{ .order.amount }}' },
  ],
  'VPS 信息': [
    { key: 'vps.name', label: 'VPS 名称', code: '{{ .vps.name }}' },
    { key: 'vps.ip', label: 'IP 地址', code: '{{ .vps.ip }}' },
    { key: 'vps.expire_at', label: '到期时间', code: '{{ .vps.expire_at }}' },
  ],
  '其他': [
    { key: 'message', label: '消息', code: '{{ .message }}' },
    { key: 'now', label: '当前时间', code: '{{ .now }}' },
  ],
};

const handleSelect = ({ key }: { key: string }) => {
  // 查找对应的变量代码
  for (const group of Object.values(variableGroups)) {
    const item = group.find(g => g.key === key);
    if (item) {
      emit('insert', item.code);
      return;
    }
  }
};
</script>

<style scoped>
.variable-selector {
  display: inline-block;
}

.variable-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text2, #666);
  transition: all 0.2s;
  padding: 4px 8px;
}

.variable-btn:hover {
  color: var(--primary, #1890ff);
  background: var(--bg-soft, #f5f6f8);
}

.variable-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  font-family: 'Fira Code', 'Consolas', 'Monaco', monospace;
}

.variable-label {
  flex: 1;
}

.variable-code {
  font-family: 'Fira Code', 'Consolas', 'Monaco', monospace;
  color: var(--primary, #1890ff);
  background: var(--primary-light, #e6f7ff);
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 12px;
}
</style>