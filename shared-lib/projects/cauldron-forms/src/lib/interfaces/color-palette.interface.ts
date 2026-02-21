import { InjectionToken } from '@angular/core';

export interface ColorPaletteProvider {
  getColorPalette(): string[];
}

export const COLOR_PALETTE_PROVIDER = new InjectionToken<ColorPaletteProvider>('COLOR_PALETTE_PROVIDER');

export const DEFAULT_COLOR_PALETTE: string[] = [
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
];
