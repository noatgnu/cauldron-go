import { Component, OnInit, signal, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { DynamicFormComponent } from '../../components/dynamic-form/dynamic-form';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { models } from '../../../wailsjs/go/models';
import { EnvironmentIndicator } from '../../components/environment-indicator/environment-indicator';

@Component({
  selector: 'app-plugin-execute',
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
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

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private pluginService: PluginV2Service,
    private snackBar: MatSnackBar
  ) {}

  async ngOnInit() {
    this.route.paramMap.subscribe(async params => {
      const pluginId = params.get('id');
      if (!pluginId) {
        this.error.set('Plugin ID not provided');
        this.loading.set(false);
        return;
      }

      this.createdJobId.set(null);
      await this.loadPlugin(pluginId);
    });
  }

  async loadPlugin(id: string) {
    try {
      this.loading.set(true);
      this.error.set('');
      const plugin = await this.pluginService.getPlugin(id);
      this.plugin.set(plugin);
    } catch (err) {
      this.error.set(`Failed to load plugin: ${err}`);
    } finally {
      this.loading.set(false);
    }
  }

  async onExecute(parameters: Record<string, any>) {
    const plugin = this.plugin();
    if (!plugin) return;

    try {
      this.executing.set(true);
      const jobId = await this.pluginService.executePlugin(plugin.definition.plugin.id, parameters);
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

  getRuntimeIcon(runtime: string): { python: boolean; r: boolean; pythonWithR: boolean } {
    return {
      python: runtime === 'python',
      r: runtime === 'r',
      pythonWithR: runtime === 'pythonWithR'
    };
  }
}
