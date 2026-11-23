<template>
  <SectionTitle :title="$t('appearance.title')" />
  <div class="appearance-setting-summary">
    <strong>{{ $t(`appearance.${settingStore.appearanceTheme}`) }}</strong>
    <span>{{ $t('setting.theme.title') }} · {{ currentModeLabel }}</span>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { useI18n } from 'vue-i18n'
  import SectionTitle from './SectionTitle.vue'
  import { SystemThemeEnum } from '@/enums/appEnum'
  import { useSettingStore } from '@/store/modules/setting'

  const settingStore = useSettingStore()
  const { t } = useI18n()
  const modeIndex: Record<SystemThemeEnum, number> = {
    [SystemThemeEnum.LIGHT]: 0,
    [SystemThemeEnum.DARK]: 1,
    [SystemThemeEnum.AUTO]: 2
  }
  const currentModeLabel = computed(() =>
    t(`setting.theme.list[${modeIndex[settingStore.systemThemeMode]}]`)
  )
</script>

<style lang="scss" scoped>
  .appearance-setting-summary {
    display: grid;
    gap: 8px;
    padding: 12px 14px;
    background: var(--theme-surface-soft);
    border: 1px solid var(--theme-border);
    border-radius: var(--theme-control-radius, 7px);
  }

  .appearance-setting-summary strong {
    color: var(--theme-text);
    font-weight: 600;
  }

  .appearance-setting-summary span {
    color: var(--theme-muted-text);
    font-size: 12px;
  }
</style>
