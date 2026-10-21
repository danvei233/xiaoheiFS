<template>
  <div class="cms-posts-page">
    <div class="page-header">
      <h1 class="page-title">CMS文章管理</h1>
      <a-button type="primary" @click="openCreateModal">
        <template #icon><PlusOutlined /></template>
        新建文章
      </a-button>
    </div>

    <a-card :bordered="false">
      <ProTable
        :columns="columns"
        :data-source="posts"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
      >
        <template #toolbar>
          <a-select
            v-model:value="filters.category_id"
            placeholder="分类筛选"
            style="width: 150px"
            allow-clear
            @change="handleFilterChange"
          >
            <a-select-option value="">全部分类</a-select-option>
            <a-select-option v-for="cat in categories" :key="cat.id" :value="cat.id">
              {{ cat.name }}
            </a-select-option>
          </a-select>
          <a-select
            v-model:value="filters.status"
            placeholder="状态筛选"
            style="width: 120px"
            allow-clear
            @change="handleFilterChange"
          >
            <a-select-option value="">全部</a-select-option>
            <a-select-option value="draft">草稿</a-select-option>
            <a-select-option value="published">已发布</a-select-option>
          </a-select>
        </template>

        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'title'">
            <div class="post-title-cell">
              <a-tag v-if="record.pinned" color="red" size="small">置顶</a-tag>
              {{ record.title }}
            </div>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="record.status === 'published' ? 'success' : 'default'">
              {{ record.status === 'published' ? '已发布' : '草稿' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'published_at'">
            {{ formatDate(record.published_at) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button type="link" size="small" @click="openEditModal(record)">
                编辑
              </a-button>
              <a-popconfirm
                title="确定要删除此文章吗？"
                @confirm="handleDelete(record)"
              >
                <a-button type="link" danger size="small">
                  删除
                </a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </ProTable>
    </a-card>

    <!-- Create/Edit Modal -->
    <a-modal
      v-model:open="modalVisible"
      :title="isEditing ? '编辑文章' : '新建文章'"
      @ok="handleSubmit"
      :confirm-loading="submitting"
      width="800px"
    >
      <a-form :model="form" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="标题">
              <a-input v-model:value="form.title" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="分类">
              <a-select v-model:value="form.category_id">
                <a-select-option v-for="cat in categories" :key="cat.id" :value="cat.id">
                  {{ cat.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="Slug">
          <a-input v-model:value="form.slug" placeholder="url-friendly-identifier" />
        </a-form-item>

        <a-form-item label="摘要">
          <a-textarea v-model:value="form.summary" :rows="2" />
        </a-form-item>

        <a-form-item label="封面图URL">
          <a-input v-model:value="form.cover_url" />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="语言">
              <a-select v-model:value="form.lang">
                <a-select-option value="zh-CN">简体中文</a-select-option>
                <a-select-option value="en-US">English</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="状态">
              <a-select v-model:value="form.status">
                <a-select-option value="draft">草稿</a-select-option>
                <a-select-option value="published">发布</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="排序">
              <a-input-number v-model:value="form.sort_order" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="置顶">
          <a-switch v-model:checked="form.pinned" />
        </a-form-item>

        <a-form-item label="内容">
          <a-textarea v-model:value="form.content_html" :rows="10" placeholder="HTML内容" />
        </a-form-item>

        <a-form-item v-if="form.status === 'published'" label="发布时间">
          <a-date-picker
            v-model:value="publishedAtDate"
            show-time
            format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { message } from "ant-design-vue";
import { PlusOutlined } from "@ant-design/icons-vue";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import ProTable from "@/components/ProTable.vue";
import {
  listCmsPosts,
  listCmsCategories,
  createCmsPost,
  updateCmsPost,
  deleteCmsPost
} from "@/services/admin";

const loading = ref(false);
const submitting = ref(false);
const modalVisible = ref(false);
const isEditing = ref(false);
const posts = ref<any[]>([]);
const categories = ref<any[]>([]);
const publishedAtDate = ref<Dayjs>();

const filters = reactive({
  category_id: undefined,
  status: ""
});

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0
});

const form = reactive({
  id: undefined,
  category_id: undefined,
  title: "",
  slug: "",
  summary: "",
  content_html: "",
  cover_url: "",
  lang: "zh-CN",
  status: "draft",
  pinned: false,
  sort_order: 0,
  published_at: undefined
});

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "标题", dataIndex: "title", key: "title", ellipsis: true },
  { title: "分类", dataIndex: "category_id", key: "category_id", width: 100 },
  { title: "状态", dataIndex: "status", key: "status", width: 100 },
  { title: "发布时间", dataIndex: "published_at", key: "published_at", width: 180 },
  { title: "操作", key: "actions", width: 150, fixed: "right" }
];

const formatDate = (date: string) => {
  if (!date) return "-";
  return new Date(date).toLocaleString("zh-CN");
};

const fetchPosts = async () => {
  loading.value = true;
  try {
    const params: any = {
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize
    };
    if (filters.category_id) params.category_id = filters.category_id;
    if (filters.status) params.status = filters.status;
    const res = await listCmsPosts(params);
    posts.value = res.data?.items || [];
    pagination.total = res.data?.total || 0;
  } finally {
    loading.value = false;
  }
};

const fetchCategories = async () => {
  try {
    const res = await listCmsCategories();
    categories.value = res.data?.items || [];
  } catch (error) {
    console.error("Failed to fetch categories:", error);
  }
};

const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  fetchPosts();
};

const handleFilterChange = () => {
  pagination.current = 1;
  fetchPosts();
};

const openCreateModal = () => {
  isEditing.value = false;
  Object.assign(form, {
    id: undefined,
    category_id: undefined,
    title: "",
    slug: "",
    summary: "",
    content_html: "",
    cover_url: "",
    lang: "zh-CN",
    status: "draft",
    pinned: false,
    sort_order: 0,
    published_at: undefined
  });
  publishedAtDate.value = undefined;
  modalVisible.value = true;
};

const openEditModal = (record: any) => {
  isEditing.value = true;
  Object.assign(form, record);
  if (record.published_at) {
    publishedAtDate.value = dayjs(record.published_at);
  }
  modalVisible.value = true;
};

const handleSubmit = async () => {
  if (!form.title || !form.category_id) {
    message.error("请填写标题和选择分类");
    return;
  }

  submitting.value = true;
  try {
    const payload: any = {
      category_id: form.category_id,
      title: form.title,
      slug: form.slug,
      summary: form.summary,
      content_html: form.content_html,
      cover_url: form.cover_url,
      lang: form.lang,
      status: form.status,
      pinned: form.pinned,
      sort_order: form.sort_order
    };

    if (form.status === "published" && publishedAtDate.value) {
      payload.published_at = publishedAtDate.value.toISOString();
    }

    if (isEditing.value) {
      await updateCmsPost(form.id, payload);
    } else {
      await createCmsPost(payload);
    }

    message.success(isEditing.value ? "更新成功" : "创建成功");
    modalVisible.value = false;
    fetchPosts();
  } catch (error: any) {
    message.error(error.response?.data?.error || "操作失败");
  } finally {
    submitting.value = false;
  }
};

const handleDelete = async (record: any) => {
  try {
    await deleteCmsPost(record.id);
    message.success("删除成功");
    fetchPosts();
  } catch (error: any) {
    message.error(error.response?.data?.error || "删除失败");
  }
};

onMounted(() => {
  fetchPosts();
  fetchCategories();
});
</script>

<style scoped>
.cms-posts-page {
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

.post-title-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
