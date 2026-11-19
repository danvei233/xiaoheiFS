<template>
  <div ref="controlRoot" class="appearance-settings" :dir="direction">
    <details name="appearance-settings" class="appearance-control theme-control" @toggle="handleControlToggle('theme', $event)">
      <summary class="appearance-trigger" :title="t('appearance.theme')" :aria-label="t('appearance.theme')" :aria-expanded="openControl === 'theme'">
        <span class="appearance-icon" aria-hidden="true"><BgColorsOutlined /></span>
      </summary>

      <div class="appearance-popover appearance-panel">
        <header class="panel-head">
          <div class="panel-head-text">
            <h2>{{ t('appearance.title') }}</h2>
            <p>{{ t('appearance.description') }}</p>
          </div>
          <button type="button" class="panel-reset" :title="t('appearance.reset')" :aria-label="t('appearance.reset')" @click="resetAppearance">
            <BgColorsOutlined />
          </button>
        </header>

        <section class="panel-section">
          <h3>{{ t('appearance.interfaceTheme') }}</h3>
          <div class="theme-grid">
            <label v-for="preset in THEME_PRESETS" :key="preset" class="theme-choice" :class="{ selected: app.themePreset === preset }">
              <input type="radio" :name="`appearance-theme-${instanceId}`" :value="preset" :checked="app.themePreset === preset" @change="app.setThemePreset(preset)" />
              <span class="theme-preview" :class="`preview-${preset}`" :style="previewStyle(preset)" aria-hidden="true">
                <span class="preview-sidebar"><i></i><i></i><i></i></span>
                <span class="preview-main"><i></i><span><b></b><b></b></span><em></em></span>
              </span>
              <span class="theme-choice-text">
                <strong>{{ t(`theme.${preset}`) }}</strong>
                <small>{{ themeDescription(preset) }}</small>
              </span>
              <span v-if="app.themePreset === preset" class="choice-dot" aria-hidden="true">
                <CheckOutlined />
              </span>
            </label>
          </div>
        </section>

        <div class="panel-row">
          <div class="panel-row-text">
            <strong>{{ t('appearance.colorMode') }}</strong>
          </div>
          <div class="mode-segment" role="group" :aria-label="t('appearance.colorMode')">
            <button
              v-for="mode in modes"
              :key="mode"
              type="button"
              class="mode-option"
              :class="{ active: app.consoleThemeMode === mode }"
              :aria-pressed="app.consoleThemeMode === mode"
              @click="app.setConsoleThemeMode(mode)"
            >
              <span class="mode-icon" aria-hidden="true">
                <DesktopOutlined v-if="mode === 'auto'" />
                <svg v-else-if="mode === 'light'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"><circle cx="8" cy="8" r="2.8" /><path d="M8 1.4v1.5M8 13.1v1.5M1.4 8h1.5M13.1 8h1.5M3.4 3.4l1 1M11.6 11.6l1 1M12.6 3.4l-1 1M4.4 11.6l-1 1" /></svg>
                <svg v-else viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M13.5 10.2A5.9 5.9 0 0 1 5.8 2.5 5.9 5.9 0 1 0 13.5 10.2Z" /></svg>
              </span>
              {{ t(`appearance.${mode}`) }}
            </button>
          </div>
        </div>

        <label class="panel-row panel-row-toggle">
          <div class="panel-row-text">
            <strong>{{ t('appearance.transparent') }}</strong>
            <small>{{ t('appearance.transparentHint') }}</small>
          </div>
          <input type="checkbox" :checked="app.transparentSurface" @change="app.setTransparentSurface(($event.target as HTMLInputElement).checked)" />
          <span class="theme-switch" aria-hidden="true"></span>
        </label>

        <label class="panel-row panel-row-toggle">
          <div class="panel-row-text">
            <strong>{{ t('appearance.rounded') }}</strong>
            <small>{{ t('appearance.roundedHint') }}</small>
          </div>
          <input type="checkbox" :checked="app.roundedCorners" @change="app.setRoundedCorners(($event.target as HTMLInputElement).checked)" />
          <span class="theme-switch" aria-hidden="true"></span>
        </label>

        <div class="panel-row">
          <div class="panel-row-text">
            <strong>{{ t('appearance.font') }}</strong>
            <small>{{ t('appearance.fontHint') }}</small>
          </div>
          <select class="font-select" :value="app.fontStyle" :aria-label="t('appearance.font')" @change="app.setFontStyle(($event.target as HTMLSelectElement).value as FontStyle)">
            <option value="mixed">{{ t('appearance.fontMixed') }}</option>
            <option value="mono">{{ t('appearance.fontMono') }}</option>
          </select>
        </div>
      </div>
    </details>

    <details name="appearance-settings" class="appearance-control language-control" @toggle="handleControlToggle('language', $event)">
      <summary class="appearance-trigger" :title="t('appearance.language')" :aria-label="t('appearance.language')" :aria-expanded="openControl === 'language'">
        <span class="appearance-icon" aria-hidden="true"><GlobalOutlined /></span>
      </summary>
      <div class="appearance-popover language-popover">
        <fieldset>
          <legend>{{ t('appearance.language') }}</legend>
          <div class="language-grid">
            <label v-for="language in languages" :key="language.code" class="language-choice" :class="{ selected: localePreference === language.code }" :lang="language.code">
              <input type="radio" :name="`appearance-language-${instanceId}`" :value="language.code" :checked="localePreference === language.code" @change="setLocale(language.code)" />
              <span dir="auto"><strong>{{ language.label }}</strong><small>{{ language.englishLabel }}</small></span>
              <CheckOutlined v-if="localePreference === language.code" class="choice-check" aria-hidden="true" />
            </label>
            <label class="language-choice language-auto" :class="{ selected: localePreference === 'auto' }">
              <input type="radio" :name="`appearance-language-${instanceId}`" value="auto" :checked="localePreference === 'auto'" @change="setLocale('auto')" />
              <span><strong>{{ t('appearance.browser') }}</strong><small>{{ locale }}</small></span>
              <CheckOutlined v-if="localePreference === 'auto'" class="choice-check" aria-hidden="true" />
            </label>
          </div>
        </fieldset>
      </div>
    </details>
  </div>
</template>

<script setup lang="ts">
import { getCurrentInstance, ref } from 'vue';
import { BgColorsOutlined, CheckOutlined, DesktopOutlined, GlobalOutlined } from '@ant-design/icons-vue';
import { useAppStore, type FontStyle } from '@/stores/app';
import { THEME_PRESETS, getThemeAppearance, type ThemePreset } from '@/theme';
import { direction, languages, locale, localePreference, setLocale, t } from '@/locales';
import { themeDescriptions } from '@/locales/messages';

const app = useAppStore();
const instanceId = getCurrentInstance()?.uid ?? 'global';
const controlRoot = ref<HTMLElement | null>(null);
const openControl = ref<'theme' | 'language' | null>(null);
const modes = ['auto', 'light', 'dark'] as const;

const previewStyle = (preset: ThemePreset) => getThemeAppearance(preset, app.isDarkMode, {
  rounded: app.roundedCorners, transparent: app.transparentSurface
}).variables;

const themeDescription = (preset: ThemePreset) => themeDescriptions[locale.value][preset] ?? '';

function resetAppearance() {
  app.setThemePreset('executive');
  app.setConsoleThemeMode('auto');
  app.setTransparentSurface(false);
  app.setRoundedCorners(true);
  app.setFontStyle('mixed');
}

function handleControlToggle(name: 'theme' | 'language', event: Event) {
  const current = event.currentTarget as HTMLDetailsElement;
  if (current.open) {
    openControl.value = name;
    controlRoot.value?.querySelectorAll<HTMLDetailsElement>('details[open]').forEach((item) => {
      if (item !== current) item.open = false;
    });
  } else if (openControl.value === name) {
    openControl.value = null;
  }
}
</script>

<style scoped>
.appearance-settings { display: inline-flex; align-items: center; gap: 6px; color: var(--text-primary); }
.appearance-control { position: relative; flex: 0 0 auto; }
.appearance-control > summary { list-style: none; }
.appearance-control > summary::-webkit-details-marker { display: none; }
.appearance-trigger {
  display: inline-flex;
  width: 38px;
  min-width: 38px;
  height: 38px;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: var(--theme-control-radius, var(--radius-md, 8px));
  background: var(--theme-control-bg, var(--card));
  color: var(--text-primary);
  cursor: pointer;
  font: inherit;
  transition: color 140ms ease, border-color 140ms ease, background 140ms ease, box-shadow 140ms ease;
}
.appearance-trigger:hover, .appearance-control[open] > .appearance-trigger {
  color: var(--primary);
  border-color: var(--primary);
  background: var(--theme-control-hover, var(--card));
}
.appearance-trigger:focus-visible, .appearance-control :focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 3px;
}
.appearance-icon { display: inline-flex; align-items: center; justify-content: center; font-size: 16px; }

.appearance-popover {
  position: absolute;
  z-index: 1200;
  inset-block-start: calc(100% + 8px);
  inset-inline-end: 0;
  width: max-content;
  min-width: 200px;
  max-width: min(340px, calc(100vw - 24px));
  max-height: min(78vh, 640px);
  overflow: auto;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--theme-card-radius, var(--radius-lg, 12px));
  background: var(--theme-surface, var(--card));
  color: var(--text-primary);
  box-shadow: var(--theme-shadow-popover, 0 12px 28px #10233a24);
  backdrop-filter: var(--theme-backdrop, none);
  -webkit-backdrop-filter: var(--theme-backdrop, none);
}

/* Appearance panel mirrors the reference layout: header, theme list, mode, toggles, font. */
.appearance-panel {
  width: 460px;
  max-width: min(460px, calc(100vw - 24px));
  max-height: min(90vh, 820px);
  padding: 16px 18px;
  border-radius: 16px;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.panel-head-text h2 {
  margin: 0;
  color: var(--text-primary);
  font-size: 20px;
  font-weight: 700;
  letter-spacing: -.02em;
}

.panel-head-text p {
  margin: 4px 0 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.panel-reset {
  display: inline-flex;
  width: 40px;
  height: 40px;
  flex: 0 0 40px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--theme-control-bg, var(--card));
  color: var(--primary);
  font-size: 17px;
  cursor: pointer;
  transition: border-color 140ms ease, background 140ms ease;
}

.panel-reset:hover {
  border-color: var(--primary);
  background: color-mix(in srgb, var(--primary) 8%, var(--card));
}

.panel-section h3 {
  margin: 0 0 10px;
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 650;
}

.theme-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 7px;
}

.theme-choice {
  position: relative;
  display: grid;
  grid-template-columns: 46px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  min-width: 0;
  min-height: 60px;
  padding: 7px 10px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card);
  color: var(--text-primary);
  cursor: pointer;
  transition: border-color 140ms ease, background 140ms ease;
}

.theme-choice:hover { border-color: color-mix(in srgb, var(--primary) 45%, var(--border)); }

.theme-choice.selected {
  border-color: var(--primary);
  background: color-mix(in srgb, var(--primary) 8%, var(--card));
}

.theme-choice input,
.language-choice input {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip-path: inset(50%);
  white-space: nowrap;
}

.theme-preview {
  display: flex;
  width: 46px;
  height: 36px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--theme-canvas);
  background-size: 12px 12px;
}

.preview-sidebar { display: flex; width: 23%; flex-direction: column; gap: 2px; padding: 5px 3px; background: var(--theme-sidebar); }
.preview-sidebar i { display: block; height: 2px; background: var(--theme-sidebar-muted); opacity: .62; }
.preview-sidebar i:first-child { height: 4px; background: var(--theme-sidebar-accent); opacity: 1; }
.preview-main { flex: 1; padding: 4px; }
.preview-main > i { display: block; width: 48%; height: 3px; margin-bottom: 3px; background: var(--primary); }
.preview-main > span { display: flex; gap: 3px; }
.preview-main b { display: block; height: 9px; flex: 1; border: 1px solid var(--border); border-radius: 3px; background: var(--theme-surface); }
.preview-main em { display: block; height: 5px; margin-top: 3px; border-radius: 3px; background: var(--theme-surface); }
.preview-tech { background-image: linear-gradient(var(--primary-light)22 1px, transparent 1px), linear-gradient(90deg, var(--primary-light)22 1px, transparent 1px); }
.preview-cute .preview-main b:first-child { background: var(--primary-light); }
.preview-glass .preview-main b { backdrop-filter: blur(4px); }

.theme-choice-text { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.theme-choice-text strong { font-size: 13.5px; font-weight: 650; line-height: 1.3; }
.theme-choice-text small {
  display: block;
  overflow: hidden;
  color: var(--text-secondary);
  font-size: 10.5px;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.choice-dot {
  position: absolute;
  inset-block-start: 50%;
  inset-inline-end: 8px;
  display: inline-flex;
  width: 18px;
  height: 18px;
  align-items: center;
  justify-content: center;
  margin-block-start: -9px;
  border-radius: 50%;
  background: var(--primary);
  color: var(--theme-primary-foreground, #fff);
  font-size: 10px;
}

.panel-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 2px;
  border-top: 1px solid var(--border);
}

.panel-section + .panel-row { margin-top: 10px; }
.panel-row-toggle { cursor: pointer; }
.panel-row-text { display: flex; min-width: 0; flex-direction: column; gap: 2px; }
.panel-row-text strong { color: var(--text-primary); font-size: 14px; font-weight: 650; }
.panel-row-text small { color: var(--text-secondary); font-size: 12px; line-height: 1.45; }

.mode-segment {
  display: inline-flex;
  flex: 0 0 auto;
  gap: 2px;
  padding: 3px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--bg-secondary);
}

.mode-option {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 550;
  cursor: pointer;
  transition: background 140ms ease, color 140ms ease, box-shadow 140ms ease;
}

.mode-option:hover { color: var(--text-primary); }

.mode-option.active {
  background: var(--card);
  color: var(--primary);
  box-shadow: 0 1px 3px rgb(15 30 50 / 10%);
}

.mode-icon { display: inline-flex; align-items: center; font-size: 14px; }
.mode-icon svg { display: block; width: 15px; height: 15px; }

.panel-row-toggle input { position: absolute; width: 1px; height: 1px; opacity: 0; }

.theme-switch {
  position: relative;
  flex: 0 0 44px;
  width: 44px;
  height: 24px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--text-primary) 22%, transparent);
  transition: background 160ms ease;
}

.theme-switch::after {
  content: "";
  position: absolute;
  inset-block-start: 3px;
  inset-inline-start: 3px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 1px 2px rgb(0 0 0 / 20%);
  transition: transform 160ms ease;
}

.panel-row-toggle input:checked + .theme-switch { background: var(--primary); }
.panel-row-toggle input:checked + .theme-switch::after { transform: translateX(20px); }
.panel-row-toggle input:focus-visible + .theme-switch { outline: 2px solid var(--primary); outline-offset: 2px; }
[dir="rtl"] .panel-row-toggle input:checked + .theme-switch::after { transform: translateX(-20px); }

.font-select {
  min-width: 150px;
  height: 38px;
  padding: 0 32px 0 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--theme-control-bg, var(--card));
  color: var(--text-primary);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}

.font-select:focus-visible { outline: 2px solid var(--primary); outline-offset: 2px; }

.language-popover { width: 356px; max-width: min(356px, calc(100vw - 24px)); padding: 10px; }
.appearance-popover fieldset { min-width: 0; border: 0; padding: 0; margin: 0; }
.appearance-popover legend { display: block; padding: 0 2px 8px; color: var(--text-secondary); font-size: 12px; font-weight: 650; }
.language-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 2px 4px; }
.language-choice {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 16px;
  align-items: center;
  min-width: 0;
  min-height: 50px;
  padding: 6px 8px;
  border: 1px solid transparent;
  border-radius: 10px;
  color: var(--text-primary);
  cursor: pointer;
  transition: background 140ms ease, border-color 140ms ease, color 140ms ease;
}
.language-choice > span { grid-column: 1; min-width: 0; }
.language-choice strong, .language-choice small { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.language-choice strong { font-size: 13px; font-weight: 650; line-height: 1.35; }
.language-choice small { margin-top: 1px; color: var(--text-secondary); font-size: 10px; line-height: 1.35; }
.language-choice .choice-check { grid-column: 2; grid-row: 1; color: var(--primary); font-size: 12px; }
.language-choice:hover { background: var(--bg-secondary); }
.language-choice.selected { border-color: color-mix(in srgb, var(--primary) 35%, var(--border)); background: color-mix(in srgb, var(--primary) 8%, var(--card)); }
.language-choice:focus-within { outline: 2px solid var(--primary); outline-offset: 1px; }
.language-auto { grid-column: 1 / -1; min-height: 42px; }

@media (max-width: 680px) {
  .appearance-settings { gap: 4px; }
  .appearance-trigger { width: 36px; min-width: 36px; height: 36px; padding: 0; }
  .appearance-popover { position: fixed; inset-block-start: 60px; inset-inline-start: 8px; inset-inline-end: auto; max-width: min(356px, calc(100vw - 16px)); }
  .appearance-panel { width: min(460px, calc(100vw - 16px)); max-width: calc(100vw - 16px); padding: 14px; }
  .theme-grid { grid-template-columns: minmax(0, 1fr); }
  .theme-choice { min-height: 60px; }
  .panel-row { flex-wrap: wrap; }
  .mode-option { padding: 0 9px; font-size: 12px; }
  .language-popover { width: min(356px, calc(100vw - 16px)); min-width: 0; max-width: calc(100vw - 16px); }
  .language-choice { min-height: 46px; padding: 5px 6px; }
  .language-choice strong { font-size: 12px; }
  .language-choice small { font-size: 9px; }
}
@media (max-width: 360px) {
  .appearance-settings { gap: 3px; }
  .appearance-trigger { width: 32px; min-width: 32px; height: 32px; }
  .appearance-panel, .language-popover { max-height: min(82vh, 520px); }
}
@media (prefers-reduced-motion: reduce) {
  .appearance-trigger, .theme-choice, .theme-switch, .theme-switch::after, .mode-option { transition: none; }
}
@supports not ((backdrop-filter: blur(1px)) or (-webkit-backdrop-filter: blur(1px))) {
  .appearance-popover { background: var(--card); backdrop-filter: none; -webkit-backdrop-filter: none; }
}
@media (prefers-reduced-transparency: reduce) {
  .appearance-popover { background: var(--card); backdrop-filter: none; -webkit-backdrop-filter: none; }
}
</style>
