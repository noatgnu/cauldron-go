import { ComponentFixture, TestBed } from '@angular/core/testing';
import { CompatibilityBadge } from './compatibility-badge';
import { CompatibilityReport } from '../../core/services/wails';

describe('CompatibilityBadge', () => {
  let component: CompatibilityBadge;
  let fixture: ComponentFixture<CompatibilityBadge>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CompatibilityBadge]
    })
      .compileComponents();

    fixture = TestBed.createComponent(CompatibilityBadge);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('reports unknown when no report is set', () => {
    component.report = null;
    expect(component.level()).toBe('unknown');
  });

  it('reports ok when every stage is compatible', () => {
    component.report = {
      recipeId: 'r1',
      allOk: true,
      stages: [{ stageIndex: 0, pluginId: 'a', status: 'compatible' as any }]
    } as CompatibilityReport;
    expect(component.level()).toBe('ok');
  });

  it('reports warn when a stage is compatible but the version differs', () => {
    component.report = {
      recipeId: 'r1',
      allOk: true,
      stages: [{ stageIndex: 0, pluginId: 'a', status: 'compatible_version_differs' as any, recordedVersion: '1.0.0', installedVersion: '1.1.0' }]
    } as CompatibilityReport;
    expect(component.level()).toBe('warn');
    expect(component.tooltip()).toContain('v1.0.0');
    expect(component.tooltip()).toContain('v1.1.0');
  });

  it('reports error when a stage is incompatible', () => {
    component.report = {
      recipeId: 'r1',
      allOk: false,
      stages: [{ stageIndex: 1, pluginId: 'b', status: 'incompatible' as any, missingInputs: ['gone_input'] }]
    } as CompatibilityReport;
    expect(component.level()).toBe('error');
    expect(component.tooltip()).toContain('gone_input');
  });

  it('reports error when a stage plugin is missing', () => {
    component.report = {
      recipeId: 'r1',
      allOk: false,
      stages: [{ stageIndex: 0, pluginId: 'c', status: 'missing' as any }]
    } as CompatibilityReport;
    expect(component.level()).toBe('error');
    expect(component.tooltip()).toContain('not installed');
  });
});
