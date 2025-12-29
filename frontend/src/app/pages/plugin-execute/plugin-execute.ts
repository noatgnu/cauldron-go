import { Component, OnInit, signal, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatTabsModule } from '@angular/material/tabs';
import { DynamicFormComponent } from '../../components/dynamic-form/dynamic-form';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { NotificationService } from '../../core/services/notification.service';
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
    MatTooltipModule,
    MatTabsModule,
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
    private notification: NotificationService,
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

      this.notification.showSuccess('Job created successfully!');
    } catch (err) {
      this.notification.showError(`Failed to execute plugin: ${err}`);
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

  getRuntimeIndicator(plugin: models.PluginV2): { python: boolean; r: boolean; pythonWithR: boolean; direct: boolean } {
    const envs = this.getPluginEnvironments(plugin);
    const hasPython = envs.includes('python');
    const hasR = envs.includes('r');
    return {
      python: hasPython && !hasR,
      r: hasR && !hasPython,
      pythonWithR: hasPython && hasR,
      direct: envs.includes('direct')
    };
  }

  private getPluginEnvironments(plugin: models.PluginV2): string[] {
    return plugin.definition.runtime.environments || [];
  }

  getRuntimeLabel(plugin: models.PluginV2): string {
    const envs = this.getPluginEnvironments(plugin);
    if (envs.length === 0) {
      return 'Direct';
    }
    return envs.map(e => e.charAt(0).toUpperCase() + e.slice(1)).join(' + ');
  }
}
