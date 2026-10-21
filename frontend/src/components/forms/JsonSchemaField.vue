<template>
  <template v-if="isObject">
    <a-card size="small" class="object-card">
      <template #title>
        <div class="object-title">
          <span>{{ title }}</span>
          <span v-if="description" class="object-desc">{{ description }}</span>
        </div>
      </template>
      <div class="object-grid">
        <JsonSchemaField
          v-for="key in orderedKeys"
          :key="key"
          :schema="schema.properties[key]"
          :path="[...path, key]"
          :model="model"
          :required="requiredKeys.includes(key)"
          :ui="ui?.[key]"
        />
      </div>
    </a-card>
  </template>

  <template v-else>
    <a-form-item :label="title" :required="required" :help="description" class="field-item">
      <a-select
        v-if="hasEnum"
        :value="value"
        @update:value="(v:any)=>setValue(v)"
        allow-clear
        :placeholder="placeholder"
      >
        <a-select-option v-for="(opt, idx) in schema.enum" :key="String(opt)" :value="opt">
          {{ enumLabel(idx, opt) }}
        </a-select-option>
      </a-select>

      <a-switch
        v-else-if="isBoolean"
        :checked="Boolean(value)"
        @update:checked="(v:boolean)=>setValue(v)"
      />

      <a-input-number
        v-else-if="isNumber"
        :value="value"
        @update:value="(v:any)=>setValue(v)"
        style="width: 100%"
        :placeholder="placeholder"
      />

      <a-input
        v-else
        :value="value"
        @update:value="(v:string)=>setValue(v)"
        :type="isSecret ? 'password' : 'text'"
        :placeholder="placeholder"
        autocomplete="off"
      />
    </a-form-item>
  </template>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  schema: any;
  path: string[];
  model: Record<string, any>;
  required?: boolean;
  ui?: any;
}>();

const title = computed(() => String(props.schema?.title || props.path[props.path.length - 1] || ""));
const description = computed(() => String(props.schema?.description || ""));

const requiredKeys = computed(() => (Array.isArray(props.schema?.required) ? props.schema.required.map(String) : []));
const orderedKeys = computed(() => {
  const keys = Object.keys(props.schema?.properties || {});
  const uiOrder: any[] = props.ui?.["ui:order"] || props.schema?.["ui:order"] || [];
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

const isObject = computed(() => String(props.schema?.type || "") === "object" && !!props.schema?.properties);
const hasEnum = computed(() => Array.isArray(props.schema?.enum) && props.schema.enum.length > 0);
const isBoolean = computed(() => String(props.schema?.type || "") === "boolean");
const isNumber = computed(() => ["number", "integer"].includes(String(props.schema?.type || "")));
const isSecret = computed(() => {
  const format = String(props.schema?.format || "").trim().toLowerCase();
  if (format === "password") return true;
  if (props.schema?.["x-secret"] === true) return true;
  return false;
});

const placeholder = computed(() => {
  if (isSecret.value) return "留空表示不修改";
  return String(props.schema?.placeholder || props.schema?.description || "");
});

const getPathValue = (root: any, path: string[]) => {
  let cur = root;
  for (const key of path) {
    if (cur == null || typeof cur !== "object") return undefined;
    cur = cur[key];
  }
  return cur;
};

const setPathValue = (root: any, path: string[], v: any) => {
  if (!root || typeof root !== "object") return;
  let cur = root;
  for (let i = 0; i < path.length - 1; i++) {
    const key = path[i];
    if (cur[key] == null || typeof cur[key] !== "object") cur[key] = {};
    cur = cur[key];
  }
  cur[path[path.length - 1]] = v;
};

const value = computed(() => getPathValue(props.model, props.path));

const setValue = (v: any) => {
  // Keep empty-string for secrets so the backend can preserve old values ("留空表示不修改").
  if (isSecret.value && (v === undefined || v === null)) v = "";
  setPathValue(props.model, props.path, v);
};

const enumLabel = (idx: number, opt: any) => {
  const names = Array.isArray(props.schema?.enumNames) ? props.schema.enumNames : [];
  const title = names[idx];
  if (title != null && String(title).trim() !== "") return String(title);
  return String(opt);
};
</script>

<style scoped>
.object-card {
  border-radius: 10px;
  border: 1px solid rgba(0, 0, 0, 0.06);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.94), rgba(255, 255, 255, 0.9));
}
.object-title {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.object-desc {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}
.object-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 8px;
}
.field-item :deep(.ant-form-item-label > label) {
  font-weight: 600;
}
</style>
