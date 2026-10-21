<template>
  <Teleport to="body">
    <div
      v-if="visible"
      ref="menuRef"
      class="context-menu"
      :style="{ left: x + 'px', top: y + 'px' }"
      @click.stop
    >
      <div
        v-for="item in menuItems"
        :key="item.key"
        class="context-menu-item"
        :class="{ disabled: item.disabled, separator: item.separator }"
        @click="handleItemClick(item)"
      >
        <span v-if="item.icon" v-html="item.icon" class="menu-icon"></span>
        <span class="menu-text">{{ item.label }}</span>
        <span v-if="item.shortcut" class="menu-shortcut">{{ item.shortcut }}</span>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue';

interface MenuItem {
  key: string;
  label: string;
  icon?: string;
  shortcut?: string;
  disabled?: boolean;
  separator?: boolean;
  action?: () => void;
}

const props = defineProps<{
  visible: boolean;
  x: number;
  y: number;
  menuItems: MenuItem[];
}>();

const emit = defineEmits<{
  close: [];
  itemClick: [item: MenuItem];
}>();

const menuRef = ref<HTMLElement | null>(null);

const handleItemClick = (item: MenuItem) => {
  if (item.disabled || item.separator) return;
  item.action?.();
  emit('itemClick', item);
  emit('close');
};

// 点击外部关闭菜单
const handleClickOutside = (e: MouseEvent) => {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    emit('close');
  }
};

// 监听位置变化，确保菜单不超出视口
watch([() => props.x, () => props.y], ([x, y]) => {
  if (!menuRef.value) return;

  const rect = menuRef.value.getBoundingClientRect();
  const viewportWidth = window.innerWidth;
  const viewportHeight = window.innerHeight;

  if (x + rect.width > viewportWidth) {
    menuRef.value.style.left = (x - rect.width) + 'px';
  }

  if (y + rect.height > viewportHeight) {
    menuRef.value.style.top = (y - rect.height) + 'px';
  }
});

onMounted(() => {
  document.addEventListener('click', handleClickOutside);
  document.addEventListener('contextmenu', handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
  document.removeEventListener('contextmenu', handleClickOutside);
});
</script>

<style scoped>
.context-menu {
  position: fixed;
  z-index: 1080;
  min-width: 180px;
  background: white;
  border: 1px solid var(--border, #e6e8ec);
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 4px 0;
  user-select: none;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  font-size: 14px;
  color: var(--text1, #1f2329);
  transition: background 0.15s;
}

.context-menu-item:hover:not(.disabled):not(.separator) {
  background: var(--primary-light, #e6f7ff);
}

.context-menu-item.disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.context-menu-item.separator {
  padding: 0;
  margin: 4px 0;
  height: 1px;
  background: var(--border, #e6e8ec);
  cursor: default;
}

.menu-icon {
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text2, #666);
}

.menu-text {
  flex: 1;
}

.menu-shortcut {
  font-size: 12px;
  color: var(--text2, #999);
}

.context-menu-item:hover .menu-icon {
  color: var(--primary, #1890ff);
}
</style>