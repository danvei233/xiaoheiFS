import { buildDict } from './shared';
import { common } from './common';
import { adminNav } from './adminNav';
import { consoleMessages } from './console';
import { site } from './site';
import { auth } from './auth';

export { buildDict } from './shared';
export type { Dict, LocalizedText } from './shared';

/** Locale-major tables for every key kept outside the legacy array dictionaries. */
export const dictionaries = buildDict(common, adminNav, consoleMessages, site, auth);
