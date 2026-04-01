import { DataFrame } from 'data-forge';
import { Component, OnInit, OnDestroy, AfterViewChecked, signal, ViewChild, ElementRef, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatChipsModule } from '@angular/material/chips';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatTabsModule } from '@angular/material/tabs';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatDialog } from '@angular/material/dialog';
import { Events } from '@wailsio/runtime';
import { Wails, Job } from '../../core/services/wails';
import * as models from '../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';
import { PcaPlot } from './pca-plot/pca-plot';
import { PhatePlot } from './phate-plot/phate-plot';
import { FuzzyClusteringPlot } from './fuzzy-clustering-plot/fuzzy-clustering-plot';
import { PluginPlot } from './plugin-plot/plugin-plot';
import { GenericPlot } from './generic-plot/generic-plot';
import { SampleAnnotation, SampleAnnotationData } from '../../components/sample-annotation/sample-annotation';
import { Annotation, AnnotationService } from '../../core/services/annotation.service';
import { NotificationService } from '../../core/services/notification.service';

@Component({
  selector: 'app-job-detail',
  imports: [
    CommonModule,
    FormsModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatChipsModule,
    MatExpansionModule,
    MatTabsModule,
    MatInputModule,
    MatFormFieldModule,
    MatTooltipModule,
    PcaPlot,
    PhatePlot,
    FuzzyClusteringPlot,
    PluginPlot,
    GenericPlot
  ],
  templateUrl: './job-detail.html',
  styleUrl: './job-detail.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class JobDetail implements OnInit, OnDestroy, AfterViewChecked {
  protected job = signal<Job | null>(null);
  protected loading = signal(true);
  protected error = signal('');
  protected jobId: string = '';
  protected pluginPlots = signal<Array<{ fileName: string, title: string, type: string }>>([]);
  protected dynamicPlots = signal<Array<{ config: any, outputs: any[] }>>([]);
  protected dedicatedPlots = signal<Array<{ config: any }>>([]);
  protected defaultDynamicPlotIndex = signal<number>(0);
  protected defaultPluginPlotIndex = signal<number>(0);
  protected currentPlugin: any = null;
  private shouldAutoScroll = false;
  protected fullLogLoaded = signal<boolean>(false);
  protected totalLogLines = signal<number>(0);
  protected displayedLogLines = signal<number>(0);
  private fullLog: string[] = [];
  protected searchTerm = signal<string>('');
  protected caseSensitive = signal<boolean>(false);
  protected currentMatchIndex = signal<number>(0);
  protected filteredOutput = computed(() => {
    const term = this.searchTerm();
    const output = this.job()?.terminalOutput || [];

    if (!term.trim()) {
      return output.map((line, index) => ({ line, originalIndex: index, highlighted: line }));
    }

    const searchPattern = this.caseSensitive() ? term : term.toLowerCase();
    const filtered = output
      .map((line, index) => {
        const searchLine = this.caseSensitive() ? line : line.toLowerCase();
        if (searchLine.includes(searchPattern)) {
          const highlighted = this.highlightMatches(line, term);
          return { line, originalIndex: index, highlighted };
        }
        return null;
      })
      .filter((item): item is { line: string; originalIndex: number; highlighted: string } => item !== null);

    return filtered;
  });
  protected matchCount = computed(() => this.filteredOutput().length);
  protected showingAllLines = computed(() => !this.searchTerm().trim());

  @ViewChild('terminalOutput') private terminalOutput?: ElementRef;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private wails: Wails,
    private dialog: MatDialog,
    private annotationService: AnnotationService,
    private notificationService: NotificationService
  ) {}

  async ngOnInit() {
    this.route.params.subscribe(async params => {
      this.jobId = params['id'];
      await this.loadJob();
    });

    Events.On('job:update', async (ev: any) => {
        const data = ev.data as Job;
        if (data.id === this.jobId) {
          const previousStatus = this.job()?.status;
          const currentTerminalOutput = this.job()?.terminalOutput || [];

          this.job.set(new models.Job({
            ...data,
            terminalOutput: currentTerminalOutput.length > (data.terminalOutput?.length || 0)
              ? currentTerminalOutput
              : data.terminalOutput
          }));

          if (previousStatus !== 'completed' && data.status === 'completed') {
            await this.detectPluginPlots();
          }
        }
      });

    Events.On('job:output', (ev: any) => {
      const data = ev.data as { jobId: string; output: string };
      if (data.jobId === this.jobId) {
        this.job.update(currentJob => {
          if (currentJob) {
            const newJob = new models.Job({
              ...currentJob,
              terminalOutput: [...(currentJob.terminalOutput || []), data.output]
            });
            this.shouldAutoScroll = true;

            const newLineCount = newJob.terminalOutput?.length || 0;
            this.displayedLogLines.set(newLineCount);
            this.totalLogLines.set(newLineCount);

            return newJob;
          }
          return currentJob;
        });
      }
    });
  }

  ngAfterViewChecked() {
    if (this.shouldAutoScroll && this.terminalOutput) {
      try {
        this.terminalOutput.nativeElement.scrollTop = this.terminalOutput.nativeElement.scrollHeight;
      } catch (err) {
      }
      this.shouldAutoScroll = false;
    }
  }

  ngOnDestroy() {
    if (window.runtime) {
      window.runtime.EventsOff('job:update');
      window.runtime.EventsOff('job:output');
    }
  }

  async loadJob() {
    this.loading.set(true);
    try {
      const jobData = await this.wails.getJob(this.jobId);

      try {
        const executionLog = await this.wails.getJobExecutionLog(this.jobId);
        if (executionLog && executionLog.trim()) {
          this.fullLog = executionLog.split('\n').filter((line: string) => line.trim());
          this.totalLogLines.set(this.fullLog.length);

          const maxLines = 1000;
          if (this.fullLog.length > maxLines) {
            jobData.terminalOutput = this.fullLog.slice(-maxLines);
            this.displayedLogLines.set(maxLines);
            this.fullLogLoaded.set(false);
          } else {
            jobData.terminalOutput = this.fullLog;
            this.displayedLogLines.set(this.fullLog.length);
            this.fullLogLoaded.set(true);
          }
        } else {
          this.displayedLogLines.set(jobData.terminalOutput?.length || 0);
          this.totalLogLines.set(jobData.terminalOutput?.length || 0);
          this.fullLogLoaded.set(true);
        }
      } catch (err) {
        await this.wails.logToFile(`[Job Detail] Could not load execution log: ${err}`);
        this.displayedLogLines.set(jobData.terminalOutput?.length || 0);
        this.totalLogLines.set(jobData.terminalOutput?.length || 0);
        this.fullLogLoaded.set(true);
      }

      this.job.set(jobData);

      const pluginId = jobData.parameters?.['pluginId'];
      if (pluginId && typeof pluginId === 'number') {
        try {
          const plugin = await this.wails.getPlugin(pluginId);
          if (plugin) {
            this.currentPlugin = plugin;
          }
        } catch (err) {
          await this.wails.logToFile(`[Job Detail] Could not load plugin ${pluginId}: ${err}`);
        }
      }

      if (jobData.status === 'completed') {
        await this.detectPluginPlots();
      }
    } catch (err: any) {
      this.error.set(err.message || 'Failed to load job');
    } finally {
      this.loading.set(false);
    }
  }

  loadFullLog() {
    if (this.fullLog.length > 0) {
      this.job.update(currentJob => {
        if (currentJob) {
          const updatedJob = new models.Job({
            ...currentJob,
            terminalOutput: this.fullLog
          });
          this.displayedLogLines.set(this.fullLog.length);
          this.fullLogLoaded.set(true);
          this.shouldAutoScroll = true;
          return updatedJob;
        }
        return currentJob;
      });
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
    try {
      const annotations = await this.annotationService.loadAnnotationsForJob(this.jobId);
      await this.wails.logToFile(`[Job Detail] Loaded ${annotations.length} annotations for job ${this.jobId}`);

      let samples: string[] = [];
      if (annotations.length === 0) {
        samples = await this.extractSampleNames();
        await this.wails.logToFile(`[Job Detail] Extracted ${samples.length} sample names`);
      } else {
        await this.wails.logToFile(`[Job Detail] Annotations data: ${JSON.stringify(annotations)}`);
      }

      const dialogData: SampleAnnotationData = {
        mode: annotations.length > 0 ? 'edit' : 'create',
        samples: samples.length > 0 ? samples : undefined,
        annotation: annotations.length > 0 ? new DataFrame(annotations.map(a => ({ Sample: a.sample, Condition: a.condition || '', BioReplicate: a.bioreplicate || '', Batch: a.batch || '', Color: a.color || '' }))) : undefined,
      };

      console.log('[Job Detail] Dialog data:', { mode: dialogData.mode, hasAnnotations: !!dialogData.annotation, hasSamples: !!dialogData.samples });
      await this.wails.logToFile(`[Job Detail] Opening dialog in ${dialogData.mode} mode with ${annotations.length} annotations and ${samples.length} samples`);

      const dialogRef = this.dialog.open(SampleAnnotation, {
        width: '90vw',
        maxWidth: '1200px',
        height: '80vh',
        disableClose: true,
        data: dialogData
      });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        const annotationsToSave: Annotation[] = result.map((a: any) => ({
          sample: a.Sample,
          condition: a.Condition,
          bioreplicate: a.BioReplicate,
          batch: a.Batch,
          color: a.Color,
        }));

        await this.annotationService.saveAnnotationsForJob(this.jobId, annotationsToSave);

        await this.wails.logToFile(`[Job Detail] Saved annotations for job ${this.jobId}`);
        this.router.navigateByUrl('/', { skipLocationChange: true }).then(() => {
          this.router.navigate(['/jobs', this.jobId]);
        });
      }
    });
    } catch (error) {
      await this.wails.logToFile(`[Job Detail] Error opening annotation dialog: ${error}`);
    }
  }

  async extractSampleNames(): Promise<string[]> {
    try {
      if (!this.currentPlugin) {
        await this.wails.logToFile('[Job Detail] No plugin loaded, cannot extract sample names');
        return [];
      }

      const job = this.job();
      if (!job || !job.parameters) {
        await this.wails.logToFile('[Job Detail] No job parameters available');
        return [];
      }

      if (!this.currentPlugin.definition.annotation) {
        await this.wails.logToFile('[Job Detail] No annotation config in plugin definition');
        return [];
      }

      const inputName = this.currentPlugin.definition.annotation.samplesFrom;
      if (!inputName) {
        await this.wails.logToFile('[Job Detail] No samplesFrom specified in annotation config');
        return [];
      }

      const selectedColumns = job.parameters[inputName];

      if (!selectedColumns || !Array.isArray(selectedColumns)) {
        await this.wails.logToFile(`[Job Detail] No columns selected for ${inputName}`);
        return [];
      }

      const samples: string[] = selectedColumns.map((col: string) => {
        const sampleName = col.split(/[/\\]/).pop() || col;
        return sampleName;
      });

      await this.wails.logToFile(`[Job Detail] Extracted ${samples.length} samples from job parameters: ${inputName}`);
      return samples;

    } catch (err) {
      await this.wails.logToFile(`[Job Detail] Failed to extract sample names: ${err}`);
    }
    return [];
  }

  supportsAnnotations(): boolean {
    if (!this.currentPlugin) {
      return false;
    }

    return !!(this.currentPlugin.definition.annotation?.samplesFrom ||
             this.currentPlugin.definition.annotation?.annotationFile);
  }

  async detectImageGridPlots(plotConfig: any, jsonPlots: Array<{ fileName: string, title: string, type: string }>) {
    try {
      const imagePattern = plotConfig.config?.imagePattern || '*.svg';
      const matchType = plotConfig.config?.imagePatternType || 'auto';
      await this.wails.logToFile(`[Job Detail] Detecting image grid plots with pattern: ${imagePattern}, type: ${matchType}`);

      const files = await this.wails.listJobOutputFiles(this.jobId);
      await this.wails.logToFile(`[Job Detail] Found ${files.length} output files`);

      let matchingFiles: string[];
      let extension: string;

      if (matchType === 'exact') {
        matchingFiles = files.filter((f: string) => f === imagePattern);
        extension = imagePattern.split('.').pop() || 'svg';
      } else if (matchType === 'pattern' || (matchType === 'auto' && imagePattern.includes('*'))) {
        if (imagePattern.startsWith('*')) {
          extension = imagePattern.replace('*.', '');
          matchingFiles = files.filter((f: string) => f.endsWith(`.${extension}`) && !f.startsWith('.'));
        } else {
          const regex = new RegExp('^' + imagePattern.replace(/\*/g, '.*').replace(/\./g, '\\.') + '$');
          matchingFiles = files.filter((f: string) => regex.test(f) && !f.startsWith('.'));
          extension = imagePattern.split('.').pop() || 'svg';
        }
      } else {
        matchingFiles = files.filter((f: string) => f === imagePattern);
        extension = imagePattern.split('.').pop() || 'svg';
      }

      await this.wails.logToFile(`[Job Detail] Found ${matchingFiles.length} matching image files`);

      for (const file of matchingFiles) {
        const title = file.replace(`.${extension}`, '').replace(/_/g, ' ');
        jsonPlots.push({
          fileName: file,
          title: title,
          type: extension
        });
        await this.wails.logToFile(`[Job Detail] Added image plot: ${file}`);
      }
    } catch (err) {
      await this.wails.logToFile(`[Job Detail] Error detecting image grid plots: ${err}`);
    }
  }

  async detectPluginPlots() {
    try {
      const jsonPlots: Array<{ fileName: string, title: string, type: string }> = [];
      const dynamicPlotsArray: Array<{ config: any, outputs: any[] }> = [];
      const dedicatedPlotsArray: Array<{ config: any }> = [];

      if (!this.currentPlugin) {
        await this.wails.logToFile('[Job Detail] No plugin loaded for plot detection');
        return;
      }

      const plugin = this.currentPlugin;
      await this.wails.logToFile(`[Job Detail] Plugin definition: ${JSON.stringify(plugin.definition)}`);

      if (plugin.definition.plots && plugin.definition.plots.length > 0) {
        await this.wails.logToFile(`[Job Detail] Found ${plugin.definition.plots.length} plot(s) in plugin definition`);
        for (let i = 0; i < plugin.definition.plots.length; i++) {
          const plotConfig = plugin.definition.plots[i];
          const component = plotConfig.component;
          const plotType = plotConfig.type;
          const knownComponents = ['PcaPlot', 'PhatePlot', 'FuzzyClusteringPlot'];

          await this.wails.logToFile(`[Job Detail] Processing plot: ${plotConfig.id}, type: ${plotType}, component: ${component}, default: ${plotConfig.default}`);

          if (component && knownComponents.includes(component)) {
            dedicatedPlotsArray.push({ config: plotConfig });
            await this.wails.logToFile(`[Job Detail] Added dedicated plot: ${plotConfig.id}`);
          } else if (plotType === 'image-grid') {
            const plotsBeforeImageGrid = jsonPlots.length;
            await this.detectImageGridPlots(plotConfig, jsonPlots);
            if (plotConfig.default && jsonPlots.length > plotsBeforeImageGrid) {
              this.defaultPluginPlotIndex.set(plotsBeforeImageGrid);
              await this.wails.logToFile(`[Job Detail] Set default plugin plot index to ${plotsBeforeImageGrid}`);
            }
          } else if (!component) {
            if (plotConfig.default) {
              this.defaultDynamicPlotIndex.set(dynamicPlotsArray.length);
              await this.wails.logToFile(`[Job Detail] Set default dynamic plot index to ${dynamicPlotsArray.length}`);
            }
            dynamicPlotsArray.push({
              config: plotConfig,
              outputs: plugin.definition.outputs || []
            });
            await this.wails.logToFile(`[Job Detail] Added dynamic plot: ${plotConfig.id}`);
          }
        }
      } else {
        await this.wails.logToFile('[Job Detail] No plots defined in plugin definition');
      }

      for (let i = 1; i <= 10; i++) {
        try {
          const fileName = `plot_${i}.json`;
          await this.wails.readJobOutputFile(this.jobId, fileName);
          jsonPlots.push({
            fileName: fileName,
            title: `Plot ${i}`,
            type: 'json'
          });
        } catch (e) {
          break;
        }
      }

      this.dedicatedPlots.set(dedicatedPlotsArray);
      this.dynamicPlots.set(dynamicPlotsArray);
      this.pluginPlots.set(jsonPlots);

      if (dedicatedPlotsArray.length > 0 || dynamicPlotsArray.length > 0 || jsonPlots.length > 0) {
        await this.wails.logToFile(`[Job Detail] Found ${dedicatedPlotsArray.length} dedicated, ${dynamicPlotsArray.length} dynamic, and ${jsonPlots.length} JSON plot(s) for job ${this.jobId}`);
      }
    } catch (err) {
      await this.wails.logToFile(`[Job Detail] Error detecting plugin plots: ${err}`);
    }
  }

  async openResultFolder() {
    const outputPath = this.job()?.outputPath;
    if (outputPath) {
      try {
        await this.wails.openDirectoryInExplorer(outputPath);
      } catch (error) {
        await this.wails.logToFile(`[Job Detail] Failed to open result folder: ${error}`);
      }
    }
  }

  highlightMatches(line: string, searchTerm: string): string {
    if (!searchTerm.trim()) {
      return line;
    }

    const flags = this.caseSensitive() ? 'g' : 'gi';
    const escapedTerm = searchTerm.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const regex = new RegExp(escapedTerm, flags);

    return line.replace(regex, (match) => `<mark>${match}</mark>`);
  }

  clearSearch() {
    this.searchTerm.set('');
    this.currentMatchIndex.set(0);
  }

  toggleCaseSensitive() {
    this.caseSensitive.update(val => !val);
    this.currentMatchIndex.set(0);
  }

  nextMatch() {
    const count = this.matchCount();
    if (count > 0) {
      this.currentMatchIndex.update(val => (val + 1) % count);
    }
  }

  previousMatch() {
    const count = this.matchCount();
    if (count > 0) {
      this.currentMatchIndex.update(val => (val - 1 + count) % count);
    }
  }

  async cloneJob() {
    const currentJob = this.job();
    if (!currentJob) {
      return;
    }

    try {
      await this.wails.logToFile(`[JobDetail] Cloning job ${currentJob.id} (plugin: ${currentJob.type})`);

      const allPlugins = await this.wails.getPluginsV2();
      await this.wails.logToFile(`[JobDetail] Found ${allPlugins.length} installed plugins`);

      const pluginIds = allPlugins.map((p: any) => p.definition?.plugin?.id || 'unknown');
      await this.wails.logToFile(`[JobDetail] Available plugin IDs: ${pluginIds.join(', ')}`);

      const plugin = allPlugins.find((p: any) => p.definition?.plugin?.id === currentJob.type);

      if (!plugin) {
        await this.wails.logToFile(`[JobDetail] Plugin ${currentJob.type} not found in installed plugins`);
        this.notificationService.showError(
          `Plugin "${currentJob.type}" is not installed. Please install it from the plugin registry before cloning this job.`
        );
        return;
      }

      await this.wails.logToFile(`[JobDetail] Found plugin: ${plugin.definition.plugin.name} (ID: ${plugin.id})`);
      await this.wails.logToFile(`[JobDetail] Navigating to plugin ${plugin.id} with job ${currentJob.id}`);

      this.router.navigate(['/plugin', plugin.id], {
        queryParams: { cloneJobId: currentJob.id }
      });
    } catch (error) {
      await this.wails.logToFile(`[JobDetail] Error cloning job: ${error}`);
      this.notificationService.showError(`Failed to clone job: ${error}`);
    }
  }
}
