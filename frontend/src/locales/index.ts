import { computed, ref, watch } from 'vue';
import zhCN from 'ant-design-vue/es/locale/zh_CN';
import zhTW from 'ant-design-vue/es/locale/zh_TW';
import enUS from 'ant-design-vue/es/locale/en_US';
import jaJP from 'ant-design-vue/es/locale/ja_JP';
import koKR from 'ant-design-vue/es/locale/ko_KR';
import frFR from 'ant-design-vue/es/locale/fr_FR';
import deDE from 'ant-design-vue/es/locale/de_DE';
import esES from 'ant-design-vue/es/locale/es_ES';
import itIT from 'ant-design-vue/es/locale/it_IT';
import ptPT from 'ant-design-vue/es/locale/pt_PT';
import ruRU from 'ant-design-vue/es/locale/ru_RU';
import arSA from 'ant-design-vue/es/locale/ar_EG';
import { messages, type LocaleCode } from './messages';
import { shellMessages } from './shell';
import { dictionaries } from './dict';
export type { LocaleCode } from './messages';

export const languages: { code: LocaleCode; label: string; englishLabel: string }[] = [
  { code: 'zh-CN', label: '简体中文', englishLabel: 'Chinese (Simplified)' },
  { code: 'zh-TW', label: '繁體中文', englishLabel: 'Chinese (Traditional)' },
  { code: 'en-US', label: 'English', englishLabel: 'English' },
  { code: 'ja-JP', label: '日本語', englishLabel: 'Japanese' },
  { code: 'ko-KR', label: '한국어', englishLabel: 'Korean' },
  { code: 'de-DE', label: 'Deutsch', englishLabel: 'German' },
  { code: 'es-ES', label: 'Español', englishLabel: 'Spanish' },
  { code: 'pt-PT', label: 'Português', englishLabel: 'Portuguese' },
  { code: 'it-IT', label: 'Italiano', englishLabel: 'Italian' },
  { code: 'ar-SA', label: 'العربية', englishLabel: 'Arabic' },
  { code: 'ru-RU', label: 'Русский', englishLabel: 'Russian' },
  { code: 'fr-FR', label: 'Français', englishLabel: 'French' }
];
const storageKey = 'xiaohei.locale';
const antLocales = { 'zh-CN': zhCN, 'zh-TW': zhTW, 'en-US': enUS, 'ja-JP': jaJP, 'ko-KR': koKR, 'fr-FR': frFR, 'de-DE': deDE, 'es-ES': esES, 'it-IT': itIT, 'pt-PT': ptPT, 'ru-RU': ruRU, 'ar-SA': arSA };

/** Match regional variants, preserving traditional Chinese scripts and regions. */
export function matchLocale(value: string): LocaleCode | undefined {
  const code = value.toLowerCase().replace(/_/g, '-');
  if (code === 'zh' || code.startsWith('zh-')) {
    return /(?:^|-)(hant|tw|hk|mo)(?:-|$)/.test(code) ? 'zh-TW' : 'zh-CN';
  }
  return languages.find(item => item.code.split('-')[0] === code.split('-')[0])?.code;
}
export function detectBrowserLocale(): LocaleCode {
  if (typeof navigator === 'undefined') return 'en-US';
  for (const candidate of navigator.languages?.length ? navigator.languages : [navigator.language]) {
    const matched = matchLocale(candidate);
    if (matched) return matched;
  }
  return 'en-US';
}
function readPreference(): LocaleCode | 'auto' {
  try {
    const value = localStorage.getItem(storageKey);
    return languages.some(item => item.code === value) ? value as LocaleCode : 'auto';
  } catch { return 'auto'; }
}
export const localePreference = ref<LocaleCode | 'auto'>(readPreference());
export const locale = ref<LocaleCode>(localePreference.value === 'auto' ? detectBrowserLocale() : localePreference.value);
export const direction = computed(() => locale.value === 'ar-SA' ? 'rtl' : 'ltr');
export const antdLocale = computed(() => antLocales[locale.value]);

/** Look up one locale table; dictionaries win over the legacy array tables. */
function lookup(code: LocaleCode, key: string): string {
  return dictionaries[code]?.[key] ?? messages[code]?.[key] ?? shellMessages[code]?.[key] ?? '';
}

/**
 * Resolve a key for the active locale, then fall back to English and finally to
 * simplified Chinese. Unknown keys return an empty string so existing
 * `$t(key) || "中文"` call sites keep their inline default.
 */
export function t(key: string, params?: Record<string, string | number>): string {
  const value = lookup(locale.value, key) || lookup('en-US', key) || lookup('zh-CN', key);
  if (!value) return '';
  if (!params) return value;
  return value.replace(/\{(\w+)\}/g, (match, name: string) => (
    params[name] === undefined ? match : String(params[name])
  ));
}
export function setLocale(value: string = 'auto') {
  if (value !== 'auto' && !languages.some(item => item.code === value)) return;
  localePreference.value = value as LocaleCode | 'auto';
  locale.value = value === 'auto' ? detectBrowserLocale() : value as LocaleCode;
  try {
    if (value === 'auto') localStorage.removeItem(storageKey);
    else localStorage.setItem(storageKey, value);
  } catch { /* Storage can be disabled; in-memory preferences still work. */ }
}

/** Synchronize document metadata and browser preference changes; dispose at app unmount. */
export function initLocale() {
  const stop = watch(locale, () => {
    document.documentElement.lang = locale.value;
    document.documentElement.dir = direction.value;
  }, { immediate: true });
  const updateBrowser = () => { if (localePreference.value === 'auto') locale.value = detectBrowserLocale(); };
  const updateStorage = (event: StorageEvent) => {
    if (event.key !== storageKey && event.key !== null) return;
    localePreference.value = readPreference();
    locale.value = localePreference.value === 'auto' ? detectBrowserLocale() : localePreference.value;
  };
  window.addEventListener('languagechange', updateBrowser);
  window.addEventListener('storage', updateStorage);
  return () => {
    stop();
    window.removeEventListener('languagechange', updateBrowser);
    window.removeEventListener('storage', updateStorage);
  };
}
