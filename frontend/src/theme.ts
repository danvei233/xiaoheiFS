import type { ThemeConfig } from "ant-design-vue/es/config-provider/context";

/** Persisted appearance preference; auto follows the operating system. */
export type ThemeMode = "auto" | "light" | "dark";
/** Stable theme identifiers shared with the appearance settings UI. */
export type ThemePreset = "minimal" | "tech" | "executive" | "gold" | "cute" | "glass";
export const THEME_PRESETS: readonly ThemePreset[] = ["minimal", "tech", "executive", "gold", "cute", "glass"];

type Palette = { primary: string; bg: string; surface: string; muted: string; border: string; text: string; secondary: string };

/** Structural identity: each preset is a different component language, not a different hue. */
export type ThemeShape = "flat" | "outline" | "editorial" | "cut" | "soft" | "liquid";

type ThemeDefinition = {
  radius: number;
  controlRadius: number;
  controlHeight: number;
  density: number;
  shape: ThemeShape;
  /** Component-level structure, consumed by theme.css as CSS variables. */
  buttonRadius: string;
  buttonWeight: number;
  buttonLetter: string;
  buttonTransform: string;
  buttonClip: string;
  cardRadius: string;
  cardBorderWidth: string;
  cardBorderStyle: string;
  cardShadow: string;
  cardHoverShadow: string;
  cardHoverTransform: string;
  inputRadius: string;
  inputBorderWidth: string;
  rowPadding: string;
  headTransform: string;
  navActive: string;
  surfaceFilter: string;
  canvasPattern: string;
  light: Palette;
  dark: Palette;
};

const structure: Record<ThemePreset, Omit<ThemeDefinition, "light" | "dark">> = {
  // No ornament: hairline borders, square-ish controls, flat surfaces, tight rhythm.
  minimal: {
    radius: 4, controlRadius: 4, controlHeight: 34, density: 0.92, shape: "flat",
    buttonRadius: "4px", buttonWeight: 500, buttonLetter: "0", buttonTransform: "none", buttonClip: "none",
    cardRadius: "6px", cardBorderWidth: "1px", cardBorderStyle: "solid", cardShadow: "none",
    cardHoverShadow: "none", cardHoverTransform: "none",
    inputRadius: "4px", inputBorderWidth: "1px", rowPadding: "10px", headTransform: "none",
    navActive: "bar", surfaceFilter: "none", canvasPattern: "none"
  },
  // Technical drawing: hard square edges, 2px rules, corner ticks, monospace labels, scanline canvas.
  tech: {
    radius: 0, controlRadius: 0, controlHeight: 32, density: 0.78, shape: "outline",
    buttonRadius: "0", buttonWeight: 700, buttonLetter: ".06em", buttonTransform: "translateY(-1px)", buttonClip: "none",
    cardRadius: "0", cardBorderWidth: "1px", cardBorderStyle: "solid", cardShadow: "0 0 0 1px rgb(8 98 212 / 10%)",
    cardHoverShadow: "0 0 0 1px var(--primary), 0 8px 24px rgb(8 98 212 / 14%)", cardHoverTransform: "none",
    inputRadius: "0", inputBorderWidth: "1px", rowPadding: "7px", headTransform: "uppercase",
    navActive: "block", surfaceFilter: "none", canvasPattern: "grid"
  },
  // FrontPage "tech" language: navy chrome, electronic-blue accent, hairlines, zero elevation, pill tabs.
  executive: {
    radius: 8, controlRadius: 6, controlHeight: 38, density: 0.9, shape: "editorial",
    buttonRadius: "6px", buttonWeight: 550, buttonLetter: "0", buttonTransform: "none", buttonClip: "none",
    cardRadius: "8px", cardBorderWidth: "1px", cardBorderStyle: "solid", cardShadow: "none",
    cardHoverShadow: "0 3px 14px rgb(13 40 66 / 10%)", cardHoverTransform: "none",
    inputRadius: "6px", inputBorderWidth: "1px", rowPadding: "11px", headTransform: "uppercase",
    navActive: "pill", surfaceFilter: "none", canvasPattern: "none"
  },
  // Obsidian & metal: near-black surfaces, brushed-gold accents, hairline gold rules.
  gold: {
    radius: 10, controlRadius: 8, controlHeight: 40, density: 0.95, shape: "cut",
    buttonRadius: "8px", buttonWeight: 650, buttonLetter: ".01em", buttonTransform: "translateY(-1px)", buttonClip: "none",
    cardRadius: "10px", cardBorderWidth: "1px", cardBorderStyle: "solid", cardShadow: "0 6px 18px rgb(23 32 51 / 8%)",
    cardHoverShadow: "0 12px 28px rgb(23 32 51 / 12%)", cardHoverTransform: "translateY(-2px)",
    inputRadius: "8px", inputBorderWidth: "1px", rowPadding: "11px", headTransform: "uppercase",
    navActive: "bar", surfaceFilter: "none", canvasPattern: "none"
  },
  // Soft toy: oversized radii, pill controls, candy shadows, springy hover, dotted canvas.
  cute: {
    radius: 24, controlRadius: 999, controlHeight: 44, density: 1.12, shape: "soft",
    buttonRadius: "999px", buttonWeight: 700, buttonLetter: "0", buttonTransform: "translateY(-3px) scale(1.02)", buttonClip: "none",
    cardRadius: "24px", cardBorderWidth: "1px", cardBorderStyle: "solid", cardShadow: "0 10px 24px rgb(24 120 99 / 12%)",
    cardHoverShadow: "0 18px 38px rgb(24 120 99 / 20%)", cardHoverTransform: "translateY(-4px) rotate(-.4deg)",
    inputRadius: "999px", inputBorderWidth: "1px", rowPadding: "13px", headTransform: "none",
    navActive: "pill", surfaceFilter: "none", canvasPattern: "dots"
  },
  // Liquid glass: translucent layered panes, refraction edge, inner highlight, blurred canvas.
  glass: {
    radius: 22, controlRadius: 14, controlHeight: 42, density: 1.02, shape: "liquid",
    buttonRadius: "14px", buttonWeight: 640, buttonLetter: ".01em", buttonTransform: "translateY(-1px)", buttonClip: "none",
    cardRadius: "26px", cardBorderWidth: "1px", cardBorderStyle: "solid", cardShadow: "0 18px 48px rgb(23 38 84 / 18%)",
    cardHoverShadow: "0 26px 64px rgb(23 38 84 / 26%)", cardHoverTransform: "translateY(-2px)",
    inputRadius: "14px", inputBorderWidth: "1px", rowPadding: "12px", headTransform: "none",
    navActive: "glass", surfaceFilter: "blur(22px) saturate(185%) brightness(1.04)", canvasPattern: "aurora"
  }
};

const palettes: Record<ThemePreset, { light: Palette; dark: Palette }> = {
  minimal: {
    light: { primary: "#1677ff", bg: "#f5f7fa", surface: "#ffffff", muted: "#f0f2f5", border: "#d9dfe8", text: "#1f2937", secondary: "#667085" },
    dark: { primary: "#69a9ff", bg: "#141a22", surface: "#1d2632", muted: "#263240", border: "#394757", text: "#edf2f7", secondary: "#a4b1c1" } },
  tech: {
    light: { primary: "#0862d4", bg: "#edf5ff", surface: "#ffffff", muted: "#e5f0ff", border: "#b5cde9", text: "#122b48", secondary: "#526b87" },
    dark: { primary: "#54bcff", bg: "#081423", surface: "#102338", muted: "#163049", border: "#285071", text: "#e7f4ff", secondary: "#9db9d0" } },
  executive: {
    light: { primary: "#147caf", bg: "#f0f5fa", surface: "#ffffff", muted: "#eaf1f7", border: "#d8e2ec", text: "#10243a", secondary: "#4c647b" },
    dark: { primary: "#42c5ec", bg: "#081422", surface: "#0f2031", muted: "#11263a", border: "#21405a", text: "#edf6fc", secondary: "#8eabc5" } },
  gold: {
    light: { primary: "#e8a020", bg: "#f7f8fa", surface: "#ffffff", muted: "#eef1f6", border: "#e2e8f0", text: "#172033", secondary: "#5b6a82" },
    dark: { primary: "#ffb020", bg: "#0b1220", surface: "#121a29", muted: "#182335", border: "#24314a", text: "#e8eef7", secondary: "#93a3b8" } },
  cute: {
    light: { primary: "#187863", bg: "#edf9f3", surface: "#ffffff", muted: "#ddf4e9", border: "#add8c5", text: "#20483c", secondary: "#52756a" },
    dark: { primary: "#7bdfbc", bg: "#10231e", surface: "#1a352c", muted: "#24463b", border: "#3f6b5a", text: "#e5f7ed", secondary: "#a1c9b8" } },
  glass: {
    light: { primary: "#435ac7", bg: "#eaf0fb", surface: "#f9fbff", muted: "#e3eafa", border: "#b8c7e6", text: "#243450", secondary: "#586b8b" },
    dark: { primary: "#a5b7ff", bg: "#10182d", surface: "#202d47", muted: "#2a3855", border: "#516384", text: "#eef3ff", secondary: "#b2c0da" } }
};

const definitions = Object.fromEntries(
  THEME_PRESETS.map((preset) => [preset, { ...structure[preset], ...palettes[preset] }])
) as Record<ThemePreset, ThemeDefinition>;

/** Validate storage and UI values before using them as theme identifiers. */
export const isThemePreset = (value: unknown): value is ThemePreset => THEME_PRESETS.includes(value as ThemePreset);
export const isThemeMode = (value: unknown): value is ThemeMode => value === "auto" || value === "light" || value === "dark";

function mixHex(color: string, target: string, amount: number): string {
  const channels = (value: string) => [1, 3, 5].map(offset => Number.parseInt(value.slice(offset, offset + 2), 16));
  const from = channels(color);
  const to = channels(target);
  return `#${from.map((channel, index) => Math.round(channel + (to[index] - channel) * amount).toString(16).padStart(2, "0")).join("")}`;
}

/** Optional appearance switches exposed by the theme panel. */
export type ThemeAppearanceOptions = { rounded?: boolean; transparent?: boolean };

function withAlpha(hex: string, alpha: number): string {
  const value = hex.replace("#", "");
  const full = value.length === 3 ? value.split("").map((c) => c + c).join("") : value;
  const [r, g, b] = [0, 2, 4].map((i) => Number.parseInt(full.slice(i, i + 2), 16));
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

/** Ant Design tokens and application CSS variables are derived from the same palette. */
export function getThemeAppearance(preset: ThemePreset, dark: boolean, options: ThemeAppearanceOptions = {}) {
  const rounded = options.rounded ?? true;
  const transparent = options.transparent ?? false;
  const definition = definitions[preset];
  const p = definition[dark ? "dark" : "light"];
  const accentSoft = `${p.primary}18`;
  const shadow = definition.cardShadow;
  const primaryHover = mixHex(p.primary, dark ? "#ffffff" : "#000000", 0.12);
  const primaryActive = mixHex(p.primary, dark ? "#ffffff" : "#000000", 0.2);
  const darkChrome = preset === "executive";
  const chrome = preset === "executive" ? (dark ? "#06111d" : "#10243a") : p.surface;
  const chromeText = preset === "executive" ? (dark ? "#eaf7ff" : "#eff8ff") : p.text;
  const chromeMuted = preset === "executive" ? (dark ? "#8eacbf" : "#a8c1d4") : p.secondary;
  const sidebar = preset === "executive" ? (dark ? "#0a1928" : "#f7fafd") : chrome;
  const sidebarText = preset === "executive" && !dark ? p.text : chromeText;
  const sidebarMuted = preset === "executive" && !dark ? p.secondary : chromeMuted;
  const glassSurface = preset === "glass" ? `${p.surface}dc` : p.surface;
  const headerSurface = preset === "tech"
    ? dark ? "linear-gradient(110deg, #12385b, #0c2842)" : "linear-gradient(110deg, #dcecff, #f6faff)"
    : preset === "glass" ? glassSurface
      : darkChrome ? chrome : `${p.surface}f5`;

  // Rounded corners and translucent surfaces are independent switches shared by every preset.
  const controlRadius = rounded ? definition.controlRadius : 2;
  const cardRadius = rounded ? definition.cardRadius : "2px";
  const radiusSmall = rounded ? `${Math.max(2, definition.controlRadius - 2)}px` : "2px";
  const radiusLarge = rounded ? `${definition.radius}px` : "2px";
  const surface = transparent ? withAlpha(p.surface, dark ? 0.58 : 0.62) : p.surface;
  const mutedSurface = transparent ? withAlpha(p.muted, dark ? 0.5 : 0.55) : p.muted;
  const controlSurface = transparent ? withAlpha(p.surface, dark ? 0.46 : 0.5) : p.surface;
  const backdrop = transparent ? "blur(18px) saturate(155%)" : definition.surfaceFilter;
  const canvas = transparent
    ? [
      `radial-gradient(ellipse 62% 52% at 12% 0%, ${p.primary}${dark ? "3d" : "26"}, transparent 62%)`,
      `radial-gradient(ellipse 55% 48% at 92% 92%, ${p.primary}${dark ? "2b" : "1a"}, transparent 58%)`,
      p.bg
    ].join(", ")
    : preset === "glass" ? [
      `radial-gradient(ellipse 78% 58% at 16% 0%, ${p.primary}${dark ? "4d" : "38"}, transparent 62%)`,
      `radial-gradient(ellipse 62% 52% at 92% 100%, ${p.primary}${dark ? "33" : "26"}, transparent 60%)`,
      p.bg
    ].join(", ")
      : preset === "tech"
        ? `linear-gradient(${p.primary}14 1px, transparent 1px), linear-gradient(90deg, ${p.primary}14 1px, transparent 1px), ${p.bg}`
        : preset === "cute"
          ? `radial-gradient(${p.primary}22 1.5px, transparent 1.6px), ${p.bg}`
          : p.bg;

  const token: ThemeConfig["token"] = {
    colorPrimary: p.primary, colorInfo: p.primary, colorBgLayout: p.bg,
    colorBgContainer: p.surface, colorBgElevated: p.surface, colorBorder: p.border,
    colorBorderSecondary: p.border, colorText: p.text, colorTextSecondary: p.secondary,
    borderRadius: controlRadius, borderRadiusLG: rounded ? definition.radius : 2,
    boxShadow: shadow, boxShadowSecondary: dark ? "0 10px 30px #00000055" : "0 10px 30px #16253d18"
  };
  const variables: Record<string, string> = {
    "--bg-primary": p.bg, "--bg-secondary": mutedSurface, "--bg-tertiary": mutedSurface,
    "--card": surface, "--card-hover": mutedSurface, "--border": p.border,
    "--border-light": p.border, "--border-dark": mixHex(p.border, dark ? "#ffffff" : "#000000", 0.12),
    "--text-primary": p.text, "--text-secondary": p.secondary, "--text-tertiary": p.secondary,
    "--text-inverse": dark ? "#101820" : "#ffffff",
    "--primary": p.primary, "--primary-light": mixHex(p.primary, dark ? "#ffffff" : "#000000", 0.18),
    "--primary-dark": primaryActive, "--primary-hover": primaryHover,
    "--primary-gradient": p.primary, "--primary-gradient-subtle": accentSoft,
    "--primary-gradient-glow": `radial-gradient(ellipse at center, ${p.primary}24, transparent 70%)`,
    "--primary-bg": accentSoft,
    "--radius-sm": radiusSmall, "--radius-md": `${controlRadius}px`,
    "--radius-lg": radiusLarge, "--radius-xl": rounded ? `${definition.radius + 4}px` : "2px",
    "--shadow-sm": shadow, "--shadow-md": shadow, "--shadow-lg": token.boxShadowSecondary!,
    "--theme-shadow": shadow, "--theme-shadow-popover": token.boxShadowSecondary!,
    "--theme-accent-soft": accentSoft, "--theme-focus-ring": `${p.primary}38`,
    "--theme-primary-hover": primaryHover, "--theme-primary-active": primaryActive,
    "--theme-primary-foreground": preset === "executive" ? (dark ? "#092131" : "#ffffff")
      : preset === "gold" ? (dark ? "#1d1405" : "#241a06") : dark ? "#142033" : "#ffffff",
    "--theme-density": String(definition.density),
    "--theme-control-height": `${definition.controlHeight}px`,
    "--theme-control-radius": rounded ? definition.inputRadius : "2px",
    "--theme-card-radius": cardRadius,
    "--theme-border-width": definition.cardBorderWidth,

    // Structural identity: shape, stroke, elevation, rhythm and material per preset.
    "--theme-shape": definition.shape,
    "--theme-btn-radius": rounded ? definition.buttonRadius : "2px",
    "--theme-btn-weight": String(definition.buttonWeight),
    "--theme-btn-letter": definition.buttonLetter,
    "--theme-btn-transform": definition.buttonTransform,
    "--theme-btn-clip": definition.buttonClip === "cut" && rounded
      ? "polygon(12px 0, 100% 0, 100% calc(100% - 12px), calc(100% - 12px) 100%, 0 100%, 0 12px)"
      : "none",
    "--theme-card-border-width": definition.cardBorderWidth,
    "--theme-card-border-style": definition.cardBorderStyle,
    "--theme-card-shadow": definition.cardShadow,
    "--theme-card-hover-shadow": definition.cardHoverShadow,
    "--theme-card-hover-transform": definition.cardHoverTransform,
    "--theme-input-radius": rounded ? definition.inputRadius : "2px",
    "--theme-input-border-width": definition.inputBorderWidth,
    "--theme-row-padding": definition.rowPadding,
    "--theme-head-transform": definition.headTransform,
    "--theme-nav-active": definition.navActive,
    "--theme-surface-filter": backdrop,
    "--theme-canvas-pattern": definition.canvasPattern,
    "--theme-rounded": rounded ? "1" : "0",
    "--theme-transparent": transparent ? "1" : "0",
    "--theme-glass-edge": preset === "glass" || transparent ? (dark ? "#ffffff4d" : "#ffffffcc") : "transparent",
    "--theme-glass-inner": preset === "glass" || transparent ? (dark ? "#ffffff1f" : "#ffffffb3") : "transparent",
    "--theme-glass-tint": preset === "glass"
      ? (dark
        ? "linear-gradient(135deg, rgba(88,106,168,.34), rgba(24,32,58,.46))"
        : "linear-gradient(135deg, rgba(255,255,255,.62), rgba(226,236,255,.30))")
      : "transparent",
    "--theme-control-bg": transparent ? controlSurface : preset === "glass" ? `${p.surface}b3` : p.surface,
    "--theme-control-hover": transparent ? withAlpha(p.surface, dark ? 0.62 : 0.72) : preset === "cute" ? p.muted : p.surface,
    "--theme-table-head-bg": transparent ? mutedSurface : preset === "tech" ? p.muted : preset === "gold" ? `${p.primary}12` : p.muted,
    "--theme-table-row-hover": preset === "cute" ? `${p.primary}0d` : `${p.primary}0a`,
    "--theme-selection-color": p.primary,
    "--theme-selection-soft": accentSoft,
    "--theme-header-surface": transparent ? withAlpha(dark && !darkChrome ? p.surface : chrome, 0.72) : headerSurface,
    "--theme-header-text": chromeText, "--theme-header-muted": chromeMuted,
    "--theme-header-border": darkChrome ? (preset === "executive" ? (dark ? "#1b3a54" : "#1d3c58") : "#34465e") : p.border,
    "--theme-sidebar": transparent ? withAlpha(chrome, 0.66) : sidebar,
    "--theme-sidebar-text": sidebarText,
    "--theme-sidebar-muted": sidebarMuted,
    "--theme-sidebar-accent": preset === "executive" ? (dark ? "#64d4f2" : "#116c9a")
      : preset === "gold" ? (dark ? "#ffb020" : "#b06f0a")
        : darkChrome && (preset !== "executive" || dark) ? "#a8c9f0" : p.primary,
    "--theme-sidebar-active": preset === "executive" ? accentSoft
      : darkChrome && (preset !== "executive" || dark) ? "#ffffff14" : accentSoft,
    "--theme-surface": transparent ? surface : preset === "glass" ? `${p.surface}a8` : p.surface,
    "--theme-backdrop": backdrop,
    "--theme-glass-highlight": preset === "glass" || transparent ? (dark ? "#ffffff1c" : "#ffffffb8") : "transparent",
    "--theme-canvas": canvas
  };
  return { token, variables };
}
