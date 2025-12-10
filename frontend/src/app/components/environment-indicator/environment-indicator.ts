import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';

@Component({
  selector: 'app-environment-indicator',
  standalone: true,
  imports: [CommonModule, MatChipsModule, MatIconModule, MatTooltipModule],
  templateUrl: './environment-indicator.html',
  styleUrl: './environment-indicator.scss'
})
export class EnvironmentIndicator {
  @Input() python: boolean = false;
  @Input() r: boolean = false;
  @Input() pythonWithR: boolean = false;
  @Input() direct: boolean = false;
}
