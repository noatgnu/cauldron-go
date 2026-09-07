import { Component, EventEmitter, Input, Output, ChangeDetectionStrategy } from '@angular/core';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatIconModule } from '@angular/material/icon';
import { GelLaneROI } from '../../../core/services/wails';

@Component({
  selector: 'app-gel-lane-map',
  standalone: true,
  imports: [MatTooltipModule, MatIconModule],
  templateUrl: './gel-lane-map.html',
  styleUrl: './gel-lane-map.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class GelLaneMap {
  @Input() lanes: GelLaneROI[] = [];
  @Input() imageWidth = 0;
  @Input() selectedLaneId: string | null = null;
  @Output() laneSelected = new EventEmitter<string>();

  laneLeft(lane: GelLaneROI): number {
    return this.imageWidth > 0 ? (lane.x / this.imageWidth) * 100 : 0;
  }

  laneWidthPercent(lane: GelLaneROI): number {
    return this.imageWidth > 0 ? Math.max((lane.width / this.imageWidth) * 100, 0.6) : 0;
  }

  select(laneId: string): void {
    this.laneSelected.emit(laneId);
  }
}
