import type { LocaleCode } from '../messages';

/** One row holds every language for a single key; TypeScript enforces completeness. */
export interface LocalizedText {
  zh: string;
  tw: string;
  en: string;
  ja: string;
  ko: string;
  fr: string;
  de: string;
  es: string;
  it: string;
  pt: string;
  ru: string;
  ar: string;
}

export type Dict = Record<string, LocalizedText>;

export const localeKeyMap: Record<keyof LocalizedText, LocaleCode> = {
  zh: 'zh-CN',
  tw: 'zh-TW',
  en: 'en-US',
  ja: 'ja-JP',
  ko: 'ko-KR',
  fr: 'fr-FR',
  de: 'de-DE',
  es: 'es-ES',
  it: 'it-IT',
  pt: 'pt-PT',
  ru: 'ru-RU',
  ar: 'ar-SA'
};

/** Flatten key-major dictionaries into locale-major lookup tables. */
export function buildDict(...modules: Dict[]): Record<LocaleCode, Record<string, string>> {
  const out = Object.fromEntries(
    (Object.keys(localeKeyMap) as (keyof LocalizedText)[]).map(key => [localeKeyMap[key], {} as Record<string, string>])
  ) as Record<LocaleCode, Record<string, string>>;
  for (const module of modules) {
    for (const [key, row] of Object.entries(module)) {
      for (const short of Object.keys(localeKeyMap) as (keyof LocalizedText)[]) {
        out[localeKeyMap[short]][key] = row[short];
      }
    }
  }
  return out;
}
