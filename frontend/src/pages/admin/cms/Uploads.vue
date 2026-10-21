<template>
  <div class="cms-uploads-page">
    <div class="page-header">
      <h1 class="page-title">文件管理</h1>
      <a-upload
        :custom-request="handleUpload"
        :show-upload-list="false"
      >
        <a-button type="primary">
          <template #icon><UploadOutlined /></template>
          上传文件
        </a-button>
      </a-upload>
    </div>

    <a-card :bordered="false">
      <ProTable
        :columns="columns"
        :data-source="uploads"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'preview'">
            <div class="file-preview-cell">
              <template v-if="isImage(record.mime)">
                <a-image
                  :src="record.url"
                  :width="50"
                  :height="50"
                  :preview="{
                    visible: previewVisible && previewImage === record.url,
                    onVisibleChange: (vis) => { if (!vis) closePreview(); }
                  }"
                  class="file-thumbnail"
                  @click="showPreview(record.url)"
                />
              </template>
              <template v-else>
                <component
                  :is="getFileIcon(record.mime)"
                  :class="['file-icon', `icon-${getFileIconType(record.mime)}`]"
                />
              </template>
            </div>
          </template>
          <template v-else-if="column.key === 'name'">
            <a :href="record.url" target="_blank" class="file-link">
              {{ record.name }}
            </a>
          </template>
          <template v-else-if="column.key === 'size'">
            {{ formatSize(record.size) }}
          </template>
          <template v-else-if="column.key === 'mime'">
            <a-tag>{{ record.mime }}</a-tag>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatDate(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button type="link" size="small" :href="record.url" target="_blank">
                查看
              </a-button>
              <a-button
                type="link"
                size="small"
                @click="copyUrl(record.url)"
              >
                复制链接
              </a-button>
            </a-space>
          </template>
        </template>
      </ProTable>
    </a-card>

    <!-- 图片预览模态框 -->
    <a-modal
      :open="previewVisible"
      :footer="null"
      :centered="true"
      :width="'80%'"
      @cancel="closePreview"
    >
      <img :src="previewImage" alt="预览" class="preview-image" />
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { message } from "ant-design-vue";
import {
  UploadOutlined,
  FileOutlined,
  FileImageOutlined,
  FilePdfOutlined,
  FileWordOutlined,
  FileExcelOutlined,
  FilePptOutlined,
  FileTextOutlined,
  FileZipOutlined,
  VideoCameraOutlined,
  AudioOutlined,
} from "@ant-design/icons-vue";
import type { UploadRequestOption } from "ant-design-vue";
import ProTable from "@/components/ProTable.vue";
import { listUploads, uploadFile } from "@/services/admin";

const loading = ref(false);
const uploads = ref<any[]>([]);
const previewImage = ref<string>("");
const previewVisible = ref(false);

// 图标映射
const iconMap: Record<string, any> = {
  "application/pdf": FilePdfOutlined,
  "application/msword": FileWordOutlined,
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document": FileWordOutlined,
  "application/vnd.ms-excel": FileExcelOutlined,
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": FileExcelOutlined,
  "application/vnd.ms-powerpoint": FilePptOutlined,
  "application/vnd.openxmlformats-officedocument.presentationml.presentation": FilePptOutlined,
  "application/zip": FileZipOutlined,
  "application/x-zip-compressed": FileZipOutlined,
  "application/x-rar-compressed": FileZipOutlined,
  "application/x-7z-compressed": FileZipOutlined,
  "text/plain": FileTextOutlined,
  "text/html": FileTextOutlined,
  "text/css": FileTextOutlined,
  "text/javascript": FileTextOutlined,
  "application/json": FileTextOutlined,
};

// 获取文件图标组件
const getFileIcon = (mime: string) => {
  // 检查是否为图片
  if (mime?.startsWith("image/")) {
    return FileImageOutlined;
  }
  // 检查是否为视频
  if (mime?.startsWith("video/")) {
    return VideoCameraOutlined;
  }
  // 检查是否为音频
  if (mime?.startsWith("audio/")) {
    return AudioOutlined;
  }
  // 从映射表获取
  return iconMap[mime] || FileOutlined;
};

// 获取文件图标类型（用于样式）
const getFileIconType = (mime: string) => {
  if (mime?.startsWith("image/")) return "image";
  if (mime?.startsWith("video/")) return "video";
  if (mime?.startsWith("audio/")) return "audio";
  if (mime?.includes("pdf")) return "pdf";
  if (mime?.includes("word") || mime?.includes("document")) return "word";
  if (mime?.includes("excel") || mime?.includes("spreadsheet")) return "excel";
  if (mime?.includes("powerpoint") || mime?.includes("presentation")) return "ppt";
  if (mime?.includes("zip") || mime?.includes("rar") || mime?.includes("7z")) return "zip";
  if (mime?.startsWith("text/")) return "text";
  return "default";
};

// 判断是否为图片
const isImage = (mime: string) => {
  return mime?.startsWith("image/");
};

// 显示图片预览
const showPreview = (url: string) => {
  previewImage.value = url;
  previewVisible.value = true;
};

// 关闭预览
const closePreview = () => {
  previewVisible.value = false;
};

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0
});

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "预览", key: "preview", width: 80 },
  { title: "文件名", dataIndex: "name", key: "name", ellipsis: true },
  { title: "大小", dataIndex: "size", key: "size", width: 100 },
  { title: "类型", dataIndex: "mime", key: "mime", width: 150 },
  { title: "上传者ID", dataIndex: "uploader_id", key: "uploader_id", width: 100 },
  { title: "上传时间", dataIndex: "created_at", key: "created_at", width: 180 },
  { title: "操作", key: "actions", width: 150, fixed: "right" }
];

const formatSize = (bytes: number) => {
  if (!bytes) return "-";
  if (bytes < 1024) return bytes + " B";
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + " KB";
  return (bytes / (1024 * 1024)).toFixed(2) + " MB";
};

const formatDate = (date: string) => {
  if (!date) return "-";
  return new Date(date).toLocaleString("zh-CN");
};

const copyUrl = (url: string) => {
  navigator.clipboard.writeText(url).then(() => {
    message.success("链接已复制到剪贴板");
  });
};

const fetchUploads = async () => {
  loading.value = true;
  try {
    const params = {
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize
    };
    const res = await listUploads(params);
    uploads.value = res.data?.items || [];
    pagination.total = res.data?.total || 0;
  } finally {
    loading.value = false;
  }
};

const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  fetchUploads();
};

const handleUpload = async (options: UploadRequestOption) => {
  const { file } = options;
  try {
    await uploadFile(file as File);
    message.success("上传成功");
    fetchUploads();
  } catch (error: any) {
    message.error(error.response?.data?.error || "上传失败");
  }
};

onMounted(() => {
  fetchUploads();
});
</script>

<style scoped>
.cms-uploads-page {
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

.file-link {
  color: var(--primary);
  text-decoration: none;
}

.file-link:hover {
  text-decoration: underline;
}

.file-preview-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 50px;
}

.file-thumbnail {
  border-radius: 4px;
  cursor: pointer;
  overflow: hidden;
}

.file-thumbnail :deep(.ant-image-img) {
  object-fit: cover;
  width: 50px;
  height: 50px;
}

.file-icon {
  font-size: 32px;
  color: var(--text-secondary);
}

.file-icon.icon-pdf {
  color: #ff4d4f;
}

.file-icon.icon-word {
  color: #1890ff;
}

.file-icon.icon-excel {
  color: #52c41a;
}

.file-icon.icon-ppt {
  color: #fa8c16;
}

.file-icon.icon-zip {
  color: #faad14;
}

.file-icon.icon-video {
  color: #722ed1;
}

.file-icon.icon-audio {
  color: #13c2c2;
}

.file-icon.icon-text {
  color: #8c8c8c;
}

.file-icon.icon-image {
  color: #52c41a;
}

.preview-image {
  width: 100%;
  max-height: 70vh;
  object-fit: contain;
}
</style>
