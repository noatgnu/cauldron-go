import { Component, OnInit, OnDestroy, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatChipsModule } from '@angular/material/chips';
import { MatDialog } from '@angular/material/dialog';
import { Wails, Job } from '../../core/services/wails';
import { PcaPlot } from './pca-plot/pca-plot';
import { PhatePlot } from './phate-plot/phate-plot';
import { FuzzyClusteringPlot } from './fuzzy-clustering-plot/fuzzy-clustering-plot';
import { PluginPlot } from './plugin-plot/plugin-plot';
import { SampleAnnotation, SampleAnnotationData } from '../../components/sample-annotation/sample-annotation';
import { AnnotationService } from '../../core/services/annotation.service';

@Component({
  selector: 'app-job-detail',
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatChipsModule,
    PcaPlot,
    PhatePlot,
    FuzzyClusteringPlot,
    PluginPlot
  ],
  templateUrl: './job-detail.html',
  styleUrl: './job-detail.scss'
})
export class JobDetail implements OnInit, OnDestroy {
  protected job = signal<Job | null>(null);
  protected loading = signal(true);
  protected error = signal('');
  protected jobId: string = '';
  protected pluginPlots = signal<Array<{ fileName: string, title: string }>>([]);

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private wails: Wails,
    private dialog: MatDialog,
    private annotationService: AnnotationService
  ) {}

  async ngOnInit() {
    this.route.params.subscribe(async params => {
      this.jobId = params['id'];
      await this.loadJob();
    });

    if (window.runtime) {
      window.runtime.EventsOn('job:update', (data: Job) => {
        if (data.id === this.jobId) {
          this.job.set(data);
        }
      });
    }
  }

  ngOnDestroy() {
    if (window.runtime) {
      window.runtime.EventsOff('job:update');
    }
  }

  async loadJob() {
    this.loading.set(true);
    try {
      const jobData = await this.wails.getJob(this.jobId);
      this.job.set(jobData);

      if (jobData.status === 'completed') {
        await this.detectPluginPlots();
      }
    } catch (err: any) {
      this.error.set(err.message || 'Failed to load job');
    } finally {
      this.loading.set(false);
    }
  }

  goBack() {
    this.router.navigate(['/jobs']);
  }

  getStatusColor(status: string): string {
    switch (status) {
      case 'completed': return 'primary';
      case 'failed': return 'warn';
      case 'in_progress': return 'accent';
      default: return '';
    }
  }

  getStatusIcon(status: string): string {
    switch (status) {
      case 'completed': return 'check_circle';
      case 'in_progress': return 'sync';
      case 'failed': return 'error';
      default: return 'schedule';
    }
  }

  async openAnnotationDialog() {
    const annotations = await this.annotationService.loadAnnotationsForJob(this.jobId);

    let samples: string[] = [];
    if (annotations.length === 0) {
      samples = await this.extractSampleNames();
    }

    const dialogData: SampleAnnotationData = {
      mode: annotations.length > 0 ? 'edit' : 'create',
      samples: samples.length > 0 ? samples : undefined,
      annotation: undefined
    };

    const dialogRef = this.dialog.open(SampleAnnotation, {
      width: '800px',
      data: dialogData
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        await this.annotationService.saveAnnotationsForJob(this.jobId, result);
        await this.wails.logToFile(`[Job Detail] Saved annotations for job ${this.jobId}`);
        window.location.reload();
      }
    });
  }

  async extractSampleNames(): Promise<string[]> {
    try {
      const jobType = this.job()?.type;
      let dataFile = '';

      if (jobType === 'pca') {
        dataFile = 'pca_output.txt';
      } else if (jobType === 'phate') {
        dataFile = 'phate_output.txt';
      } else if (jobType === 'fuzzy-clustering') {
        dataFile = 'fuzzy_clustering_output.txt';
      }

      if (dataFile) {
        const content = await this.wails.readJobOutputFile(this.jobId, dataFile);
        const lines = content.trim().split('\n');
        if (lines.length > 1) {
          const headers = lines[0].split('\t');
          const sampleIndex = headers.indexOf('sample');

          if (sampleIndex !== -1) {
            const samples: string[] = [];
            for (let i = 1; i < lines.length; i++) {
              const values = lines[i].split('\t');
              if (values[sampleIndex]) {
                const samplePath = values[sampleIndex];
                const sampleName = samplePath.split(/[/\\]/).pop() || samplePath;
                if (!samples.includes(sampleName)) {
                  samples.push(sampleName);
                }
              }
            }
            return samples;
          }
        }
      }
    } catch (err) {
      await this.wails.logToFile(`[Job Detail] Failed to extract sample names: ${err}`);
    }
    return [];
  }

  supportsAnnotations(): boolean {
    const jobType = this.job()?.type;
    return jobType === 'pca' || jobType === 'phate' || jobType === 'fuzzy-clustering';
  }

  async detectPluginPlots() {
    try {
      const plotFiles: Array<{ fileName: string, title: string }> = [];

      for (let i = 1; i <= 10; i++) {
        try {
          const fileName = `plot_${i}.json`;
          const content = await this.wails.readJobOutputFile(this.jobId, fileName);
          if (content) {
            const plotData = JSON.parse(content);
            const title = plotData.layout?.title || `Plot ${i}`;
            plotFiles.push({ fileName, title });
          }
        } catch (e) {
          break;
        }
      }

      this.pluginPlots.set(plotFiles);

      if (plotFiles.length > 0) {
        await this.wails.logToFile(`[Job Detail] Found ${plotFiles.length} plugin plot(s) for job ${this.jobId}`);
      }
    } catch (err) {
      await this.wails.logToFile(`[Job Detail] Error detecting plugin plots: ${err}`);
    }
  }
}
