<template>
  <div class="preview-footer">
    <p class="footer-desc">{{ content.description || 'Footer 描述' }}</p>
    <div class="footer-sections">
      <div class="footer-section" v-for="(section, index) in content.sections || []" :key="index">
        <div class="section-title">{{ section.title }}</div>
        <div class="section-links">
          <span v-for="(link, linkIndex) in section.links || []" :key="linkIndex">{{ link.label }}</span>
        </div>
      </div>
    </div>
    <div class="footer-badges">
      <a-tag v-for="(badge, index) in content.badges || []" :key="index">{{ badge }}</a-tag>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ content: { description?: string; sections?: Array<{ title?: string; links?: Array<{ label?: string }> }>; badges?: string[] } }>();
</script>

<style scoped>
.preview-footer {
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 12px;
  padding: 16px;
  background: #0f172a;
  color: #e2e8f0;
}

.footer-desc {
  margin-bottom: 12px;
}

.footer-sections {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 12px;
}

.section-title {
  font-weight: 600;
  margin-bottom: 6px;
}

.section-links {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  color: #cbd5f5;
}

.footer-badges {
  margin-top: 12px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
