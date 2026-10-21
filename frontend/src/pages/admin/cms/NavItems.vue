<template>
  <div class="cms-nav-items-page">
    <div class="page-header">
      <div class="title-wrap">
        <h1 class="page-title">主页顶栏</h1>
        <div class="page-subtitle">管理 PublicLayout 顶部导航（设置项：<code>site_nav_items</code>）</div>
      </div>

      <a-space>
        <a-button @click="resetToDefault" :disabled="loading || saving">
          恢复默认
        </a-button>
        <a-button @click="fetchData" :loading="loading">
          刷新
        </a-button>
        <a-button type="primary" @click="save" :loading="saving">
          保存
        </a-button>
      </a-space>
    </div>

    <a-row :gutter="16">
      <a-col :xs="24" :lg="9">
        <a-card :bordered="false" class="card">
          <template #title>
            <div class="card-title">
              <span>预览</span>
              <a-tag color="blue">{{ previewLang }}</a-tag>
            </div>
          </template>

          <div class="preview-shell">
            <div class="preview-brand">
              <div class="preview-logo">+</div>
              <div class="preview-name">站点</div>
            </div>
            <div class="preview-links">
              <div class="preview-link is-fixed">首页</div>
              <div
                v-for="item in previewItems"
                :key="item.id"
                class="preview-link"
                :class="{ disabled: item.enabled === false }"
                :title="item.url"
              >
                <span class="label">{{ item.label || "未命名" }}</span>
                <span class="hint">{{ item.target === '_blank' ? "新窗口" : "本页" }}</span>
              </div>
              <div v-if="previewItems.length === 0" class="preview-empty">
                暂无导航项（不包含固定的“首页”）
              </div>
            </div>
          </div>

          <a-divider style="margin: 16px 0" />

          <a-form layout="vertical">
            <a-form-item label="预览语言">
              <a-select v-model:value="previewLang" :options="langOptions" style="width: 100%" />
            </a-form-item>
            <a-form-item label="说明">
              <div class="help">
                - “首页”在前台固定显示；这里管理其余链接。<br />
                - 语言会按 <code>lang</code> 精确匹配；为空则对所有语言生效。
              </div>
            </a-form-item>
          </a-form>
        </a-card>

        <a-card :bordered="false" class="card json-card" title="高级：JSON">
          <a-textarea
            v-model:value="rawJson"
            :rows="10"
            placeholder='[{"label":"产品","url":"/products","target":"_self","lang":"zh-CN"}]'
          />
          <div class="json-actions">
            <a-button @click="applyRawJson" :disabled="saving || loading">应用 JSON 到表格</a-button>
            <a-button @click="syncRawJson" :disabled="saving || loading">从表格生成 JSON</a-button>
          </div>
        </a-card>
      </a-col>

      <a-col :xs="24" :lg="15">
        <a-card :bordered="false" class="card">
          <template #title>
            <div class="card-title">
              <span>导航项</span>
              <a-space size="small">
                <a-button type="dashed" @click="addItem">
                  新增
                </a-button>
                <a-button @click="syncRawJson">
                  生成 JSON
                </a-button>
              </a-space>
            </div>
          </template>

          <a-table
            :data-source="items"
            :columns="columns"
            :pagination="false"
            :row-key="(r) => r.id"
            size="middle"
            class="nav-table"
          >
            <template #bodyCell="{ column, record, index }">
              <template v-if="column.key === 'sort'">
                <a-space>
                  <a-button size="small" @click="moveUp(index)" :disabled="index === 0">上移</a-button>
                  <a-button size="small" @click="moveDown(index)" :disabled="index === items.length - 1">下移</a-button>
                </a-space>
              </template>

              <template v-else-if="column.key === 'label'">
                <a-input v-model:value="record.label" placeholder="例如：产品" />
              </template>

              <template v-else-if="column.key === 'url'">
                <a-input v-model:value="record.url" placeholder="/products 或 https://..." />
              </template>

              <template v-else-if="column.key === 'target'">
                <a-select
                  v-model:value="record.target"
                  style="width: 110px"
                  :options="[
                    { label: '本页', value: '_self' },
                    { label: '新窗口', value: '_blank' }
                  ]"
                />
              </template>

              <template v-else-if="column.key === 'lang'">
                <a-select
                  v-model:value="record.lang"
                  allow-clear
                  style="width: 120px"
                  :options="langOptionsWithAll"
                  placeholder="全部语言"
                />
              </template>

              <template v-else-if="column.key === 'enabled'">
                <a-switch v-model:checked="record.enabled" />
              </template>

              <template v-else-if="column.key === 'actions'">
                <a-space>
                  <a-button size="small" @click="duplicate(record)">复制</a-button>
                  <a-button size="small" danger @click="remove(record.id)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>

          <div class="table-foot">
            <div class="foot-left">
              <span class="muted">保存后，前台将从 <code>/api/v1/site/settings</code> 读取并渲染。</span>
            </div>
            <div class="foot-right">
              <a-space>
                <a-button @click="addItem">新增</a-button>
                <a-button type="primary" @click="save" :loading="saving">保存</a-button>
              </a-space>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { message, Modal } from "ant-design-vue";
import { v4 as uuidv4 } from "uuid";
import { listSettings, updateSetting } from "@/services/admin";

type NavItemTarget = "_self" | "_blank";

type NavItem = {
  id: string;
  label: string;
  url: string;
  target: NavItemTarget;
  lang?: string;
  enabled: boolean;
};

const SETTING_KEY = "site_nav_items";

const loading = ref(false);
const saving = ref(false);

const items = ref<NavItem[]>([]);
const rawJson = ref("");

const siteLanguages = ["zh-CN", "en-US"];
const previewLang = ref(siteLanguages[0]);

const langOptions = siteLanguages.map((l) => ({ label: l, value: l }));
const langOptionsWithAll = [{ label: "全部", value: "" }, ...langOptions].map((o) => ({
  label: o.label,
  value: o.value || undefined
}));

const defaultItems: NavItem[] = [
  { id: uuidv4(), label: "产品", url: "/products", target: "_self", lang: "zh-CN", enabled: true },
  { id: uuidv4(), label: "活动", url: "/activities", target: "_self", lang: "zh-CN", enabled: true },
  { id: uuidv4(), label: "文档", url: "/docs", target: "_self", lang: "zh-CN", enabled: true },
  { id: uuidv4(), label: "帮助", url: "/help", target: "_self", lang: "zh-CN", enabled: true },
  { id: uuidv4(), label: "Products", url: "/products", target: "_self", lang: "en-US", enabled: true },
  { id: uuidv4(), label: "Activities", url: "/activities", target: "_self", lang: "en-US", enabled: true },
  { id: uuidv4(), label: "Docs", url: "/docs", target: "_self", lang: "en-US", enabled: true },
  { id: uuidv4(), label: "Help", url: "/help", target: "_self", lang: "en-US", enabled: true }
];

const sanitize = (input: any): NavItem[] => {
  const arr = Array.isArray(input) ? input : [];
  return arr
    .map((x: any) => {
      const id = String(x?.id || uuidv4());
      const label = String(x?.label || "").trim();
      const url = String(x?.url || "").trim();
      const target = (String(x?.target || "_self") as NavItemTarget) === "_blank" ? "_blank" : "_self";
      const lang = x?.lang ? String(x.lang).trim() : undefined;
      const enabled = x?.enabled === false ? false : true;
      return { id, label, url, target, lang: lang || undefined, enabled };
    })
    .filter((x: NavItem) => x.label || x.url);
};

const previewItems = computed(() => {
  const lang = previewLang.value;
  return items.value.filter((x) => x.enabled !== false && (!x.lang || x.lang === lang));
});

const columns = [
  { title: "排序", key: "sort", width: 160 },
  { title: "名称", key: "label" },
  { title: "链接", key: "url" },
  { title: "打开方式", key: "target", width: 140 },
  { title: "语言", key: "lang", width: 140 },
  { title: "启用", key: "enabled", width: 90 },
  { title: "操作", key: "actions", width: 160 }
];

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listSettings();
    const list = res.data?.items || [];
    const found = list.find((x: any) => x?.key === SETTING_KEY);
    if (!found?.value) {
      items.value = sanitize(defaultItems);
      syncRawJson();
      return;
    }
    try {
      const parsed = JSON.parse(String(found.value));
      items.value = sanitize(parsed);
      syncRawJson();
    } catch {
      items.value = [];
      rawJson.value = String(found.value);
      message.error("site_nav_items 不是合法 JSON：已载入原始内容，请修正后点“应用 JSON 到表格”");
    }
  } catch (e: any) {
    message.error(e?.response?.data?.error || "加载失败");
  } finally {
    loading.value = false;
  }
};

const syncRawJson = () => {
  rawJson.value = JSON.stringify(
    items.value.map((x) => ({
      id: x.id,
      label: x.label?.trim(),
      url: x.url?.trim(),
      target: x.target,
      lang: x.lang || undefined,
      enabled: x.enabled !== false
    })),
    null,
    2
  );
};

const applyRawJson = () => {
  try {
    const parsed = JSON.parse(rawJson.value || "[]");
    items.value = sanitize(parsed);
    message.success("已应用 JSON");
  } catch {
    message.error("JSON 解析失败");
  }
};

const addItem = () => {
  items.value.push({
    id: uuidv4(),
    label: "",
    url: "",
    target: "_self",
    lang: previewLang.value,
    enabled: true
  });
};

const remove = (id: string) => {
  items.value = items.value.filter((x) => x.id !== id);
  syncRawJson();
};

const duplicate = (src: NavItem) => {
  items.value.push({ ...src, id: uuidv4() });
  syncRawJson();
};

const moveUp = (index: number) => {
  if (index <= 0) return;
  const next = [...items.value];
  const [item] = next.splice(index, 1);
  next.splice(index - 1, 0, item);
  items.value = next;
  syncRawJson();
};

const moveDown = (index: number) => {
  if (index >= items.value.length - 1) return;
  const next = [...items.value];
  const [item] = next.splice(index, 1);
  next.splice(index + 1, 0, item);
  items.value = next;
  syncRawJson();
};

const resetToDefault = () => {
  Modal.confirm({
    title: "恢复默认导航？",
    content: "将用默认模板覆盖当前表格内容（未保存的修改会丢失）。",
    okText: "恢复",
    cancelText: "取消",
    onOk: () => {
      items.value = sanitize(defaultItems);
      syncRawJson();
      message.success("已恢复默认（别忘了点保存）");
    }
  });
};

const save = async () => {
  saving.value = true;
  try {
    const toSave = sanitize(items.value).map((x) => ({
      id: x.id,
      label: x.label.trim(),
      url: x.url.trim(),
      target: x.target,
      lang: x.lang || undefined,
      enabled: x.enabled !== false
    }));
    await updateSetting({ key: SETTING_KEY, value: JSON.stringify(toSave) });
    message.success("保存成功");
    syncRawJson();
  } catch (e: any) {
    message.error(e?.response?.data?.error || "保存失败");
  } finally {
    saving.value = false;
  }
};

onMounted(() => {
  fetchData();
});
</script>

<style scoped>
.cms-nav-items-page {
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

.title-wrap {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.page-subtitle {
  color: rgba(148, 163, 184, 0.9);
  font-size: 12px;
}

.page-subtitle code {
  padding: 2px 6px;
  border: 1px solid rgba(30, 41, 59, 0.9);
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.6);
}

.card {
  border-radius: 16px;
  border: 1px solid rgba(30, 41, 59, 0.8);
  background: rgba(17, 24, 39, 0.6);
  backdrop-filter: blur(10px);
}

.card-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.preview-shell {
  border: 1px solid rgba(30, 41, 59, 0.85);
  border-radius: 14px;
  overflow: hidden;
  background: radial-gradient(1200px 250px at 10% -40%, rgba(14, 165, 233, 0.35), transparent 60%),
    radial-gradient(900px 260px at 85% -60%, rgba(99, 102, 241, 0.22), transparent 60%),
    rgba(15, 23, 42, 0.65);
}

.preview-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 1px solid rgba(30, 41, 59, 0.85);
}

.preview-logo {
  width: 30px;
  height: 30px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.08);
  border: 1px solid rgba(14, 165, 233, 0.35);
  font-weight: 800;
}

.preview-name {
  font-weight: 700;
  color: rgba(241, 245, 249, 0.95);
  letter-spacing: 0.2px;
}

.preview-links {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  padding: 12px 14px 14px;
}

.preview-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 999px;
  border: 1px solid rgba(30, 41, 59, 0.85);
  background: rgba(2, 6, 23, 0.25);
  color: rgba(226, 232, 240, 0.95);
  font-size: 12px;
}

.preview-link.is-fixed {
  border-color: rgba(14, 165, 233, 0.45);
  background: rgba(14, 165, 233, 0.08);
}

.preview-link .hint {
  color: rgba(148, 163, 184, 0.95);
  font-size: 11px;
}

.preview-link.disabled {
  opacity: 0.5;
  filter: grayscale(0.3);
}

.preview-empty {
  color: rgba(148, 163, 184, 0.9);
  font-size: 12px;
  padding: 6px 2px 0;
}

.help {
  color: rgba(148, 163, 184, 0.92);
  font-size: 12px;
  line-height: 1.6;
}

.json-actions {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  margin-top: 10px;
}

.nav-table :deep(.ant-table) {
  background: transparent;
}

.table-foot {
  margin-top: 14px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.muted {
  color: rgba(148, 163, 184, 0.92);
  font-size: 12px;
}
</style>
