export type ColorTheme = 'azure' | 'teal' | 'purple' | 'rose' | 'orange' | 'green' | 'cyber';

export interface ThemeDefinition {
  id: ColorTheme;
  displayName: string;
  previewLight: string;
  previewDark: string;
}

export const COLOR_THEMES: Record<ColorTheme, ThemeDefinition> = {
  azure: {
    id: 'azure',
    displayName: 'Azure',
    previewLight: '#1976d2',
    previewDark: '#64b5f6'
  },
  teal: {
    id: 'teal',
    displayName: 'Teal',
    previewLight: '#009688',
    previewDark: '#4db6ac'
  },
  purple: {
    id: 'purple',
    displayName: 'Purple',
    previewLight: '#7b1fa2',
    previewDark: '#ba68c8'
  },
  rose: {
    id: 'rose',
    displayName: 'Rose',
    previewLight: '#c2185b',
    previewDark: '#f48fb1'
  },
  orange: {
    id: 'orange',
    displayName: 'Orange',
    previewLight: '#e65100',
    previewDark: '#ffb74d'
  },
  green: {
    id: 'green',
    displayName: 'Green',
    previewLight: '#2e7d32',
    previewDark: '#81c784'
  },
  cyber: {
    id: 'cyber',
    displayName: 'Cyber',
    previewLight: '#0077be',
    previewDark: '#ffb400'
  }
};

export function getColorTheme(themeId: ColorTheme): ThemeDefinition {
  return COLOR_THEMES[themeId] || COLOR_THEMES.azure;
}

export function getColorThemeIds(): ColorTheme[] {
  return Object.keys(COLOR_THEMES) as ColorTheme[];
}
