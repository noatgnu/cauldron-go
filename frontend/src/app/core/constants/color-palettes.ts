export interface ColorPalette {
  name: string;
  displayName: string;
  description: string;
  colors: string[];
}

export const COLOR_PALETTES: Record<string, ColorPalette> = {
  default: {
    name: 'default',
    displayName: 'Default',
    description: 'Standard color palette for general use',
    colors: [
      '#1f77b4',
      '#ff7f0e',
      '#2ca02c',
      '#d62728',
      '#9467bd',
      '#8c564b',
      '#e377c2',
      '#7f7f7f',
      '#bcbd22',
      '#17becf'
    ]
  },
  deuteranopia: {
    name: 'deuteranopia',
    displayName: 'Deuteranopia Safe',
    description: 'Optimized for red-green color blindness (most common type)',
    colors: [
      '#0173B2',
      '#DE8F05',
      '#029E73',
      '#CC78BC',
      '#CA9161',
      '#949494',
      '#ECE133',
      '#56B4E9',
      '#F0E442',
      '#D55E00'
    ]
  },
  protanopia: {
    name: 'protanopia',
    displayName: 'Protanopia Safe',
    description: 'Optimized for red-green color blindness (protanopia variant)',
    colors: [
      '#0173B2',
      '#FBB13C',
      '#009E73',
      '#D55E00',
      '#CC79A7',
      '#56B4E9',
      '#F0E442',
      '#808080',
      '#B66DFF',
      '#73D055'
    ]
  },
  tritanopia: {
    name: 'tritanopia',
    displayName: 'Tritanopia Safe',
    description: 'Optimized for blue-yellow color blindness (rare)',
    colors: [
      '#E8435A',
      '#F99B0C',
      '#00C2A0',
      '#00C5DC',
      '#FA6F84',
      '#FDB750',
      '#95E7D7',
      '#95E4F5',
      '#F9BFCA',
      '#FDDEA5'
    ]
  },
  monochrome: {
    name: 'monochrome',
    displayName: 'Monochrome',
    description: 'Grayscale palette for complete color blindness',
    colors: [
      '#000000',
      '#1C1C1C',
      '#383838',
      '#555555',
      '#717171',
      '#8D8D8D',
      '#A9A9A9',
      '#C5C5C5',
      '#E2E2E2',
      '#FFFFFF'
    ]
  },
  highContrast: {
    name: 'highContrast',
    displayName: 'High Contrast',
    description: 'Maximum contrast palette meeting WCAG AAA standards',
    colors: [
      '#000000',
      '#0000FF',
      '#FF0000',
      '#00FF00',
      '#FFFF00',
      '#FF00FF',
      '#00FFFF',
      '#800000',
      '#008000',
      '#000080'
    ]
  }
};

export function getColorPalette(paletteName: string): string[] {
  const palette = COLOR_PALETTES[paletteName];
  return palette ? palette.colors : COLOR_PALETTES['default'].colors;
}

export function getPaletteNames(): string[] {
  return Object.keys(COLOR_PALETTES);
}
