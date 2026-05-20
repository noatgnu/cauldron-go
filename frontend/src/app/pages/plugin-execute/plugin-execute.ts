import { Component, OnInit, OnDestroy, signal, ViewChild, ChangeDetectionStrategy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';
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
import * as models from '../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';
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
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class PluginExecute implements OnInit, OnDestroy {
  @ViewChild(DynamicFormComponent) dynamicForm?: DynamicFormComponent;
  private paramMapSubscription?: Subscription;

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

  ngOnDestroy() {
    this.paramMapSubscription?.unsubscribe();
  }

  ngOnInit() {
    this.paramMapSubscription = this.route.paramMap.subscribe(async params => {
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
  }

  async loadPlugin(id: number) {
    try {
      this.loading.set(true);
      this.error.set('');
      const plugin = await this.pluginService.getPlugin(id);
      this.plugin.set(plugin);
      await this.loadPluginBinding(plugin.definition.plugin.id);

      const cloneJobId = this.route.snapshot.queryParamMap.get('cloneJobId');
      if (cloneJobId) {
        await this.wails.logToFile(`[PluginExecute] Cloning job: ${cloneJobId}`);
        setTimeout(async () => {
          await this.loadJobParameters(cloneJobId);
        }, 500);
      }
    } catch (err) {
      this.error.set(`Failed to load plugin: ${err}`);
    } finally {
      this.loading.set(false);
    }
  }

  async loadJobParameters(jobId: string) {
    try {
      await this.wails.logToFile(`[PluginExecute] Fetching job ${jobId} for cloning`);
      const job = await this.wails.getJob(jobId);
      if (job && job.parameters) {
        await this.wails.logToFile(`[PluginExecute] Loading parameters from job ${jobId}`);
        await this.dynamicForm?.loadFromJobParameters(job.parameters);
      } else {
        await this.wails.logToFile(`[PluginExecute] Job ${jobId} has no parameters`);
        this.notification.showError('Failed to load job parameters');
      }
    } catch (err) {
      await this.wails.logToFile(`[PluginExecute] Error loading job parameters: ${err}`);
      this.notification.showError(`Failed to load job parameters: ${err}`);
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

      const envs = plugin.definition.runtime.environments || [];
      const pluginStringId = plugin.definition.plugin.id;

      if (envs.includes('python') && !this.pythonBound()) {
        const bound = await this.ensurePythonBinding(pluginStringId);
        if (!bound) {
          this.notification.showError('No Python environment configured. Please set an active Python environment in Settings.');
          return;
        }
      }

      if (envs.includes('r') && !this.rBound()) {
        const bound = await this.ensureRBinding(pluginStringId);
        if (!bound) {
          this.notification.showError('No R environment configured. Please set an active R environment in Settings.');
          return;
        }
      }

      const jobId = await this.pluginService.executePlugin(plugin.id, parameters);
      this.createdJobId.set(jobId);

      this.notification.showSuccess('Job created successfully!');
    } catch (err) {
      this.notification.showError(`Failed to execute plugin: ${err}`);
    } finally {
      this.executing.set(false);
    }
  }

  private async ensurePythonBinding(pluginStringId: string): Promise<boolean> {
    try {
      const pythonEnv = await this.wails.getActivePythonEnvironment();
      if (!pythonEnv?.path) return false;
      await this.wails.bindPluginToEnvironment(pluginStringId, 'python', 0, pythonEnv.path);
      this.pythonBound.set(true);
      this.hasCustomBinding.set(true);
      return true;
    } catch {
      return false;
    }
  }

  private async ensureRBinding(pluginStringId: string): Promise<boolean> {
    try {
      const rEnv = await this.wails.getActiveREnvironment();
      if (!rEnv?.path) return false;
      await this.wails.bindPluginToEnvironment(pluginStringId, 'r', 0, rEnv.path);
      this.rBound.set(true);
      this.hasCustomBinding.set(true);
      return true;
    } catch {
      return false;
    }
  }

  reset() {
    this.dynamicForm?.reset();
    this.createdJobId.set(null);
  }

  loadExample() {
    this.dynamicForm?.loadExample();
  }

  viewJob(): void {
    const jobId = this.createdJobId();
    if (jobId) {
      this.router.navigate(['/job', jobId]);
    }
  }

  getRuntimeIcon(runtime: string): { python: boolean; r: boolean; direct: boolean } {
    return {
      python: runtime === 'python',
      r: runtime === 'r',
      direct: runtime === 'direct'
    };
  }

  getRuntimeIndicator(plugin: models.PluginV2): { python: boolean; r: boolean; direct: boolean } {
    const envs = this.getPluginEnvironments(plugin);
    return {
      python: envs.includes('python'),
      r: envs.includes('r'),
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
