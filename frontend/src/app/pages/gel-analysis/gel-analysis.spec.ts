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
      detectGelBoundary: vi.fn()
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

  it('setDrawMode switches the active draw mode', async () => {
    await fixture.whenStable();
    expect(state().drawMode()).toBe('lane');

    component.setDrawMode('boundary');

    expect(state().drawMode()).toBe('boundary');
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

  it('detectBoundary expands the boundary panel on success', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 20, y: 0, width: 10, height: 4, isMarker: false }]);
    state().boundaryPadding.set(10);
    state().boundaryPanelExpanded.set(false);
    wailsMock.detectGelBoundary.mockResolvedValue({ x: 10, y: 0, width: 30, height: 24 });

    await component.detectBoundary();

    expect(state().boundaryPanelExpanded()).toBe(true);
  });

  it('loadSession expands the boundary panel when a boundary is already stored', async () => {
    await fixture.whenStable();
    wailsMock.loadGelSession.mockResolvedValue({ sessionId: 'sess-1', width: 800, height: 600 });
    wailsMock.getGelBoundary.mockResolvedValue({ x: 10, y: 0, width: 30, height: 24 });
    wailsMock.getGelImagePreview.mockRejectedValueOnce(new Error('no canvas in test environment'));
    state().boundaryPanelExpanded.set(false);

    await component.loadSession(1);

    expect(state().boundaryPanelExpanded()).toBe(true);
  });

  it('loadSession leaves the boundary panel collapsed when no boundary is stored', async () => {
    await fixture.whenStable();
    wailsMock.loadGelSession.mockResolvedValue({ sessionId: 'sess-1', width: 800, height: 600 });
    wailsMock.getGelBoundary.mockResolvedValue(null);
    wailsMock.getGelImagePreview.mockRejectedValueOnce(new Error('no canvas in test environment'));
    state().boundaryPanelExpanded.set(false);

    await component.loadSession(1);

    expect(state().boundaryPanelExpanded()).toBe(false);
  });

  it('markAsLadder expands the calibration panel', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    const lane = { id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 10, height: 10, isMarker: false };
    state().lanes.set([lane]);
    state().calibrationPanelExpanded.set(false);
    dialogMock.open.mockReturnValueOnce({ afterClosed: () => of([66000, 45000, 31000]) });

    await component.markAsLadder(lane as any);

    expect(state().calibrationPanelExpanded()).toBe(true);
  });
});
