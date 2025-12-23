import { Component, OnInit, signal, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { DynamicFormComponent } from '../../components/dynamic-form/dynamic-form';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { models } from '../../../wailsjs/go/models';
import { EnvironmentIndicator } from '../../components/environment-indicator/environment-indicator';
import { Wails } from '../../core/services/wails';

@Component({
  selector: 'app-plugin-execute',
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
    MatTooltipModule,
    DynamicFormComponent,
    EnvironmentIndicator
  ],
  templateUrl: './plugin-execute.html',
  styleUrl: './plugin-execute.scss',
})
export class PluginExecute implements OnInit {
  @ViewChild(DynamicFormComponent) dynamicForm?: DynamicFormComponent;

  plugin = signal<models.PluginV2 | null>(null);
  loading = signal(true);
  executing = signal(false);
  error = signal('');
  createdJobId = signal<string | null>(null);
  hasCustomBinding = signal(false);
  bindingTooltip = signal('');
  pythonBound = signal(false);
  rBound = signal(false);

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private pluginService: PluginV2Service,
    private snackBar: MatSnackBar,
    private wails: Wails
  ) {}

  async ngOnInit() {
    this.route.paramMap.subscribe(async params => {
      const pluginIdStr = params.get('id');
      if (!pluginIdStr) {
        this.error.set('Plugin ID not provided');
        this.loading.set(false);
        return;
      }

      const pluginId = parseInt(pluginIdStr, 10);
      if (isNaN(pluginId)) {
        this.error.set('Invalid plugin ID');
        this.loading.set(false);
        return;
      }

      this.createdJobId.set(null);
      await this.loadPlugin(pluginId);
    });

    this.wails.bindingsUpdated$.subscribe(() => {
      const p = this.plugin();
      if (p) {
        this.loadPluginBinding(p.id.toString());
      }
    });
  }

  async loadPlugin(id: number) {
    try {
      this.loading.set(true);
      this.error.set('');
      const plugin = await this.pluginService.getPlugin(id);
      this.plugin.set(plugin);
      await this.loadPluginBinding(id.toString());
    } catch (err) {
      this.error.set(`Failed to load plugin: ${err}`);
    } finally {
      this.loading.set(false);
    }
  }

  async loadPluginBinding(pluginId: string) {
    let hasPython = false;
    let hasR = false;

    try {
      const pythonBinding = await this.wails.getPluginEnvironmentBinding(pluginId, 'python');
      hasPython = !!pythonBinding;
    } catch {}

    try {
      const rBinding = await this.wails.getPluginEnvironmentBinding(pluginId, 'r');
      hasR = !!rBinding;
    } catch {}

    this.pythonBound.set(hasPython);
    this.rBound.set(hasR);

    if (hasPython || hasR) {
      this.hasCustomBinding.set(true);
      const parts: string[] = [];
      if (hasPython) parts.push('Python');
      if (hasR) parts.push('R');
      this.bindingTooltip.set(`Custom environment bound: ${parts.join(' & ')}`);
    } else {
      this.hasCustomBinding.set(false);
      this.bindingTooltip.set('');
    }
  }

  async onExecute(parameters: Record<string, any>) {
    const plugin = this.plugin();
    if (!plugin) return;

    try {
      this.executing.set(true);
      const jobId = await this.pluginService.executePlugin(plugin.id, parameters);
      this.createdJobId.set(jobId);

      this.snackBar.open('Job created successfully!', 'Close', {
        duration: 3000,
        horizontalPosition: 'end',
        verticalPosition: 'top'
      });
    } catch (err) {
      this.snackBar.open(`Failed to execute plugin: ${err}`, 'Close', {
        duration: 5000,
        horizontalPosition: 'end',
        verticalPosition: 'top',
        panelClass: ['error-snackbar']
      });
    } finally {
      this.executing.set(false);
    }
  }

  reset() {
    this.dynamicForm?.reset();
    this.createdJobId.set(null);
  }

  loadExample() {
    this.dynamicForm?.loadExample();
  }

  viewJob() {
    const jobId = this.createdJobId();
    if (jobId) {
      this.router.navigate(['/job', jobId]);
    }
  }

  getRuntimeIcon(runtime: string): { python: boolean; r: boolean; pythonWithR: boolean; direct: boolean } {
    return {
      python: runtime === 'python',
      r: runtime === 'r',
      pythonWithR: runtime === 'pythonWithR',
      direct: runtime === 'direct'
    };
  }
}
