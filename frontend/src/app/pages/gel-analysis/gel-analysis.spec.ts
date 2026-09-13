import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { vi, beforeAll } from 'vitest';
import { of } from 'rxjs';
import { GelAnalysis } from './gel-analysis';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { MatDialog } from '@angular/material/dialog';
import { PlotlyModule } from 'angular-plotly.js';
import { PlotlyMock, mockMatchMedia } from '../../core/mocks/plotly-mock';

describe('GelAnalysis', () => {
  let component: GelAnalysis;
  let fixture: ComponentFixture<GelAnalysis>;
  let wailsMock: any;
  let notificationMock: any;
  let dialogMock: any;

  function state(): any {
    return component as any;
  }

  beforeAll(() => {
    mockMatchMedia();
  });

  beforeEach(async () => {
    wailsMock = {
      progress: signal(null),
      openGelImageDialog: vi.fn().mockResolvedValue(''),
      loadGelImage: vi.fn(),
      getGelImagePreview: vi.fn().mockResolvedValue(''),
      getGelImagePreviewWithLevels: vi.fn().mockResolvedValue(''),
      setGelLane: vi.fn().mockResolvedValue(undefined),
      removeGelLane: vi.fn().mockResolvedValue(undefined),
      getGelLanes: vi.fn().mockResolvedValue([]),
      computeGelLaneProfile: vi.fn(),
      computeAllGelProfiles: vi.fn().mockResolvedValue({}),
      fitGelCalibrationCurve: vi.fn(),
      applyGelCalibration: vi.fn().mockResolvedValue({}),
      exportGelResultsDialog: vi.fn().mockResolvedValue(''),
      exportGelResultsCSV: vi.fn().mockResolvedValue(undefined),
      saveGelSession: vi.fn().mockResolvedValue(1),
      getGelSessions: vi.fn().mockResolvedValue([]),
      loadGelSession: vi.fn(),
      deleteGelSession: vi.fn().mockResolvedValue(undefined),
      closeGelSession: vi.fn().mockResolvedValue(undefined),
      runGelAutoDetect: vi.fn(),
      cancelGelAutoDetect: vi.fn().mockResolvedValue(undefined),
      getPluginEnvironmentBinding: vi.fn().mockResolvedValue(null),
      getGelProvenance: vi.fn(),
      getGelRawMetadata: vi.fn().mockResolvedValue({}),
      centerGelLane: vi.fn(),
      setGelBoundary: vi.fn().mockResolvedValue(undefined),
      getGelBoundary: vi.fn().mockResolvedValue(null),
      clearGelBoundary: vi.fn().mockResolvedValue(undefined),
      detectGelBoundary: vi.fn(),
      setGelBandOverride: vi.fn(),
      removeGelBandOverride: vi.fn(),
      getGelBandOverrides: vi.fn().mockResolvedValue([])
    };

    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn(),
      showInfo: vi.fn(),
      showWarning: vi.fn()
    };

    dialogMock = {
      open: vi.fn().mockReturnValue({ afterClosed: () => of(null) })
    };

    await TestBed.configureTestingModule({
      imports: [GelAnalysis, PlotlyModule.forRoot(PlotlyMock)],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock },
        { provide: MatDialog, useValue: dialogMock }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(GelAnalysis);
    component = fixture.componentInstance;
  });

  it('should create', async () => {
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });

  it('does nothing when the open-image dialog is cancelled', async () => {
    await fixture.whenStable();
    await component.openImage();
    expect(wailsMock.loadGelImage).not.toHaveBeenCalled();
  });

  it('applyDisplayLevels uses the levels-based preview when black/white points are adjusted', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().blackPoint.set(0.1);
    state().whitePoint.set(0.9);

    await component.applyDisplayLevels();

    expect(wailsMock.getGelImagePreviewWithLevels).toHaveBeenCalledWith('sess-1', 0.1, 0.9);
    expect(wailsMock.getGelImagePreview).not.toHaveBeenCalled();
  });

  it('resetDisplayLevels restores defaults and falls back to the auto-contrast preview', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().blackPoint.set(0.2);
    state().whitePoint.set(0.8);

    component.resetDisplayLevels();
    await fixture.whenStable();

    expect(state().blackPoint()).toBe(0);
    expect(state().whitePoint()).toBe(1);
    expect(wailsMock.getGelImagePreview).toHaveBeenCalledWith('sess-1');
  });

  it('removeLane removes the lane from state and calls the backend', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 10, height: 10, isMarker: false }]);

    await component.removeLane('lane1');

    expect(wailsMock.removeGelLane).toHaveBeenCalledWith('sess-1', 'lane1');
    expect(state().lanes()).toEqual([]);
  });

  it('updateSelectedLane persists a single-field edit to the selected lane', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 60, height: 600, isMarker: false }]);
    state().selectedLaneId.set('lane1');

    await component.updateSelectedLane('width', 80);

    expect(wailsMock.setGelLane).toHaveBeenCalledWith('sess-1', expect.objectContaining({ id: 'lane1', width: 80 }));
    expect(state().lanes()[0].width).toBe(80);
  });

  it('matchLaneSize copies width/height from the chosen lane onto the selected one', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([
      { id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 60, height: 600, isMarker: false },
      { id: 'lane2', label: 'Lane 2', x: 100, y: 0, width: 45, height: 550, isMarker: false }
    ]);
    state().selectedLaneId.set('lane2');

    await component.matchLaneSize('lane1');

    expect(wailsMock.setGelLane).toHaveBeenCalledWith('sess-1', expect.objectContaining({ id: 'lane2', width: 60, height: 600 }));
    const updated = state().lanes().find((l: any) => l.id === 'lane2');
    expect(updated.width).toBe(60);
    expect(updated.height).toBe(600);
    expect(updated.x).toBe(100); // position untouched, only size copied
  });

  it('centerSelectedLane calls the backend and applies the recentered lane', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 20, y: 0, width: 10, height: 4, isMarker: false }]);
    state().selectedLaneId.set('lane1');
    wailsMock.centerGelLane.mockResolvedValue({ id: 'lane1', label: 'Lane 1', x: 24.5, y: 0, width: 10, height: 4, isMarker: false });

    await component.centerSelectedLane();

    expect(wailsMock.centerGelLane).toHaveBeenCalledWith('sess-1', 'lane1');
    expect(state().lanes()[0].x).toBe(24.5);
  });

  it('centerSelectedLane does nothing when no lane is selected', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 20, y: 0, width: 10, height: 4, isMarker: false }]);

    await component.centerSelectedLane();

    expect(wailsMock.centerGelLane).not.toHaveBeenCalled();
  });

  it('detectBoundary calls the backend and stores the returned boundary', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 20, y: 0, width: 10, height: 4, isMarker: false }]);
    state().boundaryPadding.set(10);
    wailsMock.detectGelBoundary.mockResolvedValue({ x: 10, y: 0, width: 30, height: 24 });

    await component.detectBoundary();

    expect(wailsMock.detectGelBoundary).toHaveBeenCalledWith('sess-1', 10);
    expect(state().boundary()).toEqual({ x: 10, y: 0, width: 30, height: 24 });
  });

  it('clearBoundary clears local state and calls the backend', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().boundary.set({ x: 10, y: 0, width: 30, height: 24 });

    await component.clearBoundary();

    expect(wailsMock.clearGelBoundary).toHaveBeenCalledWith('sess-1');
    expect(state().boundary()).toBeNull();
  });

  it('updateBoundary persists a single-field edit to the boundary', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().boundary.set({ x: 10, y: 0, width: 30, height: 24 });

    await component.updateBoundary('width', 50);

    expect(wailsMock.setGelBoundary).toHaveBeenCalledWith('sess-1', expect.objectContaining({ x: 10, width: 50 }));
    expect(state().boundary().width).toBe(50);
  });

  it('updateBoundary does nothing when no boundary is set', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');

    await component.updateBoundary('width', 50);

    expect(wailsMock.setGelBoundary).not.toHaveBeenCalled();
  });

  it('canUndo/canRedo reflect the history stacks', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    expect(state().canUndo()).toBe(false);
    expect(state().canRedo()).toBe(false);

    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 10, height: 10, isMarker: false }]);
    await component.removeLane('lane1');

    expect(state().canUndo()).toBe(true);
    expect(state().canRedo()).toBe(false);
  });

  it('undo restores a removed lane and re-adds it on the backend', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    const lane = { id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 10, height: 10, isMarker: false };
    state().lanes.set([lane]);

    await component.removeLane('lane1');
    expect(state().lanes()).toEqual([]);

    await component.undo();

    expect(wailsMock.setGelLane).toHaveBeenCalledWith('sess-1', expect.objectContaining({ id: 'lane1' }));
    expect(state().lanes()).toEqual([lane]);
    expect(state().canUndo()).toBe(false);
    expect(state().canRedo()).toBe(true);
  });

  it('redo re-applies an undone change', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    const lane = { id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 10, height: 10, isMarker: false };
    state().lanes.set([lane]);

    await component.removeLane('lane1');
    await component.undo();
    await component.redo();

    expect(wailsMock.removeGelLane).toHaveBeenLastCalledWith('sess-1', 'lane1');
    expect(state().lanes()).toEqual([]);
    expect(state().canRedo()).toBe(false);
  });

  it('undo does nothing when the history stack is empty', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');

    await component.undo();

    expect(wailsMock.setGelLane).not.toHaveBeenCalled();
    expect(wailsMock.removeGelLane).not.toHaveBeenCalled();
  });

  it('a new action after undo clears the redo stack', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 10, height: 10, isMarker: false }]);

    await component.removeLane('lane1');
    await component.undo();
    expect(state().canRedo()).toBe(true);

    state().lanes.set([{ id: 'lane2', label: 'Lane 2', x: 0, y: 0, width: 10, height: 10, isMarker: false }]);
    await component.removeLane('lane2');

    expect(state().canRedo()).toBe(false);
  });

  it('undo restores a cleared boundary', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().boundary.set({ x: 10, y: 0, width: 30, height: 24 });

    await component.clearBoundary();
    expect(state().boundary()).toBeNull();

    await component.undo();

    expect(wailsMock.setGelBoundary).toHaveBeenCalledWith('sess-1', { x: 10, y: 0, width: 30, height: 24 });
    expect(state().boundary()).toEqual({ x: 10, y: 0, width: 30, height: 24 });
  });

  it('undo restores a band removed via removeBand', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 60, height: 600, isMarker: false }]);
    const profileWithBand = { laneId: 'lane1', values: [], baseline: [], bands: [{ position: 25, relativePosition: 0.25, intensity: 10, area: 10, width: 5, relativeQuantity: 100 }] };
    const profileWithoutBand = { laneId: 'lane1', values: [], baseline: [], bands: [] };
    wailsMock.setGelBandOverride.mockResolvedValue(profileWithoutBand);

    await component.removeBand('lane1', 25);

    expect(state().profiles()['lane1']).toEqual(profileWithoutBand);
    expect(state().bandOverrides()['lane1'].length).toBe(1);

    wailsMock.setGelBandOverride.mockResolvedValue(profileWithBand);
    await component.undo();

    expect(wailsMock.removeGelBandOverride).toHaveBeenCalledWith('sess-1', 'lane1', expect.any(String));
    expect(state().bandOverrides()['lane1'] ?? []).toEqual([]);
  });

  it('redo re-applies a band removal after undo', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 60, height: 600, isMarker: false }]);
    const profileWithoutBand = { laneId: 'lane1', values: [], baseline: [], bands: [] };
    wailsMock.setGelBandOverride.mockResolvedValue(profileWithoutBand);
    wailsMock.removeGelBandOverride.mockResolvedValue({ laneId: 'lane1', values: [], baseline: [], bands: [{ position: 25 }] });

    await component.removeBand('lane1', 25);
    await component.undo();
    await component.redo();

    expect(wailsMock.setGelBandOverride).toHaveBeenLastCalledWith('sess-1', 'lane1', expect.objectContaining({ laneId: 'lane1', position: 25, excluded: true }));
    expect(state().bandOverrides()['lane1'].length).toBe(1);
  });

  it('coalesces rapid edits to the same lane field into a single undo step', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 60, height: 600, isMarker: false }]);
    state().selectedLaneId.set('lane1');

    await component.updateSelectedLane('width', 70);
    await component.updateSelectedLane('width', 75);
    await component.updateSelectedLane('width', 80);

    expect(state().historyStack().length).toBe(1);
  });

  it('a discrete action after a coalesced edit starts a new undo step', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 60, height: 600, isMarker: false }]);
    state().selectedLaneId.set('lane1');

    await component.updateSelectedLane('width', 70);
    await component.removeLane('lane1');

    expect(state().historyStack().length).toBe(2);
  });

  it('hover guide is enabled by default', async () => {
    await fixture.whenStable();
    expect(state().hoverGuideEnabled()).toBe(true);
  });

  it('setHoverGuideEnabled(false) disables tracking and clears any active guide line', async () => {
    await fixture.whenStable();
    state().hoverY.set(150);

    component.setHoverGuideEnabled(false);

    expect(state().hoverGuideEnabled()).toBe(false);
    expect(state().hoverY()).toBeNull();
  });

  it('setHoverGuideEnabled(true) re-enables tracking', async () => {
    await fixture.whenStable();
    component.setHoverGuideEnabled(false);

    component.setHoverGuideEnabled(true);

    expect(state().hoverGuideEnabled()).toBe(true);
  });

  it('onOverlayMouseUp clears the hover guide line', async () => {
    await fixture.whenStable();
    state().hoverY.set(200);

    await component.onOverlayMouseUp();

    expect(state().hoverY()).toBeNull();
  });

  it('setDrawMode defaults to none and switches the active draw mode', async () => {
    await fixture.whenStable();
    expect(state().drawMode()).toBe('none');

    component.setDrawMode('boundary');

    expect(state().drawMode()).toBe('boundary');
  });

  it('selectBand sets the selected band and its parent lane', async () => {
    await fixture.whenStable();

    component.selectBand('lane1', 2);

    expect(state().selectedBand()).toEqual({ laneId: 'lane1', bandNumber: 2 });
    expect(state().selectedLaneId()).toBe('lane1');
  });

  it('selectBand clears the selection when the same band is clicked again', async () => {
    await fixture.whenStable();
    component.selectBand('lane1', 2);

    component.selectBand('lane1', 2);

    expect(state().selectedBand()).toBeNull();
  });

  it('selectBand switches to a different band without needing to deselect first', async () => {
    await fixture.whenStable();
    component.selectBand('lane1', 2);

    component.selectBand('lane1', 3);

    expect(state().selectedBand()).toEqual({ laneId: 'lane1', bandNumber: 3 });
  });

  it('isBandSelected reflects the current selection', async () => {
    await fixture.whenStable();
    component.selectBand('lane1', 2);

    expect(component.isBandSelected({ laneId: 'lane1', bandNumber: 2 })).toBe(true);
    expect(component.isBandSelected({ laneId: 'lane1', bandNumber: 3 })).toBe(false);
    expect(component.isBandSelected({ laneId: 'lane2', bandNumber: 2 })).toBe(false);
  });

  it('removeBand sets an excluded override and stores the updated profile', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    const updatedProfile = { laneId: 'lane1', values: [], baseline: [], bands: [{ position: 50, relativePosition: 0.5, intensity: 10, area: 10, width: 5, relativeQuantity: 100 }] };
    wailsMock.setGelBandOverride.mockResolvedValue(updatedProfile);

    await component.removeBand('lane1', 25);

    expect(wailsMock.setGelBandOverride).toHaveBeenCalledWith('sess-1', 'lane1', expect.objectContaining({ laneId: 'lane1', position: 25, excluded: true }));
    expect(state().profiles()['lane1']).toEqual(updatedProfile);
  });

  it('removeBand clears the selected band if it belonged to that lane', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().selectedBand.set({ laneId: 'lane1', bandNumber: 1 });
    wailsMock.setGelBandOverride.mockResolvedValue({ laneId: 'lane1', values: [], baseline: [], bands: [] });

    await component.removeBand('lane1', 25);

    expect(state().selectedBand()).toBeNull();
  });

  it('removeBand does nothing without a session', async () => {
    await fixture.whenStable();

    await component.removeBand('lane1', 25);

    expect(wailsMock.setGelBandOverride).not.toHaveBeenCalled();
  });

  it('onOverlayMouseUp in band mode adds a manual band at the drag span within the lane', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 100, width: 60, height: 400, isMarker: false }]);
    state().drawMode.set('band');
    (component as any).draftRect = { x: 10, y: 140, width: 20, height: 20 };
    wailsMock.setGelBandOverride.mockResolvedValue({ laneId: 'lane1', values: [], baseline: [], bands: [] });

    await component.onOverlayMouseUp();

    expect(wailsMock.setGelBandOverride).toHaveBeenCalledWith('sess-1', 'lane1', expect.objectContaining({ laneId: 'lane1', position: 50, width: 20, excluded: false }));
  });

  it('onOverlayMouseUp in band mode shows an error when the drag falls outside any lane', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 100, width: 60, height: 400, isMarker: false }]);
    state().drawMode.set('band');
    (component as any).draftRect = { x: 10, y: 10, width: 20, height: 20 };

    await component.onOverlayMouseUp();

    expect(wailsMock.setGelBandOverride).not.toHaveBeenCalled();
  });

  it('recomputeAllProfiles calls computeAllGelProfiles and stores the result', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    const profiles = { lane1: { laneId: 'lane1', values: [], baseline: [], bands: [] } };
    wailsMock.computeAllGelProfiles.mockResolvedValue(profiles);

    await component.recomputeAllProfiles();

    expect(wailsMock.computeAllGelProfiles).toHaveBeenCalledWith('sess-1', expect.any(Object));
    expect(state().profiles()).toEqual(profiles);
  });

  it('polarity defaults to auto', async () => {
    await fixture.whenStable();
    expect(state().polarity()).toBe('auto');
  });

  it('recomputeAllProfiles sends an empty polarity when set to auto, so the backend auto-detects', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    wailsMock.computeAllGelProfiles.mockResolvedValue({});

    await component.recomputeAllProfiles();

    expect(wailsMock.computeAllGelProfiles).toHaveBeenCalledWith('sess-1', expect.objectContaining({ polarity: '' }));
  });

  it('recomputeAllProfiles sends an explicit polarity when manually chosen', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().polarity.set('light-bands');
    wailsMock.computeAllGelProfiles.mockResolvedValue({});

    await component.recomputeAllProfiles();

    expect(wailsMock.computeAllGelProfiles).toHaveBeenCalledWith('sess-1', expect.objectContaining({ polarity: 'light-bands' }));
  });

  it('edgeExclusionFraction defaults to 0 and is sent as-is', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    wailsMock.computeAllGelProfiles.mockResolvedValue({});

    await component.recomputeAllProfiles();

    expect(state().edgeExclusionFraction()).toBe(0);
    expect(wailsMock.computeAllGelProfiles).toHaveBeenCalledWith('sess-1', expect.objectContaining({ edgeExclusionFraction: 0 }));
  });

  it('recomputeAllProfiles sends a set edgeExclusionFraction', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().edgeExclusionFraction.set(0.05);
    wailsMock.computeAllGelProfiles.mockResolvedValue({});

    await component.recomputeAllProfiles();

    expect(wailsMock.computeAllGelProfiles).toHaveBeenCalledWith('sess-1', expect.objectContaining({ edgeExclusionFraction: 0.05 }));
  });

  it('fitCalibration shows an error when no ladder lane is selected', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().calibrationLaneId.set(null);

    await component.fitCalibration();

    expect(notificationMock.showError).toHaveBeenCalled();
    expect(wailsMock.fitGelCalibrationCurve).not.toHaveBeenCalled();
  });

  it('fitCalibration fits then applies the calibration curve', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().calibrationLaneId.set('marker1');
    const curve = { slope: -1, intercept: 2, rSquared: 0.99, points: [] };
    wailsMock.fitGelCalibrationCurve.mockResolvedValue(curve);
    wailsMock.applyGelCalibration.mockResolvedValue({ lane1: { laneId: 'lane1', values: [], baseline: [], bands: [] } });

    await component.fitCalibration();

    expect(wailsMock.fitGelCalibrationCurve).toHaveBeenCalledWith('sess-1', 'marker1');
    expect(state().calibration()).toEqual(curve);
    expect(wailsMock.applyGelCalibration).toHaveBeenCalledWith('sess-1');
  });

  it('exportResults does nothing when the save dialog is cancelled', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    wailsMock.exportGelResultsDialog.mockResolvedValue('');

    await component.exportResults();

    expect(wailsMock.exportGelResultsCSV).not.toHaveBeenCalled();
  });

  it('exportResults exports to the chosen path', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    wailsMock.exportGelResultsDialog.mockResolvedValue('/tmp/out.csv');

    await component.exportResults();

    expect(wailsMock.exportGelResultsCSV).toHaveBeenCalledWith('sess-1', '/tmp/out.csv');
    expect(notificationMock.showSuccess).toHaveBeenCalled();
  });

  it('viewProvenance fetches and opens the provenance dialog', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    const provenance = { imageSha256: 'abc123', analysisEngineVersion: '1.0.0' };
    wailsMock.getGelProvenance.mockResolvedValue(provenance);

    await component.viewProvenance();

    expect(wailsMock.getGelProvenance).toHaveBeenCalledWith('sess-1');
    expect(dialogMock.open).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ data: { provenance } })
    );
  });

  it('viewProvenance shows an error when fetching provenance fails', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    wailsMock.getGelProvenance.mockRejectedValue(new Error('boom'));

    await component.viewProvenance();

    expect(notificationMock.showError).toHaveBeenCalled();
    expect(dialogMock.open).not.toHaveBeenCalled();
  });

  it('viewRawMetadata fetches and opens the metadata dialog', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    const metadata = { Software: 'GraphicsMagick 1.3.7', Compression: '5' };
    wailsMock.getGelRawMetadata.mockResolvedValue(metadata);

    await component.viewRawMetadata();

    expect(wailsMock.getGelRawMetadata).toHaveBeenCalledWith('sess-1');
    expect(dialogMock.open).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ data: { metadata } })
    );
  });

  it('viewRawMetadata shows an error when fetching metadata fails', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    wailsMock.getGelRawMetadata.mockRejectedValue(new Error('boom'));

    await component.viewRawMetadata();

    expect(notificationMock.showError).toHaveBeenCalled();
    expect(dialogMock.open).not.toHaveBeenCalled();
  });

  it('runAutoDetect opens PluginEnvironmentDialog when no binding exists yet', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    wailsMock.getPluginEnvironmentBinding
      .mockResolvedValueOnce(null)
      .mockResolvedValueOnce({ pluginID: 'gel-analysis', environmentType: 'python', environmentPath: '/venv' });
    wailsMock.runGelAutoDetect.mockResolvedValue({ lanes: [], deskewAngle: 0 });

    await component.runAutoDetect();

    expect(dialogMock.open).toHaveBeenCalled();
    expect(wailsMock.runGelAutoDetect).toHaveBeenCalledWith('sess-1', 0);
  });

  it('runAutoDetect skips the dialog when already bound', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    wailsMock.getPluginEnvironmentBinding.mockResolvedValue({ pluginID: 'gel-analysis', environmentType: 'python', environmentPath: '/venv' });
    wailsMock.runGelAutoDetect.mockResolvedValue({ lanes: [], deskewAngle: 0 });

    await component.runAutoDetect();

    expect(dialogMock.open).not.toHaveBeenCalled();
    expect(wailsMock.runGelAutoDetect).toHaveBeenCalledWith('sess-1', 0);
  });

  it('runAutoDetect passes the expected lane count through to the backend', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().expectedLaneCount.set(12);
    wailsMock.getPluginEnvironmentBinding.mockResolvedValue({ pluginID: 'gel-analysis', environmentType: 'python', environmentPath: '/venv' });
    wailsMock.runGelAutoDetect.mockResolvedValue({ lanes: [], deskewAngle: 0 });

    await component.runAutoDetect();

    expect(wailsMock.runGelAutoDetect).toHaveBeenCalledWith('sess-1', 12);
  });

  it('runAutoDetect skips a detected lane that overlaps an already-existing one', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 100, y: 0, width: 50, height: 200, isMarker: false }]);
    wailsMock.getPluginEnvironmentBinding.mockResolvedValue({ pluginID: 'gel-analysis', environmentType: 'python', environmentPath: '/venv' });
    wailsMock.runGelAutoDetect.mockResolvedValue({
      lanes: [{ id: 'detected1', label: 'Lane 1', x: 110, y: 0, width: 40, height: 200, isMarker: false }],
      deskewAngle: 0
    });

    await component.runAutoDetect();

    expect(state().lanes().length).toBe(1);
    expect(state().lanes()[0].id).toBe('lane1');
    expect(wailsMock.setGelLane).not.toHaveBeenCalled();
  });

  it('runAutoDetect adds a detected lane that does not overlap any existing lane', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 100, y: 0, width: 50, height: 200, isMarker: false }]);
    wailsMock.getPluginEnvironmentBinding.mockResolvedValue({ pluginID: 'gel-analysis', environmentType: 'python', environmentPath: '/venv' });
    wailsMock.runGelAutoDetect.mockResolvedValue({
      lanes: [{ id: 'detected1', label: 'Lane 2', x: 300, y: 0, width: 40, height: 200, isMarker: false }],
      deskewAngle: 0
    });

    await component.runAutoDetect();

    expect(state().lanes().length).toBe(2);
    expect(state().lanes().map((l: any) => l.id)).toContain('detected1');
    expect(wailsMock.setGelLane).toHaveBeenCalledWith('sess-1', expect.objectContaining({ id: 'detected1' }));
  });

  it('runAutoDetect adds only the non-overlapping lane from a mixed result', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 100, y: 0, width: 50, height: 200, isMarker: false }]);
    wailsMock.getPluginEnvironmentBinding.mockResolvedValue({ pluginID: 'gel-analysis', environmentType: 'python', environmentPath: '/venv' });
    wailsMock.runGelAutoDetect.mockResolvedValue({
      lanes: [
        { id: 'overlap', label: 'Lane 1', x: 110, y: 0, width: 40, height: 200, isMarker: false },
        { id: 'fresh', label: 'Lane 2', x: 300, y: 0, width: 40, height: 200, isMarker: false }
      ],
      deskewAngle: 0
    });

    await component.runAutoDetect();

    const ids = state().lanes().map((l: any) => l.id);
    expect(ids).toEqual(['lane1', 'fresh']);
    expect(notificationMock.showSuccess).toHaveBeenCalledWith('Auto-detect added 1 new lane(s) (1 already present)');
  });

  it('updateLaneIndex persists a lane index onto the selected lane', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 10, height: 10, isMarker: true }]);
    state().selectedLaneId.set('lane1');

    await component.updateLaneIndex(2);

    expect(wailsMock.setGelLane).toHaveBeenCalledWith('sess-1', expect.objectContaining({ id: 'lane1', laneIndex: 2 }));
    expect(state().lanes()[0].laneIndex).toBe(2);
  });

  it('updateLaneIndex clears the lane index when given null', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 10, height: 10, isMarker: true, laneIndex: 2 }]);
    state().selectedLaneId.set('lane1');

    await component.updateLaneIndex(null);

    expect(wailsMock.setGelLane).toHaveBeenCalledWith('sess-1', expect.objectContaining({ id: 'lane1' }));
    expect(state().lanes()[0].laneIndex).toBeUndefined();
  });

  it('runAutoDetect does nothing when the user abandons the setup dialog', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    wailsMock.getPluginEnvironmentBinding.mockResolvedValue(null);

    await component.runAutoDetect();

    expect(wailsMock.runGelAutoDetect).not.toHaveBeenCalled();
  });

  it('saveSession prompts for a name and saves it', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    dialogMock.open.mockReturnValueOnce({ afterClosed: () => of('My Session') });

    await component.saveSession();

    expect(wailsMock.saveGelSession).toHaveBeenCalledWith('sess-1', 'My Session');
    expect(notificationMock.showSuccess).toHaveBeenCalled();
  });

  it('closes the open session on destroy', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');

    await component.ngOnDestroy();

    expect(wailsMock.closeGelSession).toHaveBeenCalledWith('sess-1');
  });

  it('setZoom converts a percentage value into a scale factor', async () => {
    await fixture.whenStable();

    component.setZoom(150);

    expect(state().zoomLevel()).toBe(1.5);
  });

  it('setZoom(null) resets to fit-to-container', async () => {
    await fixture.whenStable();
    state().zoomLevel.set(2);

    component.setZoom(null);

    expect(state().zoomLevel()).toBeNull();
  });

  it('resetZoom resets to fit-to-container', async () => {
    await fixture.whenStable();
    state().zoomLevel.set(2);

    component.resetZoom();

    expect(state().zoomLevel()).toBeNull();
  });

  it('fitToWindow scales to the smaller of width/height fit ratios', async () => {
    await fixture.whenStable();
    state().imageMeta.set({ width: 1000, height: 200 });
    (component as any).canvasStackRef = { nativeElement: { clientWidth: 500, clientHeight: 500 } };

    component.fitToWindow();

    expect(state().zoomLevel()).toBeCloseTo(0.5, 5);
  });

  it('fitToWindow falls back to fit-to-container without image metadata', async () => {
    await fixture.whenStable();
    state().imageMeta.set(null);
    state().zoomLevel.set(2);

    component.fitToWindow();

    expect(state().zoomLevel()).toBeNull();
  });

  it('canvasDisplayWidth is null at fit-to-container zoom', async () => {
    await fixture.whenStable();
    state().imageMeta.set({ width: 800, height: 600 });
    state().zoomLevel.set(null);

    expect(state().canvasDisplayWidth()).toBeNull();
  });

  it('canvasDisplayWidth scales the image width by the zoom factor', async () => {
    await fixture.whenStable();
    state().imageMeta.set({ width: 800, height: 600 });
    state().zoomLevel.set(1.5);

    expect(state().canvasDisplayWidth()).toBe(1200);
  });

  it('detectBoundary switches to the boundary tab on success', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 20, y: 0, width: 10, height: 4, isMarker: false }]);
    state().boundaryPadding.set(10);
    state().selectedControlTab.set(0);
    wailsMock.detectGelBoundary.mockResolvedValue({ x: 10, y: 0, width: 30, height: 24 });

    await component.detectBoundary();

    expect(state().selectedControlTab()).toBe(1);
  });

  it('loadSession switches to the boundary tab when a boundary is already stored', async () => {
    await fixture.whenStable();
    wailsMock.loadGelSession.mockResolvedValue({ sessionId: 'sess-1', width: 800, height: 600 });
    wailsMock.getGelBoundary.mockResolvedValue({ x: 10, y: 0, width: 30, height: 24 });
    wailsMock.getGelImagePreview.mockRejectedValueOnce(new Error('no canvas in test environment'));
    state().selectedControlTab.set(0);

    await component.loadSession(1);

    expect(state().selectedControlTab()).toBe(1);
  });

  it('loadSession leaves the lanes tab active when no boundary is stored', async () => {
    await fixture.whenStable();
    wailsMock.loadGelSession.mockResolvedValue({ sessionId: 'sess-1', width: 800, height: 600 });
    wailsMock.getGelBoundary.mockResolvedValue(null);
    wailsMock.getGelImagePreview.mockRejectedValueOnce(new Error('no canvas in test environment'));
    state().selectedControlTab.set(0);

    await component.loadSession(1);

    expect(state().selectedControlTab()).toBe(0);
  });

  it('markAsLadder switches to the calibration tab', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    const lane = { id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 10, height: 10, isMarker: false };
    state().lanes.set([lane]);
    state().selectedControlTab.set(0);
    dialogMock.open.mockReturnValueOnce({ afterClosed: () => of([66000, 45000, 31000]) });

    await component.markAsLadder(lane as any);

    expect(state().selectedControlTab()).toBe(3);
  });
});
