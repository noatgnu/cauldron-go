import { Component, input, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';

interface Bubble {
  left: number;
  duration: string;
  delay: string;
}

export type ProgressBarMode = 'determinate' | 'indeterminate';

@Component({
  selector: 'app-boiling-progress-bar',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './boiling-progress-bar.html',
  styleUrl: './boiling-progress-bar.scss'
})
export class BoilingProgressBarComponent implements OnInit {
  value = input<number>(0);
  displayValue = input<boolean>(false);
  mode = input<ProgressBarMode>('determinate');
  
  bubbles: Bubble[] = [];

  ngOnInit() {
    // Generate static random bubbles to avoid hydration mismatch and performance hits
    // They are positioned relative to the filled width (0-100%)
    this.bubbles = Array.from({ length: 12 }, () => ({
      left: Math.random() * 100,
      duration: `${1.5 + Math.random() * 2}s`, // 1.5s to 3.5s
      delay: `${Math.random() * 2}s`
    }));
  }
}
