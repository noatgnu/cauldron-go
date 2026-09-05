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
      getGelProvenance: vi.fn()
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

  it('removeLane removes the lane from state and calls the backend', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    state().lanes.set([{ id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 10, height: 10, isMarker: false }]);

    await component.removeLane('lane1');

    expect(wailsMock.removeGelLane).toHaveBeenCalledWith('sess-1', 'lane1');
    expect(state().lanes()).toEqual([]);
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

  it('runAutoDetect opens PluginEnvironmentDialog when no binding exists yet', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    wailsMock.getPluginEnvironmentBinding
      .mockResolvedValueOnce(null)
      .mockResolvedValueOnce({ pluginID: 'gel-analysis', environmentType: 'python', environmentPath: '/venv' });
    wailsMock.runGelAutoDetect.mockResolvedValue({ lanes: [], deskewAngle: 0, usedFallback: false });

    await component.runAutoDetect();

    expect(dialogMock.open).toHaveBeenCalled();
    expect(wailsMock.runGelAutoDetect).toHaveBeenCalledWith('sess-1');
  });

  it('runAutoDetect skips the dialog when already bound', async () => {
    await fixture.whenStable();
    state().sessionId.set('sess-1');
    wailsMock.getPluginEnvironmentBinding.mockResolvedValue({ pluginID: 'gel-analysis', environmentType: 'python', environmentPath: '/venv' });
    wailsMock.runGelAutoDetect.mockResolvedValue({ lanes: [], deskewAngle: 0, usedFallback: false });

    await component.runAutoDetect();

    expect(dialogMock.open).not.toHaveBeenCalled();
    expect(wailsMock.runGelAutoDetect).toHaveBeenCalledWith('sess-1');
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
});
