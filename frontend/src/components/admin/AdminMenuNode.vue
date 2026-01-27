<template>
  <a-menu-item v-if="!hasChildren" :key="node.key" :class="itemClass">
    <component v-if="node.icon" :is="node.icon" />
    <span>{{ node.label }}</span>
  </a-menu-item>

  <a-sub-menu v-else :key="node.key">
    <template #title>
      <component v-if="node.icon" :is="node.icon" />
      <span>{{ node.label }}</span>
    </template>
    <AdminMenuNode v-for="child in node.children" :key="child.key" :node="child" :variant="variant" />
  </a-sub-menu>
</template>

<script setup lang="ts">
import { computed } from "vue";

defineOptions({ name: "AdminMenuNode" });

type MenuNode = {
  key: string;
  label: string;
  icon?: unknown;
  children?: MenuNode[];
};

const props = defineProps<{
  node: MenuNode;
  variant?: "default" | "drawer";
}>();

const hasChildren = computed(() => Array.isArray(props.node.children) && props.node.children.length > 0);
const itemClass = computed(() => (props.variant === "drawer" ? "drawer-menu-item" : undefined));
</script>
