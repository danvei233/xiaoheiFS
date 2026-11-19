<template>
  <div class="theme-preference-controls">
    <ElDropdown
      trigger="click"
      placement="bottom-end"
      popper-class="theme-preference-popper theme-mode-popper"
      @command="selectSystemMode"
    >
      <button
        type="button"
        class="theme-preference-trigger"
        :aria-label="`${t('setting.theme.title')}: ${modeLabel(settingStore.systemThemeMode)}`"
        :title="`${t('setting.theme.title')}: ${modeLabel(settingStore.systemThemeMode)}`"
      >
        <ArtSvgIcon :icon="modeIcon" />
      </button>
      <template #dropdown>
        <ElDropdownMenu class="theme-mode-menu">
          <ElDropdownItem
            v-for="mode in systemModes"
            :key="mode"
            :command="mode"
            :class="{ 'is-selected': settingStore.systemThemeMode === mode }"
          >
            <span>{{ modeLabel(mode) }}</span>
            <ArtSvgIcon
              v-if="settingStore.systemThemeMode === mode"
              icon="ri:check-line"
              class="theme-menu-check"
            />
          </ElDropdownItem>
        </ElDropdownMenu>
      </template>
    </ElDropdown>

    <ElDropdown
      trigger="click"
      placement="bottom-end"
      popper-class="theme-preference-popper theme-appearance-popper"
      @command="selectAppearance"
    >
      <button
        type="button"
        class="theme-preference-trigger"
        :aria-label="t('appearance.theme')"
        :title="t('appearance.theme')"
      >
        <ArtSvgIcon icon="ri:palette-line" />
      </button>
      <template #dropdown>
        <ElDropdownMenu class="theme-appearance-menu">
          <li class="theme-appearance-menu__heading" role="presentation">
            {{ t('appearance.theme') }}
          </li>
          <ElDropdownItem
            v-for="appearance in appearanceOptions"
            :key="appearance"
            :command="appearance"
            class="theme-appearance-item"
            :class="{ 'is-current': settingStore.appearanceTheme === appearance }"
          >
            <span class="theme-appearance-item__preview" :class="`preview-${appearance}`">
              <i class="theme-appearance-item__sidebar"></i>
              <i class="theme-appearance-item__content">
                <b class="theme-appearance-item__top"></b>
                <b class="theme-appearance-item__card"></b>
                <b class="theme-appearance-item__card"></b>
              </i>
            </span>
            <span class="theme-appearance-item__label">{{ t(`appearance.${appearance}`) }}</span>
            <ArtSvgIcon
              v-if="settingStore.appearanceTheme === appearance"
              icon="ri:check-line"
              class="theme-appearance-item__check"
            />
          </ElDropdownItem>
        </ElDropdownMenu>
      </template>
    </ElDropdown>

    <ElDropdown
      v-if="props.showLanguage"
      trigger="click"
      placement="bottom-end"
      popper-class="theme-preference-popper theme-language-popper"
      @command="changeLanguage"
    >
      <button
        type="button"
        class="theme-preference-trigger"
        :aria-label="t('language.title')"
        :title="t('language.title')"
      >
        <ArtSvgIcon icon="ri:translate-2" />
      </button>
      <template #dropdown>
        <ElDropdownMenu class="theme-language-menu">
          <ElDropdownItem
            v-for="language in appearanceLanguages"
            :key="language.value"
            :command="language.value"
            class="theme-language-item"
            :class="{ 'is-selected': !userStore.languageAuto && locale === language.value }"
          >
            <span class="theme-language-item__label" :lang="language.tag">
              <strong>{{ language.label }}</strong>
              <small>{{ language.englishLabel }}</small>
            </span>
            <ArtSvgIcon
              v-if="!userStore.languageAuto && locale === language.value"
              icon="ri:check-line"
              class="theme-menu-check"
            />
          </ElDropdownItem>
          <ElDropdownItem command="auto" class="theme-language-auto" :class="{ 'is-selected': userStore.languageAuto }">
            <span>{{ t('language.browserAuto') }}</span>
            <ArtSvgIcon v-if="userStore.languageAuto" icon="ri:check-line" class="theme-menu-check" />
          </ElDropdownItem>
        </ElDropdownMenu>
      </template>
    </ElDropdown>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { LanguageEnum, SystemThemeEnum } from '@/enums/appEnum'
  import { appearanceThemes, type AppearanceTheme } from '@/config/appearance'
  import { languageOptions } from '@/locales'
  import { useSettingStore } from '@/store/modules/setting'
  import { useUserStore } from '@/store/modules/user'
  import { useTheme } from '@/hooks/core/useTheme'

  defineOptions({ name: 'ThemePreferenceControls' })

  const props = withDefaults(defineProps<{ showLanguage?: boolean }>(), { showLanguage: true })
  const { locale, t } = useI18n()
  const settingStore = useSettingStore()
  const userStore = useUserStore()
  const { switchThemeStyles } = useTheme()

  const appearanceOptions = Object.keys(appearanceThemes) as AppearanceTheme[]
  const appearanceLanguages = [
    LanguageEnum.ZH,
    LanguageEnum.ZH_TW,
    LanguageEnum.EN,
    LanguageEnum.JA,
    LanguageEnum.KO,
    LanguageEnum.DE,
    LanguageEnum.ES,
    LanguageEnum.PT,
    LanguageEnum.IT,
    LanguageEnum.AR,
    LanguageEnum.RU,
    LanguageEnum.FR
  ].map((value) => languageOptions.find((item) => item.value === value)!)
  const systemModes = [SystemThemeEnum.AUTO, SystemThemeEnum.LIGHT, SystemThemeEnum.DARK]
  const modeIndexes: Record<SystemThemeEnum, number> = {
    [SystemThemeEnum.LIGHT]: 0,
    [SystemThemeEnum.DARK]: 1,
    [SystemThemeEnum.AUTO]: 2
  }

  const modeLabel = (mode: SystemThemeEnum) => t(`setting.theme.list[${modeIndexes[mode]}]`)
  const modeIcon = computed(() => {
    if (settingStore.systemThemeMode === SystemThemeEnum.AUTO) return 'ri:computer-line'
    return settingStore.isDark ? 'ri:moon-line' : 'ri:sun-line'
  })

  const selectAppearance = (appearance: string) => {
    if (Object.hasOwn(appearanceThemes, appearance)) {
      settingStore.setAppearanceTheme(appearance as AppearanceTheme)
    }
  }

  const selectSystemMode = (mode: SystemThemeEnum) => {
    switchThemeStyles(mode)
  }

  const changeLanguage = (language: LanguageEnum | 'auto') => {
    userStore.setLanguage(language)
    locale.value = userStore.language
  }
</script>

<style lang="scss">
  .theme-preference-controls {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .theme-preference-trigger {
    display: inline-flex;
    width: 32px;
    height: 32px;
    align-items: center;
    justify-content: center;
    padding: 0;
    color: inherit;
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: var(--theme-control-radius, 7px);
    transition:
      color 0.16s ease,
      background-color 0.16s ease,
      transform 0.16s ease;
  }

  .theme-preference-trigger:hover {
    background-color: var(--art-hover-color);
    transform: translateY(-1px);
  }

  .theme-preference-trigger:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }

  .theme-preference-trigger .art-svg-icon {
    font-size: 18px;
  }

  .theme-appearance-menu {
    display: grid;
    width: 452px;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 7px;
    padding: 10px !important;
  }

  .theme-appearance-menu__heading {
    grid-column: 1 / -1;
    padding: 3px 6px 7px;
    color: var(--theme-muted-text);
    font-size: 12px;
    font-weight: 650;
    list-style: none;
  }

  .el-dropdown-menu__item.theme-appearance-item {
    position: relative;
    display: flex;
    min-width: 0;
    min-height: 72px;
    flex-direction: row;
    align-items: center;
    justify-content: flex-start;
    gap: 9px;
    padding: 7px !important;
    white-space: normal;
    border: 1px solid var(--theme-border);
    border-radius: 10px;
  }

  .el-dropdown-menu__item.theme-appearance-item.is-current {
    color: var(--theme-text) !important;
    background-color: var(--theme-surface-soft) !important;
    border-color: var(--el-color-primary);
  }

  .theme-appearance-item__preview {
    display: flex;
    width: 78px;
    height: 48px;
    flex: 0 0 78px;
    gap: 2px;
    overflow: hidden;
    background: #f2f5f8;
    border: 1px solid rgb(31 45 61 / 12%);
    border-radius: 4px;
  }

  .theme-appearance-item__sidebar {
    width: 9px;
    flex: 0 0 9px;
    background: #ffffff;
  }

  .theme-appearance-item__content {
    display: grid;
    min-width: 0;
    flex: 1;
    grid-template-rows: 7px 1fr;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 3px;
    padding: 3px;
    font-style: normal;
  }

  .theme-appearance-item__top {
    grid-column: 1 / -1;
    background: #ffffff;
    border-radius: 2px;
  }

  .theme-appearance-item__card {
    background: #ffffff;
    border-radius: 2px;
  }

  .preview-minimal {
    background: #f2f4f7;
  }

  .preview-minimal .theme-appearance-item__sidebar,
  .preview-minimal .theme-appearance-item__top {
    background: #ffffff;
  }

  .preview-tech {
    background: #e9f5fc;
  }

  .preview-tech .theme-appearance-item__sidebar {
    background: #d6ebf8;
  }

  .preview-tech .theme-appearance-item__top {
    background: #b8def2;
  }

  .preview-executive {
    background: #edf2f7;
  }

  .preview-executive .theme-appearance-item__sidebar {
    background: #dce9f5;
  }

  .preview-executive .theme-appearance-item__top {
    background: #14263d;
  }

  .preview-gold {
    background: #f6f0e2;
  }

  .preview-gold .theme-appearance-item__sidebar {
    background: #e8ddc3;
  }

  .preview-gold .theme-appearance-item__top {
    background: #d59b32;
  }

  .preview-cute {
    background: #e8f7ef;
    border-radius: 9px;
  }

  .preview-cute .theme-appearance-item__sidebar {
    background: #d5f0e2;
    border-radius: 6px;
  }

  .preview-cute .theme-appearance-item__top,
  .preview-cute .theme-appearance-item__card {
    border-radius: 5px;
  }

  .preview-glass {
    background: linear-gradient(135deg, #eeeafd 10%, #dcd6f8 55%, #f7f5ff);
    border-color: rgb(112 95 187 / 34%);
    box-shadow: inset 0 1px 0 rgb(255 255 255 / 85%);
  }

  .preview-glass .theme-appearance-item__sidebar,
  .preview-glass .theme-appearance-item__top,
  .preview-glass .theme-appearance-item__card {
    background: rgb(255 255 255 / 78%);
    border: 1px solid rgb(255 255 255 / 72%);
  }

  .theme-appearance-item__label {
    min-width: 0;
    flex: 1;
    overflow: hidden;
    color: var(--theme-text);
    font-size: 13px;
    font-weight: 600;
    line-height: 18px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .theme-appearance-item__check {
    position: absolute;
    top: 6px;
    right: 7px;
    color: var(--el-color-primary);
    font-size: 12px;
  }

  .theme-mode-menu,
  .theme-language-menu {
    min-width: 176px;
    padding: 6px !important;
  }

  .theme-language-menu {
    display: grid;
    width: 524px;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 3px 7px;
    max-height: min(72vh, 520px);
    overflow: auto;
  }

  .theme-mode-menu .el-dropdown-menu__item,
  .theme-language-menu .el-dropdown-menu__item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
  }

  .theme-language-menu .el-dropdown-menu__item.theme-language-item {
    min-width: 0;
    min-height: 62px;
    padding: 8px 10px;
    white-space: normal;
    border: 1px solid transparent;
    border-radius: 10px;
  }

  .theme-language-item.is-selected {
    color: var(--theme-text) !important;
    background-color: var(--theme-surface-soft) !important;
    border-color: var(--el-color-primary) !important;
  }

  .theme-language-item__label {
    display: block;
    min-width: 0;
    flex: 1;
    line-height: 1.35;
  }

  .theme-language-item__label strong,
  .theme-language-item__label small {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .theme-language-item__label strong {
    font-size: 14px;
    font-weight: 650;
  }

  .theme-language-item__label small {
    margin-top: 2px;
    color: var(--theme-muted-text);
    font-size: 11px;
  }

  .theme-language-menu .el-dropdown-menu__item.theme-language-auto {
    grid-column: 1 / -1;
    min-height: 42px;
    border-top: 1px solid var(--theme-border);
  }

  @media (max-width: 600px) {
    .theme-preference-popper.theme-appearance-popper .theme-appearance-menu {
      width: min(452px, calc(100vw - 20px));
    }

    .theme-preference-popper.theme-language-popper .theme-language-menu {
      width: min(524px, calc(100vw - 20px));
      gap: 2px;
    }

    .theme-language-menu .el-dropdown-menu__item.theme-language-item {
      min-height: 56px;
      padding: 6px;
    }

    .theme-language-item__label strong {
      font-size: 12px;
    }

    .theme-language-item__label small {
      font-size: 10px;
    }
  }

  .theme-menu-check {
    color: var(--el-color-primary);
  }
</style>
