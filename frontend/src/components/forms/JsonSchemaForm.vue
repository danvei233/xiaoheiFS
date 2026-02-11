<template>
  <a-form layout="vertical" class="schema-form">
    <a-alert
      v-if="!isObjectRoot"
      type="warning"
      show-icon
      message="该插件未提供可渲染的 JSON Schema（仅支持 object/properties）"
      style="margin-bottom: 12px"
    />
    <template v-else>
      <JsonSchemaField
        v-for="key in orderedKeys"
        :key="key"
        :schema="schema.properties[key]"
        :path="[key]"
        :model="localModel"
        :required="requiredKeys.includes(key)"
        :ui="uiSchema?.[key]"
      />
    </template>
  </a-form>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import JsonSchemaField from "./JsonSchemaField.vue";

const props = defineProps<{
  schema: any;
  uiSchema?: any;
  modelValue: Record<string, any>;
}>();
const emit = defineEmits<{
  (e: "update:modelValue", v: Record<string, any>): void;
}>();

const localModel = reactive<Record<string, any>>({});
let syncingFromParent = false;
let lastSnapshot = "{}";

const safeParse = (obj: any) => {
  try {
    return JSON.parse(JSON.stringify(obj || {}));
  } catch {
    return {};
  }
};

const snapshotOf = (obj: any) => {
  try {
    return JSON.stringify(obj || {});
  } catch {
    return "{}";
  }
};

watch(
  () => props.modelValue,
  (v) => {
    const next = safeParse(v);
    const nextSnapshot = snapshotOf(next);
    if (nextSnapshot === lastSnapshot) return;
    syncingFromParent = true;
    Object.keys(localModel).forEach((k) => delete localModel[k]);
    Object.assign(localModel, next);
    lastSnapshot = nextSnapshot;
    syncingFromParent = false;
  },
  { immediate: true, deep: true }
);

watch(
  localModel,
  () => {
    if (syncingFromParent) return;
    const next = safeParse(localModel);
    const nextSnapshot = snapshotOf(next);
    if (nextSnapshot === lastSnapshot) return;
    lastSnapshot = nextSnapshot;
    emit("update:modelValue", next);
  },
  { deep: true }
);

const isObjectRoot = computed(() => String(props.schema?.type || "") === "object" && !!props.schema?.properties);
const requiredKeys = computed(() => (Array.isArray(props.schema?.required) ? props.schema.required.map(String) : []));
const orderedKeys = computed(() => {
  const keys = Object.keys(props.schema?.properties || {});
  const uiOrder: any[] = props.uiSchema?.["ui:order"] || props.schema?.["ui:order"] || [];
  if (!Array.isArray(uiOrder) || uiOrder.length === 0) return keys;
  const out: string[] = [];
  for (const it of uiOrder) {
    if (it === "*") {
      for (const k of keys) if (!out.includes(k)) out.push(k);
      continue;
    }
    const k = String(it);
    if (keys.includes(k) && !out.includes(k)) out.push(k);
  }
  for (const k of keys) if (!out.includes(k)) out.push(k);
  return out;
});
</script>

<style scoped>
.schema-form :deep(.ant-form-item) {
  margin-bottom: 12px;
}
</style>