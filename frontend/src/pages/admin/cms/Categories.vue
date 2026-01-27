<template>
  <div class="cms-categories-page">
    <div class="page-header">
      <h1 class="page-title">CMS分类管理</h1>
      <a-button type="primary" @click="openCreateModal">
        <template #icon><PlusOutlined /></template>
        新建分类
      </a-button>
    </div>

    <a-card :bordered="false">
      <a-table
        :columns="columns"
        :data-source="categories"
        :loading="loading"
        :pagination="false"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'visible'">
            <a-switch
              :checked="record.visible"
              @change="(checked: boolean) => handleToggle(record, checked)"
            />
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button type="link" size="small" @click="openEditModal(record)">
                编辑
              </a-button>
              <a-popconfirm
                title="确定要删除此分类吗？"
                @confirm="handleDelete(record)"
              >
                <a-button type="link" danger size="small">
                  删除
                </a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- Create/Edit Modal -->
    <a-modal
      v-model:open="modalVisible"
      :title="isEditing ? '编辑分类' : '新建分类'"
      @ok="handleSubmit"
      :confirm-loading="submitting"
    >
      <a-form :model="form" layout="vertical">
        <a-form-item label="分类Key">
          <a-input
            v-model:value="form.key"
            placeholder="如: docs, announcements"
            :disabled="isEditing"
          />
        </a-form-item>
        <a-form-item label="分类名称">
          <a-input v-model:value="form.name" placeholder="分类显示名称" />
        </a-form-item>
        <a-form-item label="语言">
          <a-select v-model:value="form.lang">
            <a-select-option value="zh-CN">简体中文</a-select-option>
            <a-select-option value="en-US">English</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="排序">
          <a-input-number v-model:value="form.sort_order" :min="0" style="width: 100%" />
        </a-form-item>
        <a-form-item label="可见">
          <a-switch v-model:checked="form.visible" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { message } from "ant-design-vue";
import { PlusOutlined } from "@ant-design/icons-vue";
import { listCmsCategories, createCmsCategory, updateCmsCategory, deleteCmsCategory } from "@/services/admin";

const loading = ref(false);
const submitting = ref(false);
const modalVisible = ref(false);
const isEditing = ref(false);
const categories = ref<any[]>([]);

const form = reactive({
  id: undefined,
  key: "",
  name: "",
  lang: "zh-CN",
  sort_order: 0,
  visible: true
});

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "Key", dataIndex: "key", key: "key" },
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "语言", dataIndex: "lang", key: "lang", width: 100 },
  { title: "排序", dataIndex: "sort_order", key: "sort_order", width: 80 },
  { title: "可见", dataIndex: "visible", key: "visible", width: 80 },
  { title: "操作", key: "actions", width: 150 }
];

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listCmsCategories();
    categories.value = res.data?.items || [];
  } finally {
    loading.value = false;
  }
};

const handleToggle = async (record: any, checked: boolean) => {
  try {
    await updateCmsCategory(record.id, { visible: checked });
    record.visible = checked;
    message.success("操作成功");
  } catch (error: any) {
    message.error(error.response?.data?.error || "操作失败");
  }
};

const openCreateModal = () => {
  isEditing.value = false;
  Object.assign(form, {
    id: undefined,
    key: "",
    name: "",
    lang: "zh-CN",
    sort_order: 0,
    visible: true
  });
  modalVisible.value = true;
};

const openEditModal = (record: any) => {
  isEditing.value = true;
  Object.assign(form, record);
  modalVisible.value = true;
};

const handleSubmit = async () => {
  if (!form.key || !form.name) {
    message.error("请填写完整信息");
    return;
  }

  submitting.value = true;
  try {
    const payload = {
      key: form.key,
      name: form.name,
      lang: form.lang,
      sort_order: form.sort_order,
      visible: form.visible
    };

    if (isEditing.value) {
      await updateCmsCategory(form.id, payload);
    } else {
      await createCmsCategory(payload);
    }

    message.success(isEditing.value ? "更新成功" : "创建成功");
    modalVisible.value = false;
    fetchData();
  } catch (error: any) {
    message.error(error.response?.data?.error || "操作失败");
  } finally {
    submitting.value = false;
  }
};

const handleDelete = async (record: any) => {
  try {
    await deleteCmsCategory(record.id);
    message.success("删除成功");
    fetchData();
  } catch (error: any) {
    message.error(error.response?.data?.error || "删除失败");
  }
};

onMounted(() => {
  fetchData();
});
</script>

<style scoped>
.cms-categories-page {
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
</style>
