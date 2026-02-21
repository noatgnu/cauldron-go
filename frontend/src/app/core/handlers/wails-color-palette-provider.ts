import { Injectable, inject } from '@angular/core';
import { ColorPaletteProvider } from '@cauldron/forms';
import { ThemeService } from '../services/theme.service';

@Injectable({ providedIn: 'root' })
export class WailsColorPaletteProvider implements ColorPaletteProvider {
  private themeService = inject(ThemeService);

  getColorPalette(): string[] {
    return this.themeService.colorPalette();
  }
}
