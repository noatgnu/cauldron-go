import { Component, ElementRef, HostListener, OnDestroy, ViewChild, computed, effect, inject, signal } from '@angular/core';
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
import { MatTabsModule } from '@angular/material/tabs';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatDialog } from '@angular/material/dialog';
import { Wails, GelImageMeta, GelLaneROI, GelBoundary, GelPeakParams, GelLaneProfile, GelBandOverride, GelCalibrationCurve, GelAnalysisSession } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { FilePickerService } from '../../core/services/file-picker.service';
import { GelLaneMwDialog, GelLaneMwDialogData } from '../../components/gel-lane-mw-dialog/gel-lane-mw-dialog';
import { GelCalibrationPlot } from '../../components/gel-calibration-plot/gel-calibration-plot';
import { GelProvenanceDialog, GelProvenanceDialogData } from '../../components/gel-provenance-dialog/gel-provenance-dialog';
import { GelMetadataDialog, GelMetadataDialogData } from '../../components/gel-metadata-dialog/gel-metadata-dialog';
import { PluginEnvironmentDialog, PluginEnvironmentDialogData } from '../../components/plugin-environment-dialog/plugin-environment-dialog';
import { PromptDialogComponent, PromptDialogData } from '../../components/prompt-dialog/prompt-dialog';
import { GelLaneMap } from './gel-lane-map/gel-lane-map';

interface GelHistorySnapshot {
  lanes: GelLaneROI[];
  boundary: GelBoundary | null;
  bandOverrides: Record<string, GelBandOverride[]>;
}

interface ResultRow {
  laneId: string;
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
    MatTabsModule,
    MatSlideToggleModule,
    MatButtonToggleModule,
    GelCalibrationPlot,
    GelLaneMap
  ],
  templateUrl: './gel-analysis.html',
  styleUrl: './gel-analysis.scss'
})
export class GelAnalysis implements OnDestroy {
  private readonly wails = inject(Wails);
  private readonly filePicker = inject(FilePickerService);
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
  protected selectedBand = signal<{ laneId: string; bandNumber: number } | null>(null);
  protected boundary = signal<GelBoundary | null>(null);
  protected boundaryPadding = signal(10);
  protected drawMode = signal<'none' | 'lane' | 'boundary' | 'band'>('none');
  protected expectedLaneCount = signal(0);
  protected profiles = signal<Partial<Record<string, GelLaneProfile>>>({});
  protected bandOverrides = signal<Record<string, GelBandOverride[]>>({});
  protected calibration = signal<GelCalibrationCurve | null>(null);
  protected calibrationLaneId = signal<string | null>(null);

  protected blackPoint = signal(0);
  protected whitePoint = signal(1);
  protected zoomLevel = signal<number | null>(null);
  protected viewportRect = signal<{ left: number; top: number; width: number; height: number } | null>(null);

  protected hoverGuideEnabled = signal(true);
  protected hoverY = signal<number | null>(null);

  /** 0 = Lanes, 1 = Boundary, 2 = Peak Detection, 3 = Calibration. */
  protected selectedControlTab = signal(0);

  protected smoothingWindow = signal(7);
  protected minProminence = signal(0.05);
  protected minDistance = signal(0);
  protected baselineMethod = signal<'rolling-min' | 'percentile' | 'none'>('rolling-min');
  protected polarity = signal<'auto' | 'dark-bands' | 'light-bands'>('auto');
  protected edgeExclusionFraction = signal(0);

  protected loadingImage = signal(false);
  protected computingProfiles = signal(false);
  protected autoDetecting = signal(false);
  protected autoDetectMessage = signal('');
  protected autoDetectPercentage = signal(0);

  protected sessions = signal<GelAnalysisSession[]>([]);

  protected historyStack = signal<GelHistorySnapshot[]>([]);
  protected redoStack = signal<GelHistorySnapshot[]>([]);
  protected canUndo = computed(() => this.historyStack().length > 0);
  protected canRedo = computed(() => this.redoStack().length > 0);
  private readonly maxHistory = 50;
  private pendingCoalesceKey: string | null = null;
  private coalesceTimer: ReturnType<typeof setTimeout> | null = null;

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

  protected resultsColumns = ['lane', 'bandNumber', 'position', 'relativePosition', 'intensity', 'area', 'molecularWeight', 'relativeQuantity', 'actions'];

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
          laneId,
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

  @HostListener('window:keydown', ['$event'])
  handleUndoRedoKeydown(event: KeyboardEvent): void {
    const target = event.target as HTMLElement | null;
    if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)) {
      return;
    }
    if (!(event.ctrlKey || event.metaKey)) return;

    const key = event.key.toLowerCase();
    if (key === 'z' && !event.shiftKey) {
      event.preventDefault();
      this.undo();
    } else if ((key === 'z' && event.shiftKey) || key === 'y') {
      event.preventDefault();
      this.redo();
    }
  }

  private snapshotState(): GelHistorySnapshot {
    const boundary = this.boundary();
    const bandOverrides: Record<string, GelBandOverride[]> = {};
    for (const [laneId, overrides] of Object.entries(this.bandOverrides())) {
      bandOverrides[laneId] = overrides.map(o => ({ ...o }));
    }
    return {
      lanes: this.lanes().map(l => ({ ...l })),
      boundary: boundary ? { ...boundary } : null,
      bandOverrides
    };
  }

  /** Records the current state as an undo point before a mutation. coalesceKey groups rapid repeated edits to the same field (e.g. typing digits) into a single undo step instead of one per keystroke. */
  private pushHistory(coalesceKey?: string): void {
    if (this.coalesceTimer) {
      clearTimeout(this.coalesceTimer);
      this.coalesceTimer = null;
    }

    if (!coalesceKey || coalesceKey !== this.pendingCoalesceKey) {
      this.historyStack.update(stack => {
        const next = [...stack, this.snapshotState()];
        return next.length > this.maxHistory ? next.slice(next.length - this.maxHistory) : next;
      });
      this.redoStack.set([]);
    }

    this.pendingCoalesceKey = coalesceKey ?? null;
    if (coalesceKey) {
      this.coalesceTimer = setTimeout(() => {
        this.pendingCoalesceKey = null;
        this.coalesceTimer = null;
      }, 800);
    }
  }

  private resetHistory(): void {
    this.historyStack.set([]);
    this.redoStack.set([]);
    this.pendingCoalesceKey = null;
    if (this.coalesceTimer) {
      clearTimeout(this.coalesceTimer);
      this.coalesceTimer = null;
    }
  }

  async undo(): Promise<void> {
    const stack = this.historyStack();
    if (stack.length === 0) return;
    const previous = stack[stack.length - 1];
    this.historyStack.set(stack.slice(0, -1));
    this.redoStack.update(r => [...r, this.snapshotState()]);
    await this.applySnapshot(previous);
  }

  async redo(): Promise<void> {
    const stack = this.redoStack();
    if (stack.length === 0) return;
    const next = stack[stack.length - 1];
    this.redoStack.set(stack.slice(0, -1));
    this.historyStack.update(h => [...h, this.snapshotState()]);
    await this.applySnapshot(next);
  }

  private async applySnapshot(snapshot: GelHistorySnapshot): Promise<void> {
    const sid = this.sessionId();
    if (!sid) return;

    const targetIds = new Set(snapshot.lanes.map(l => l.id));
    const removedIds = this.lanes().map(l => l.id).filter(id => !targetIds.has(id));

    try {
      for (const laneId of removedIds) {
        await this.wails.removeGelLane(sid, laneId);
      }
      for (const lane of snapshot.lanes) {
        await this.wails.setGelLane(sid, lane);
      }
      if (snapshot.boundary) {
        await this.wails.setGelBoundary(sid, snapshot.boundary);
      } else {
        await this.wails.clearGelBoundary(sid);
      }

      this.lanes.set(snapshot.lanes);
      this.boundary.set(snapshot.boundary);
      this.profiles.update(profiles => {
        const kept: Partial<Record<string, GelLaneProfile>> = {};
        for (const id of targetIds) {
          if (profiles[id]) kept[id] = profiles[id];
        }
        return kept;
      });

      const currentOverrides = this.bandOverrides();
      const overrideLaneIds = new Set([...Object.keys(snapshot.bandOverrides), ...Object.keys(currentOverrides)]);
      const nextOverrides: Record<string, GelBandOverride[]> = {};
      for (const laneId of overrideLaneIds) {
        if (!targetIds.has(laneId)) continue;

        const target = snapshot.bandOverrides[laneId] ?? [];
        const current = currentOverrides[laneId] ?? [];
        const targetIdSet = new Set(target.map(o => o.id));
        const currentIdSet = new Set(current.map(o => o.id));

        for (const o of current) {
          if (!targetIdSet.has(o.id)) {
            const updated = await this.wails.removeGelBandOverride(sid, laneId, o.id);
            if (updated) this.profiles.update(p => ({ ...p, [laneId]: updated }));
          }
        }
        for (const o of target) {
          if (!currentIdSet.has(o.id)) {
            const updated = await this.wails.setGelBandOverride(sid, laneId, o);
            if (updated) this.profiles.update(p => ({ ...p, [laneId]: updated }));
          }
        }
        if (target.length > 0) nextOverrides[laneId] = target;
      }
      this.bandOverrides.set(nextOverrides);

      const selectedLaneId = this.selectedLaneId();
      if (selectedLaneId && !targetIds.has(selectedLaneId)) {
        this.selectedLaneId.set(null);
      }
      const selectedBand = this.selectedBand();
      if (selectedBand && !targetIds.has(selectedBand.laneId)) {
        this.selectedBand.set(null);
      }
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to undo/redo: ${error}`);
    }
  }

  async openImage() {
    try {
      const path = await this.filePicker.pickFilePath(() => this.wails.openGelImageDialog(), '.tif,.tiff,.png,.jpg,.jpeg');
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
      this.bandOverrides.set({});
      this.calibration.set(null);
      this.calibrationLaneId.set(null);
      this.blackPoint.set(0);
      this.whitePoint.set(1);
      this.resetHistory();

      // Canvas only exists in the DOM once loadingImage is false. Flip it before drawing.
      this.loadingImage.set(false);
      await this.refreshPreview();
      this.fitToWindow();
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

    const selectedBand = this.selectedBand();

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
        profile.bands.forEach((band, i) => {
          const isBandSelected = selectedBand?.laneId === lane.id && selectedBand.bandNumber === i + 1;
          const halfWidth = Math.max(1, band.width) / 2;
          const top = lane.y + band.position - halfWidth;
          const color = isBandSelected ? '#ffea00' : '#ff1744';
          ctx.fillStyle = isBandSelected ? 'rgba(255, 234, 0, 0.25)' : 'rgba(255, 23, 68, 0.2)';
          ctx.fillRect(lane.x, top, lane.width, halfWidth * 2);
          ctx.strokeStyle = color;
          ctx.lineWidth = isBandSelected ? 3 : 1.5;
          ctx.strokeRect(lane.x, top, lane.width, halfWidth * 2);
        });
      }
    }

    if (this.draftRect) {
      ctx.strokeStyle = this.drawMode() === 'boundary' ? '#ff9100' : this.drawMode() === 'band' ? '#76ff03' : '#00e5ff';
      ctx.setLineDash([4, 4]);
      ctx.lineWidth = 2;
      ctx.strokeRect(this.draftRect.x, this.draftRect.y, this.draftRect.width, this.draftRect.height);
      ctx.setLineDash([]);
    }

    const hoverY = this.hoverY();
    if (hoverY !== null) {
      ctx.save();
      ctx.strokeStyle = 'rgba(0, 229, 255, 0.85)';
      ctx.setLineDash([6, 4]);
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(0, hoverY);
      ctx.lineTo(overlay.width, hoverY);
      ctx.stroke();
      ctx.restore();
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
    if (this.drawMode() === 'none') return;
    const point = this.toImageCoords(event);
    if (!point) return;
    this.dragStart = point;
    this.draftRect = { x: point.x, y: point.y, width: 0, height: 0 };
  }

  onOverlayMouseMove(event: MouseEvent) {
    const point = this.toImageCoords(event);
    if (!point) return;

    if (this.dragStart) {
      const x = Math.min(this.dragStart.x, point.x);
      const y = Math.min(this.dragStart.y, point.y);
      const width = Math.abs(point.x - this.dragStart.x);
      const height = Math.abs(point.y - this.dragStart.y);
      this.draftRect = { x, y, width, height };
      this.redrawOverlay();
      return;
    }

    if (this.hoverGuideEnabled()) {
      const withinAnyLane = this.lanes().some(l =>
        point.x >= l.x && point.x <= l.x + l.width && point.y >= l.y && point.y <= l.y + l.height
      );
      this.hoverY.set(withinAnyLane ? point.y : null);
      this.redrawOverlay();
    }
  }

  async onOverlayMouseUp() {
    const rect = this.draftRect;
    this.dragStart = null;
    this.draftRect = null;
    this.hoverY.set(null);

    const sid = this.sessionId();
    if (!rect || !sid || rect.width < 3 || rect.height < 3) {
      this.redrawOverlay();
      return;
    }

    this.pushHistory();

    if (this.drawMode() === 'band') {
      const centerY = rect.y + rect.height / 2;
      const lane = this.lanes().find(l => centerY >= l.y && centerY <= l.y + l.height);
      if (!lane) {
        this.notification.showError('Drag within a lane to add a manual band there.');
        this.redrawOverlay();
        return;
      }
      const override: GelBandOverride = {
        id: crypto.randomUUID(),
        laneId: lane.id,
        position: centerY - lane.y,
        width: rect.height,
        excluded: false
      };
      try {
        const updated = await this.wails.setGelBandOverride(sid, lane.id, override);
        if (updated) {
          this.profiles.update(profiles => ({ ...profiles, [lane.id]: updated }));
        }
        this.bandOverrides.update(m => ({ ...m, [lane.id]: [...(m[lane.id] ?? []), override] }));
        this.selectedLaneId.set(lane.id);
        this.redrawOverlay();
      } catch (error) {
        this.notification.showError(`Failed to add band: ${error}`);
      }
      return;
    }

    if (this.drawMode() === 'boundary') {
      const boundary: GelBoundary = { x: rect.x, y: rect.y, width: rect.width, height: rect.height } as GelBoundary;
      try {
        await this.wails.setGelBoundary(sid, boundary);
        this.boundary.set(boundary);
        this.selectedControlTab.set(1);
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

  selectBand(laneId: string, bandNumber: number): void {
    const current = this.selectedBand();
    if (current && current.laneId === laneId && current.bandNumber === bandNumber) {
      this.selectedBand.set(null);
    } else {
      this.selectedBand.set({ laneId, bandNumber });
      this.selectedLaneId.set(laneId);
    }
    this.redrawOverlay();
  }

  isBandSelected(row: { laneId: string; bandNumber: number }): boolean {
    const selected = this.selectedBand();
    return !!selected && selected.laneId === row.laneId && selected.bandNumber === row.bandNumber;
  }

  /** Excludes a false-positive band from the lane's profile going forward; persists with the session and survives recomputes. */
  async removeBand(laneId: string, position: number): Promise<void> {
    const sid = this.sessionId();
    if (!sid) return;

    this.pushHistory();

    const selected = this.selectedBand();
    if (selected?.laneId === laneId) {
      this.selectedBand.set(null);
    }

    const override: GelBandOverride = {
      id: crypto.randomUUID(),
      laneId,
      position,
      width: 0,
      excluded: true
    };

    try {
      const updated = await this.wails.setGelBandOverride(sid, laneId, override);
      if (updated) {
        this.profiles.update(profiles => ({ ...profiles, [laneId]: updated }));
      }
      this.bandOverrides.update(m => ({ ...m, [laneId]: [...(m[laneId] ?? []), override] }));
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to remove band: ${error}`);
    }
  }

  async updateSelectedLane(field: 'x' | 'y' | 'width' | 'height', value: number): Promise<void> {
    const sid = this.sessionId();
    const lane = this.selectedLane();
    if (!sid || !lane || !Number.isFinite(value)) return;

    this.pushHistory(`lane:${lane.id}:${field}`);
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

    this.pushHistory(`lane:${lane.id}:laneIndex`);
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

    this.pushHistory();
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

    this.pushHistory();
    try {
      const updated = await this.wails.centerGelLane(sid, lane.id);
      if (!updated) return;
      this.lanes.update(lanes => lanes.map(l => (l.id === lane.id ? updated : l)));
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to center lane: ${error}`);
    }
  }

  setDrawMode(mode: 'none' | 'lane' | 'boundary' | 'band') {
    this.drawMode.set(mode);
  }

  setHoverGuideEnabled(enabled: boolean): void {
    this.hoverGuideEnabled.set(enabled);
    if (!enabled) {
      this.hoverY.set(null);
      this.redrawOverlay();
    }
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

  /** Scales so the whole image (both width and height) fits inside the visible canvas area at once. */
  fitToWindow(): void {
    const stack = this.canvasStackRef?.nativeElement;
    const meta = this.imageMeta();
    if (!stack || !meta || meta.width === 0 || meta.height === 0) {
      this.resetZoom();
      return;
    }
    const scale = Math.min(stack.clientWidth / meta.width, stack.clientHeight / meta.height);
    this.zoomLevel.set(scale > 0 ? scale : null);
    this.scheduleViewportRectUpdate();
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

    this.pushHistory();
    try {
      const boundary = await this.wails.detectGelBoundary(sid, this.boundaryPadding());
      this.boundary.set(boundary);
      this.selectedControlTab.set(1);
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to detect boundary: ${error}`);
    }
  }

  async clearBoundary(): Promise<void> {
    const sid = this.sessionId();
    if (!sid) return;

    this.pushHistory();
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

    this.pushHistory(`boundary:${field}`);
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
    this.pushHistory();
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

    this.pushHistory();
    const updated: GelLaneROI = { ...lane, isMarker: true, markerMWs } as GelLaneROI;
    try {
      await this.wails.setGelLane(sid, updated);
      this.lanes.update(lanes => lanes.map(l => (l.id === lane.id ? updated : l)));
      this.calibrationLaneId.set(lane.id);
      this.selectedControlTab.set(3);
      this.redrawOverlay();
    } catch (error) {
      this.notification.showError(`Failed to update lane: ${error}`);
    }
  }

  private currentPeakParams(): GelPeakParams {
    const polarity = this.polarity();
    return {
      smoothingWindow: this.smoothingWindow(),
      minProminence: this.minProminence(),
      minDistance: this.minDistance(),
      baselineMethod: this.baselineMethod(),
      polarity: polarity === 'auto' ? '' : polarity,
      edgeExclusionFraction: this.edgeExclusionFraction()
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
      const detected = result?.lanes ?? [];
      const existing = this.lanes();
      const newLanes = detected.filter(lane =>
        !existing.some(l => lane.x < l.x + l.width && lane.x + lane.width > l.x)
      );

      if (newLanes.length > 0) {
        this.pushHistory();
        for (const lane of newLanes) {
          await this.wails.setGelLane(sid, lane);
        }
        this.lanes.update(lanes => [...lanes, ...newLanes]);
      }
      await this.refreshPreview();

      const skipped = detected.length - newLanes.length;
      if (skipped > 0) {
        this.notification.showSuccess(`Auto-detect added ${newLanes.length} new lane(s) (${skipped} already present)`);
      } else {
        this.notification.showSuccess(`Auto-detect found ${newLanes.length} lane(s)`);
      }
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
      const loadedLanes = await this.wails.getGelLanes(meta.sessionId);
      this.lanes.set(loadedLanes);
      const loadedBoundary = await this.wails.getGelBoundary(meta.sessionId).catch(() => null);
      this.boundary.set(loadedBoundary);
      if (loadedBoundary) this.selectedControlTab.set(1);

      const bandOverrides: Record<string, GelBandOverride[]> = {};
      for (const lane of loadedLanes) {
        const overrides = await this.wails.getGelBandOverrides(meta.sessionId, lane.id).catch(() => []);
        if (overrides.length > 0) bandOverrides[lane.id] = overrides;
      }
      this.bandOverrides.set(bandOverrides);
      this.resetHistory();

      // Canvas only exists in the DOM once loadingImage is false. Flip it before drawing.
      this.loadingImage.set(false);
      await this.refreshPreview();
      this.fitToWindow();
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
