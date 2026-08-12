<template>
  <ArtStatusTag :status="status" :text-map="textMap" />
</template>

<script setup lang="ts">
  import ArtStatusTag from '../art-status-tag/index.vue'

  defineOptions({ name: 'VpsStatusTag' })

  withDefaults(
    defineProps<{
      status?: string
    }>(),
    {
      status: ''
    }
  )

  const textMap = {
    running: '运行中',
    stopped: '关机',
    locked: '锁定',
    expired_locked: '已到期',
    normal: '正常',
    abuse: '滥用',
    fraud: '欺诈',
    provisioning: '创建中',
    active: '已开通',
    reinstalling: '重装系统中',
    reinstall_failed: '重装系统失败',
    failed: '创建失败',
    deleting: '删除中',
    rescue: '救援模式',
    cracking_password: '破解密码中',
    unknown: '未知'
  }
</script>
