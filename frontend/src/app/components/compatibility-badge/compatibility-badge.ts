import { Component, Input, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { CompatibilityReport } from '../../core/services/wails';

@Component({
  selector: 'app-compatibility-badge',
  standalone: true,
  imports: [CommonModule, MatChipsModule, MatIconModule, MatTooltipModule],
  templateUrl: './compatibility-badge.html',
  styleUrl: './compatibility-badge.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class CompatibilityBadge {
  @Input() report: CompatibilityReport | null = null;

  level(): 'ok' | 'warn' | 'error' | 'unknown' {
    const report = this.report;
    if (!report) return 'unknown';
    if (report.allOk) {
      const anyVersionDiffers = report.stages.some(s => s?.status === 'compatible_version_differs');
      return anyVersionDiffers ? 'warn' : 'ok';
    }
    return 'error';
  }

  icon(): string {
    switch (this.level()) {
      case 'ok': return 'check_circle';
      case 'warn': return 'warning';
      case 'error': return 'error';
      default: return 'help';
    }
  }

  label(): string {
    switch (this.level()) {
      case 'ok': return 'Compatible';
      case 'warn': return 'Version differs';
      case 'error': return 'Incompatible';
      default: return 'Unknown';
    }
  }

  tooltip(): string {
    const report = this.report;
    if (!report) return 'Compatibility has not been checked yet';

    const lines: string[] = [];
    for (const stage of report.stages) {
      if (!stage || stage.status === 'compatible') continue;
      if (stage.status === 'missing') {
        lines.push(`Stage ${stage.stageIndex + 1}: plugin "${stage.pluginId}" is not installed`);
      } else if (stage.status === 'compatible_version_differs') {
        lines.push(`Stage ${stage.stageIndex + 1}: recorded against v${stage.recordedVersion}, installed v${stage.installedVersion}`);
      } else if (stage.status === 'incompatible') {
        const missing = [...(stage.missingInputs || []), ...(stage.missingOutputs || [])];
        lines.push(`Stage ${stage.stageIndex + 1}: missing ${missing.join(', ')}`);
      }
    }
    return lines.length ? lines.join('\n') : 'All stages compatible';
  }
}
