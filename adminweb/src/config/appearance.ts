export const appearanceThemes = {
  minimal: { color: '#4b5563', radius: '0.5' },
  tech: { color: '#087fbd', radius: '0.25' },
  executive: { color: '#52658a', radius: '0.375' },
  gold: { color: '#F2A12E', radius: '0.375' },
  cute: { color: '#17836c', radius: '1' },
  glass: { color: '#7260bf', radius: '0.875' }
} as const

export type AppearanceTheme = keyof typeof appearanceThemes
