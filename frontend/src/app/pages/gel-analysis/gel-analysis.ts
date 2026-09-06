import { Component, ElementRef, OnDestroy, ViewChild, computed, effect, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { MatSelectModule } from '@angular/material/select';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatCardModule } from '@angular/material/card';
import { MatMenuModule } from '@angular/material/menu';
import { MatDividerModule } from '@angular/material/divider';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatSliderModule } from '@angular/material/slider';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatDialog } from '@angular/material/dialog';
import { Wails, GelImageMeta, GelLaneROI, GelBoundary, GelPeakParams, GelLaneProfile, GelCalibrationCurve, GelAnalysisSession } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { GelLaneMwDialog, GelLaneMwDialogData } from '../../components/gel-lane-mw-dialog/gel-lane-mw-dialog';
import { GelCalibrationPlot } from '../../components/gel-calibration-plot/gel-calibration-plot';
import { GelProvenanceDialog, GelProvenanceDialogData } from '../../components/gel-provenance-dialog/gel-provenance-dialog';
import { GelMetadataDialog, GelMetadataDialogData } from '../../components/gel-metadata-dialog/gel-metadata-dialog';
import { PluginEnvironmentDialog, PluginEnvironmentDialogData } from '../../components/plugin-environment-dialog/plugin-environment-dialog';
import { PromptDialogComponent, PromptDialogData } from '../../components/prompt-dialog/prompt-dialog';
import { GelLaneMap } from './gel-lane-map/gel-lane-map';

interface ResultRow {
  lane: string;
  bandNumber: number;
  position: number;
  relativePosition: number;
  intensity: number;
  area: number;
  molecularWeight: number | null;
  relativeQuantity: number;
}

@Component({
  selector: 'app-gel-analysis',
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatToolbarModule,
    MatIconModule,
    MatTableModule,
    MatSelectModule,
    MatFormFieldModule,
    MatInputModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatCardModule,
    MatMenuModule,
    MatDividerModule,
    MatTooltipModule,
    MatSliderModule,
    MatExpansionModule,
    GelCalibrationPlot,
    GelLaneMap
  ],
  templateUrl: './gel-analysis.html',
  styleUrl: './gel-analysis.scss'
})
export class GelAnalysis implements OnDestroy {
  private readonly wails = inject(Wails);
  private readonly notification = inject(NotificationService);
  private readonly dialog = inject(MatDialog);

  @ViewChild('imageCanvas') imageCanvasRef?: ElementRef<HTMLCanvasElement>;
  @ViewChild('overlayCanvas') overlayCanvasRef?: ElementRef<HTMLCanvasElement>;
  @ViewChild('canvasStack') canvasStackRef?: ElementRef<HTMLDivElement>;

  protected sessionId = signal<string | null>(null);
  protected imageMeta = signal<GelImageMeta | null>(null);
  protected imagePreviewUrl = signal<string | null>(null);
  protected lanes = signal<GelLaneROI[]>([]);
  protected selectedLaneId = signal<string | null>(null);
  protected boundary = signal<GelBoundary | null>(null);
  protected boundaryPadding = signal(10);
  protected drawMode = signal<'lane' | 'boundary'>('lane');
  protected expectedLaneCount = signal(0);
  protected profiles = signal<Partial<Record<string, GelLaneProfile>>>({});
  protected calibration = signal<GelCalibrationCurve | null>(null);
  protected calibrationLaneId = signal<string | null>(null);

  protected blackPoint = signal(0);
  protected whitePoint = signal(1);
  protected zoomLevel = signal<number | null>(null);
  protected viewportRect = signal<{ left: number; top: number; width: number; height: number } | null>(null);

  protected lanesPanelExpanded = signal(true);
  protected boundaryPanelExpanded = signal(false);
  protected peakPanelExpanded = signal(true);
  protected calibrationPanelExpanded = signal(false);

  protected smoothingWindow = signal(7);
  protected minProminence = signal(0.05);
  protected minDistance = signal(0);
  protected baselineMethod = signal<'rolling-min' | 'percentile' | 'none'>('rolling-min');
  protected polarity = signal<'dark-bands' | 'light-bands'>('dark-bands');

  protected loadingImage = signal(false);
  protected computingProfiles = signal(false);
  protected autoDetecting = signal(false);
  protected autoDetectMessage = signal('');
  protected autoDetectPercentage = signal(0);

  protected sessions = signal<GelAnalysisSession[]>([]);

  protected markerLanes = computed(() => this.lanes().filter(l => l.isMarker));
  protected selectedLane = computed(() => this.lanes().find(l => l.id === this.selectedLaneId()) ?? null);
  protected otherLanes = computed(() => this.lanes().filter(l => l.id !== this.selectedLaneId()));

  /** null = fit-to-container (CSS max-width:100%); a number = explicit pixel width for zoomed scrolling. */
  protected canvasDisplayWidth = computed(() => {
    const zoom = this.zoomLevel();
    const meta = this.imageMeta();
    return zoom !== null && meta ? Math.round(meta.width * zoom) : null;
  });

  protected showMinimap = computed(() => this.zoomLevel() !== null);

  protected resultsColumns = ['lane', 'bandNumber', 'position', 'relativePosition', 'intensity', 'area', 'molecularWeight', 'relativeQuantity'];

  protected resultsRows = computed<ResultRow[]>(() => {
    const profiles = this.profiles();
    const laneById = new Map(this.lanes().map(l => [l.id, l]));
    const rows: ResultRow[] = [];

    for (const laneId of Object.keys(profiles)) {
      const lane = laneById.get(laneId);
      const profile = profiles[laneId];
      if (!profile) continue;
      profile.bands.forEach((band, i) => {
        rows.push({
          lane: lane?.label ?? laneId,
          bandNumber: i + 1,
          position: band.position,
          relativePosition: band.relativePosition,
          intensity: band.intensity,
          area: band.area,
          molecularWeight: band.molecularWeight ?? null,
          relativeQuantity: band.relativeQuantity
        });
      });
    }
    return rows;
  });

  private draftRect: { x: number; y: number; width: number; height: number } | null = null;
  private dragStart: { x: number; y: number } | null = null;

  constructor() {
    effect(() => {
      const progress = this.wails.progress();
      const sid = this.sessionId();
      if (progress && sid && progress.id === 'gel-auto-detect:' + sid) {
        this.autoDetectMessage.set(progress.message);
        this.autoDetectPercentage.set(progress.percentage);
        if (progress.status === 'completed' || progress.status === 'error') {
          this.autoDetecting.set(false);
        }
      }
    });
  }

  async ngOnDestroy() {
    const sid = this.sessionId();
    if (sid) {
      await this.wails.closeGelSession(sid).catch(() => {});
    }
  }

  private isDialogCancelled(error: unknown): boolean {
    return error instanceof Error && error.message.includes('cancelled by user');
  }

  async openImage() {
    try {
      const path = await this.wails.openGelImageDialog();
      if (!path) return;
      await this.loadImage(path);
    } catch (error) {
      if (this.isDialogCancelled(error)) return;
      this.notification.showError(`Failed to open file dialog: ${error}`);
    }
  }

  private async loadImage(path: string) {
    const previousSession = this.sessionId();
    if (previousSession) {
      await this.wails.closeGelSession(previousSession).catch(() => {});
    }

    this.loadingImage.set(true);
    try {
      const meta = await this.wails.loadGelImage(path);
      if (!meta) throw new Error('No image metadata returned');

      this.sessionId.set(meta.sessionId);
      this.imageMeta.set(meta);
      this.lanes.set([]);
      this.boundary.set(null);
      this.profiles.set({});
      this.calibration.set(null);
      this.calibrationLaneId.set(null);
      this.blackPoint.set(0);
      this.whitePoint.set(1);

      // Canvas only exists in the DOM once loadingImage is false. Flip it before drawing.
      this.loadingImage.set(false);
      await this.refreshPreview();
    } catch (error) {
      this.notification.showError(`Failed to load image: ${error}`);
      this.loadingImage.set(false);
    }
  }

  async applyDisplayLevels(): Promise<void> {
    await this.refreshPreview();
  }

  resetDisplayLevels(): void {
    this.blackPoint.set(0);
    this.whitePoint.set(1);
    this.refreshPreview();
  }

  private async refreshPreview() {
    const sid = this.sessionId();
    if (!sid) return;
    const isDefaultLevels = this.blackPoint() === 0 && this.whitePoint() === 1;
    const base64 = isDefaultLevels
      ? await this.wails.getGelImagePreview(sid)
      : await this.wails.getGelImagePreviewWithLevels(sid, this.blackPoint(), this.whitePoint());
    this.imagePreviewUrl.set(`data:image/png;base64,${base64}`);
    await this.drawImageOntoCanvas();
  }

  private drawImageOntoCanvas(retriesLeft = 5): Promise<void> {
    return new Promise((resolve) => {
      const url = this.imagePreviewUrl();
      const meta = this.imageMeta();
      const canvas = this.imageCanvasRef?.nativeElement;
      const overlay = this.overlayCanvasRef?.nativeElement;

      if (!url || !meta) {
        resolve();
        return;
      }

      // Canvas may not be mounted yet on the first tick. Retry via rAF instead of failing silently.
      if (!canvas || !overlay) {
        if (retriesLeft <= 0) {
          this.notification.showError('Failed to render the gel image: canvas was not ready. Try reopening the image.');
          resolve();
          return;
        }
        requestAnimationFrame(() => this.drawImageOntoCanvas(retriesLeft - 1).then(resolve));
        return;
      }

      canvas.width = meta.width;
      canvas.height = meta.height;
      overlay.width = meta.width;
      overlay.height = meta.height;

      const img = new Image();
      img.onload = () => {
        const ctx = canvas.getContext('2d');
        ctx?.clearRect(0, 0, canvas.width, canvas.height);
        ctx?.drawImage(img, 0, 0);
        this.redrawOverlay();
        resolve();
      };
      img.onerror = () => {
        this.notification.showError('Failed to render the gel image preview.');
        resolve();
      };
      img.src = url;
    });
  }

  private redrawOverlay() {
    const overlay = this.overlayCanvasRef?.nativeElement;
    if (!overlay) return;
    const ctx = overlay.getContext('2d');
    if (!ctx) return;

    ctx.clearRect(0, 0, overlay.width, overlay.height);

    const boundary = this.boundary();
    if (boundary) {
      ctx.strokeStyle = '#ff9100';
      ctx.setLineDash([8, 4]);
      ctx.lineWidth = 2;
      ctx.strokeRect(boundary.x, boundary.y, boundary.width, boundary.height);
      ctx.setLineDash([]);
    }

    for (const lane of this.lanes()) {
      const selected = lane.id === this.selectedLaneId();
      ctx.strokeStyle = lane.isMarker ? '#ffb300' : (selected ? '#00e5ff' : '#4caf50');
      ctx.lineWidth = selected ? 3 : 2;
      ctx.strokeRect(lane.x, lane.y, lane.width, lane.height);

      ctx.fillStyle = ctx.strokeStyle;
      ctx.font = '14px sans-serif';
      ctx.fillText(lane.label, lane.x + 2, Math.max(12, lane.y - 4));

      const profile = this.profiles()[lane.id];
      if (profile) {
        for (const band of profile.bands) {
          const y = lane.y + band.position;
          ctx.beginPath();
          ctx.moveTo(lane.x, y);
          ctx.lineTo(lane.x + lane.width, y);
          ctx.strokeStyle = '#ff1744';
          ctx.lineWidth = 1;
          ctx.stroke();
        }
      }
    }

    if (this.draftRect) {
      ctx.strokeStyle = this.drawMode() === 'boundary' ? '#ff9100' : '#00e5ff';
      ctx.setLineDash([4, 4]);
      ctx.lineWidth = 2;
      ctx.strokeRect(this.draftRect.x, this.draftRect.y, this.draftRect.width, this.draftRect.height);
      ctx.setLineDash([]);
    }
  }

  private toImageCoords(event: MouseEvent): { x: number; y: number } | null {
    const overlay = this.overlayCanvasRef?.nativeElement;
    if (!overlay) return null;
    const rect = overlay.getBoundingClientRect();
    const scaleX = overlay.width / rect.width;
    const scaleY = overlay.height / rect.height;
    return {
      x: (event.clientX - rect.left) * scaleX,
      y: (event.clientY - rect.top) * scaleY
    };
  }

  onOverlayMouseDown(event: MouseEvent) {
    const point = this.toImageCoords(event);
    if (!point) return;
    this.dragStart = point;
    this.draftRect = { x: point.x, y: point.y, width: 0, height: 0 };
  }

  onOverlayMouseMove(event: MouseEvent) {
    if (!this.dragStart) return;
    const point = this.toImageCoords(event);
    if (!point) return;

    const x = Math.min(this.dragStart.x, point.x);
    const y = Math.min(this.dragStart.y, point.y);
    const width = Math.abs(point.x - this.dragStart.x);
    const height = Math.abs(point.y - this.dragStart.y);
    this.draftRect = { x, y, width, height };
    this.redrawOverlay();
  }

  async onOverlayMouseUp() {
    const rect = this.draftRect;
    this.dragStart = null;
    this.draftRect = null;

    const sid = this.sessionId();
    if (!rect || !sid || rect.width < 3 || rect.height < 3) {
      this.redrawOverlay();
      return;
    }

    if (this.drawMode() === 'boundary') {
      const boundary: GelBoundary = { x: rect.x, y: rect.y, width: rect.width, height: rect.height } as GelBoundary;
      try {
        await this.wails.setGelBoundary(sid, boundary);
        this.boundary.set(boundary);
        this.boundaryPanelExpanded.set(true);
        this.redrawOverlay();
      } catch (error) {
        this.notification.showError(`Failed to save boundary: ${error}`);
      }
      return;
    }

    const lane: GelLaneROI = {
      id: crypto.randomUUID(),
      label: `Lane ${this.lanes().length + 1}`,
      x: rect.x,
      y: rect.y,
      width: rect.width,
      height: rect.height,
      isMarker: false
    } as GelLaneROI;

    try {
      await this.wails.setGelLane(sid, lane);
      this.lanes.update(lanes => [...lanes, lane]);
      this.selectedLaneId.set(lane.id);
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to save lane: ${error}`);
    }
  }

  selectLane(laneId: string) {
    this.selectedLaneId.set(laneId);
    this.redrawOverlay();
  }

  async updateSelectedLane(field: 'x' | 'y' | 'width' | 'height', value: number): Promise<void> {
    const sid = this.sessionId();
    const lane = this.selectedLane();
    if (!sid || !lane || !Number.isFinite(value)) return;

    const updated: GelLaneROI = { ...lane, [field]: value } as GelLaneROI;
    try {
      await this.wails.setGelLane(sid, updated);
      this.lanes.update(lanes => lanes.map(l => (l.id === lane.id ? updated : l)));
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to update lane: ${error}`);
    }
  }

  /** Sets or clears the selected lane's known position in the full expected sequence (e.g. "this ladder is lane 3 of 12"), which anchor-guided auto-detect uses to place every other lane. */
  async updateLaneIndex(value: number | null): Promise<void> {
    const sid = this.sessionId();
    const lane = this.selectedLane();
    if (!sid || !lane) return;

    const laneIndex = value === null || !Number.isFinite(value) ? undefined : Math.max(0, Math.trunc(value));
    const updated: GelLaneROI = { ...lane, laneIndex } as GelLaneROI;
    try {
      await this.wails.setGelLane(sid, updated);
      this.lanes.update(lanes => lanes.map(l => (l.id === lane.id ? updated : l)));
    } catch (error) {
      this.notification.showError(`Failed to update lane: ${error}`);
    }
  }

  async matchLaneSize(sourceLaneId: string): Promise<void> {
    const source = this.lanes().find(l => l.id === sourceLaneId);
    const sid = this.sessionId();
    const lane = this.selectedLane();
    if (!sid || !lane || !source) return;

    const updated: GelLaneROI = { ...lane, width: source.width, height: source.height } as GelLaneROI;
    try {
      await this.wails.setGelLane(sid, updated);
      this.lanes.update(lanes => lanes.map(l => (l.id === lane.id ? updated : l)));
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to update lane: ${error}`);
    }
  }

  async centerSelectedLane(): Promise<void> {
    const sid = this.sessionId();
    const lane = this.selectedLane();
    if (!sid || !lane) return;

    try {
      const updated = await this.wails.centerGelLane(sid, lane.id);
      if (!updated) return;
      this.lanes.update(lanes => lanes.map(l => (l.id === lane.id ? updated : l)));
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to center lane: ${error}`);
    }
  }

  setDrawMode(mode: 'lane' | 'boundary') {
    this.drawMode.set(mode);
  }

  /** value is a percentage (25-400); null resets to fit-to-container. */
  setZoom(value: number | null): void {
    this.zoomLevel.set(value === null ? null : value / 100);
    this.scheduleViewportRectUpdate();
  }

  resetZoom(): void {
    this.zoomLevel.set(null);
    this.viewportRect.set(null);
  }

  onCanvasScroll(): void {
    this.updateViewportRect();
  }

  /** Jumps the canvas scroll position to center on the clicked point in the minimap. */
  onMinimapClick(event: MouseEvent): void {
    const stack = this.canvasStackRef?.nativeElement;
    const minimap = event.currentTarget as HTMLElement;
    if (!stack) return;

    const rect = minimap.getBoundingClientRect();
    const fracX = (event.clientX - rect.left) / rect.width;
    const fracY = (event.clientY - rect.top) / rect.height;

    stack.scrollLeft = fracX * stack.scrollWidth - stack.clientWidth / 2;
    stack.scrollTop = fracY * stack.scrollHeight - stack.clientHeight / 2;
    this.updateViewportRect();
  }

  private scheduleViewportRectUpdate(): void {
    requestAnimationFrame(() => this.updateViewportRect());
  }

  private updateViewportRect(): void {
    const stack = this.canvasStackRef?.nativeElement;
    if (!stack || this.zoomLevel() === null || stack.scrollWidth === 0 || stack.scrollHeight === 0) {
      this.viewportRect.set(null);
      return;
    }
    this.viewportRect.set({
      left: stack.scrollLeft / stack.scrollWidth,
      top: stack.scrollTop / stack.scrollHeight,
      width: Math.min(1, stack.clientWidth / stack.scrollWidth),
      height: Math.min(1, stack.clientHeight / stack.scrollHeight)
    });
  }

  /** Derives the boundary as a padded bounding box of current lanes. There is no reliable way to find the gel's physical edge from pixel intensity alone. */
  async detectBoundary(): Promise<void> {
    const sid = this.sessionId();
    if (!sid) return;

    try {
      const boundary = await this.wails.detectGelBoundary(sid, this.boundaryPadding());
      this.boundary.set(boundary);
      this.boundaryPanelExpanded.set(true);
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to detect boundary: ${error}`);
    }
  }

  async clearBoundary(): Promise<void> {
    const sid = this.sessionId();
    if (!sid) return;

    try {
      await this.wails.clearGelBoundary(sid);
      this.boundary.set(null);
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to clear boundary: ${error}`);
    }
  }

  async updateBoundary(field: 'x' | 'y' | 'width' | 'height', value: number): Promise<void> {
    const sid = this.sessionId();
    const boundary = this.boundary();
    if (!sid || !boundary || !Number.isFinite(value)) return;

    const updated: GelBoundary = { ...boundary, [field]: value } as GelBoundary;
    try {
      await this.wails.setGelBoundary(sid, updated);
      this.boundary.set(updated);
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to update boundary: ${error}`);
    }
  }

  async removeLane(laneId: string) {
    const sid = this.sessionId();
    if (!sid) return;
    try {
      await this.wails.removeGelLane(sid, laneId);
      this.lanes.update(lanes => lanes.filter(l => l.id !== laneId));
      this.profiles.update(profiles => {
        const { [laneId]: _removed, ...rest } = profiles;
        return rest;
      });
      if (this.selectedLaneId() === laneId) this.selectedLaneId.set(null);
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to remove lane: ${error}`);
    }
  }

  async markAsLadder(lane: GelLaneROI) {
    const sid = this.sessionId();
    if (!sid) return;

    const dialogRef = this.dialog.open<GelLaneMwDialog, GelLaneMwDialogData, number[] | null>(GelLaneMwDialog, {
      width: '500px',
      data: { laneLabel: lane.label, markerMWs: lane.markerMWs }
    });

    const markerMWs = await dialogRef.afterClosed().toPromise();
    if (!markerMWs) return;

    const updated: GelLaneROI = { ...lane, isMarker: true, markerMWs } as GelLaneROI;
    try {
      await this.wails.setGelLane(sid, updated);
      this.lanes.update(lanes => lanes.map(l => (l.id === lane.id ? updated : l)));
      this.calibrationLaneId.set(lane.id);
      this.calibrationPanelExpanded.set(true);
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to update lane: ${error}`);
    }
  }

  private currentPeakParams(): GelPeakParams {
    return {
      smoothingWindow: this.smoothingWindow(),
      minProminence: this.minProminence(),
      minDistance: this.minDistance(),
      baselineMethod: this.baselineMethod(),
      polarity: this.polarity()
    } as GelPeakParams;
  }

  async recomputeAllProfiles() {
    const sid = this.sessionId();
    if (!sid) return;

    this.computingProfiles.set(true);
    try {
      const result = await this.wails.computeAllGelProfiles(sid, this.currentPeakParams());
      this.profiles.set(result);
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to compute profiles: ${error}`);
    } finally {
      this.computingProfiles.set(false);
    }
  }

  async fitCalibration() {
    const sid = this.sessionId();
    const laneId = this.calibrationLaneId();
    if (!sid || !laneId) {
      this.notification.showError('Select a ladder lane first');
      return;
    }

    try {
      const curve = await this.wails.fitGelCalibrationCurve(sid, laneId);
      this.calibration.set(curve);
      const applied = await this.wails.applyGelCalibration(sid);
      this.profiles.set(applied);
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to fit calibration curve: ${error}`);
    }
  }

  async exportResults() {
    const sid = this.sessionId();
    if (!sid) return;

    let outputPath: string;
    try {
      outputPath = await this.wails.exportGelResultsDialog('gel-analysis-results.csv');
    } catch (error) {
      if (!this.isDialogCancelled(error)) {
        this.notification.showError(`Failed to open save dialog: ${error}`);
      }
      return;
    }
    if (!outputPath) return;

    try {
      await this.wails.exportGelResultsCSV(sid, outputPath);
      this.notification.showSuccess(`Exported to ${outputPath} (with a .provenance.json audit manifest alongside it)`);
    } catch (error) {
      this.notification.showError(`Export failed: ${error}`);
    }
  }

  async viewRawMetadata() {
    const sid = this.sessionId();
    if (!sid) return;

    try {
      const metadata = await this.wails.getGelRawMetadata(sid);
      this.dialog.open<GelMetadataDialog, GelMetadataDialogData>(GelMetadataDialog, {
        width: '600px',
        data: { metadata }
      });
    } catch (error) {
      this.notification.showError(`Failed to load raw metadata: ${error}`);
    }
  }

  async viewProvenance() {
    const sid = this.sessionId();
    if (!sid) return;

    try {
      const provenance = await this.wails.getGelProvenance(sid);
      if (!provenance) throw new Error('No provenance returned');
      this.dialog.open<GelProvenanceDialog, GelProvenanceDialogData>(GelProvenanceDialog, {
        width: '600px',
        data: { provenance }
      });
    } catch (error) {
      this.notification.showError(`Failed to load provenance: ${error}`);
    }
  }

  async runAutoDetect() {
    const sid = this.sessionId();
    if (!sid) return;

    const binding = await this.wails.getPluginEnvironmentBinding('gel-analysis', 'python').catch(() => null);
    if (!binding) {
      const dialogRef = this.dialog.open<PluginEnvironmentDialog, PluginEnvironmentDialogData>(PluginEnvironmentDialog, {
        width: '600px',
        disableClose: true,
        data: {
          pluginId: 'gel-analysis',
          pluginName: 'Gel Analysis (Auto-detect)',
          runtimeEnvironments: ['python']
        }
      });
      await dialogRef.afterClosed().toPromise();

      const confirmedBinding = await this.wails.getPluginEnvironmentBinding('gel-analysis', 'python').catch(() => null);
      if (!confirmedBinding) return;
    }

    this.autoDetecting.set(true);
    this.autoDetectMessage.set('Starting...');
    this.autoDetectPercentage.set(0);
    try {
      const result = await this.wails.runGelAutoDetect(sid, this.expectedLaneCount());
      if (result?.lanes) {
        for (const lane of result.lanes) {
          await this.wails.setGelLane(sid, lane);
        }
        this.lanes.update(lanes => [...lanes, ...result.lanes]);
      }
      await this.refreshPreview();
      this.notification.showSuccess(`Auto-detect found ${result?.lanes?.length ?? 0} lane(s)`);
    } catch (error) {
      this.notification.showError(`Auto-detect failed: ${error}`);
    } finally {
      this.autoDetecting.set(false);
    }
  }

  async cancelAutoDetect() {
    const sid = this.sessionId();
    if (!sid) return;
    await this.wails.cancelGelAutoDetect(sid).catch(() => {});
  }

  async saveSession() {
    const sid = this.sessionId();
    if (!sid) return;

    const dialogRef = this.dialog.open<PromptDialogComponent, PromptDialogData, string | null>(PromptDialogComponent, {
      width: '400px',
      data: { title: 'Save Gel Analysis Session', label: 'Session name', confirmText: 'Save' }
    });
    const name = await dialogRef.afterClosed().toPromise();
    if (!name) return;

    try {
      await this.wails.saveGelSession(sid, name);
      this.notification.showSuccess('Session saved');
    } catch (error) {
      this.notification.showError(`Failed to save session: ${error}`);
    }
  }

  async refreshSessions() {
    try {
      this.sessions.set(await this.wails.getGelSessions());
    } catch (error) {
      this.notification.showError(`Failed to load sessions: ${error}`);
    }
  }

  async loadSession(id: number) {
    this.loadingImage.set(true);
    try {
      const meta = await this.wails.loadGelSession(id);
      if (!meta) throw new Error('No image metadata returned');

      this.sessionId.set(meta.sessionId);
      this.imageMeta.set(meta);
      this.profiles.set({});
      this.calibration.set(null);
      this.calibrationLaneId.set(null);
      this.lanes.set(await this.wails.getGelLanes(meta.sessionId));
      const loadedBoundary = await this.wails.getGelBoundary(meta.sessionId).catch(() => null);
      this.boundary.set(loadedBoundary);
      if (loadedBoundary) this.boundaryPanelExpanded.set(true);

      // Canvas only exists in the DOM once loadingImage is false. Flip it before drawing.
      this.loadingImage.set(false);
      await this.refreshPreview();
    } catch (error) {
      this.notification.showError(`Failed to load session: ${error}`);
      this.loadingImage.set(false);
    }
  }

  async deleteSession(id: number) {
    try {
      await this.wails.deleteGelSession(id);
      await this.refreshSessions();
    } catch (error) {
      this.notification.showError(`Failed to delete session: ${error}`);
    }
  }
}
