import { Component, input, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-cauldron-loader',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './cauldron-loader.html',
  styleUrl: './cauldron-loader.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class CauldronLoaderComponent {
  message = input<string>('Brewing...');
  progress = input<number>(50);

  liquidY = computed(() => {
    const prog = Math.max(0, Math.min(100, this.progress()));
    const minY = 15;
    const maxY = 108;
    return maxY - ((maxY - minY) * prog / 100);
  });

  liquidHeight = computed(() => {
    const maxY = 108;
    return maxY - this.liquidY();
  });

  bubbleStartY = computed(() => {
    return this.liquidY() + this.liquidHeight() - 5;
  });

  bubbleEndY = computed(() => {
    return this.liquidY();
  });
}