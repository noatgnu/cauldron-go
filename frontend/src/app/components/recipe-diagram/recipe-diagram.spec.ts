import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { RecipeDiagram } from './recipe-diagram';
import { NotificationService } from '../../core/services/notification.service';

describe('RecipeDiagram', () => {
  let component: RecipeDiagram;
  let fixture: ComponentFixture<RecipeDiagram>;
  let generateMock: any;
  let notificationMock: any;

  beforeEach(async () => {
    generateMock = vi.fn().mockResolvedValue('flowchart TD\n    S0["Stage 1"]\n');
    notificationMock = { showError: vi.fn() };

    vi.doMock('mermaid', () => ({
      default: {
        initialize: vi.fn(),
        render: vi.fn().mockResolvedValue({ svg: '<svg></svg>' })
      }
    }));
    vi.doMock('svg-pan-zoom', () => ({
      default: vi.fn().mockReturnValue({ destroy: vi.fn() })
    }));

    await TestBed.configureTestingModule({
      imports: [RecipeDiagram],
      providers: [{ provide: NotificationService, useValue: notificationMock }]
    }).compileComponents();

    fixture = TestBed.createComponent(RecipeDiagram);
    component = fixture.componentInstance;
    component.subjectId = 'recipe-1';
    component.generate = generateMock;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('generates the diagram fully collapsed on init', async () => {
    fixture.detectChanges();
    await component.ngOnChanges({ subjectId: {} as any });

    expect(generateMock).toHaveBeenCalledWith([]);
    expect(component['svg']()).toBeTruthy();
  });

  it('resets expansion and re-renders when the subject changes', async () => {
    await component.ngOnChanges({ subjectId: {} as any });
    component['expandedStages'].set(new Set([0, 2]));
    generateMock.mockClear();

    await component.ngOnChanges({ subjectId: {} as any });

    expect(generateMock).toHaveBeenCalledWith([]);
    expect(component['expandedStages']().size).toBe(0);
  });

  it('toggling a stage adds it to the expand set and re-renders', async () => {
    await component.ngOnChanges({ subjectId: {} as any });
    generateMock.mockClear();

    component['toggleStage'](2);
    await Promise.resolve();

    expect(component['expandedStages']().has(2)).toBe(true);
    expect(generateMock).toHaveBeenCalledWith([2]);
  });

  it('toggling an already-expanded stage collapses it again', async () => {
    await component.ngOnChanges({ subjectId: {} as any });
    component['toggleStage'](2);
    await Promise.resolve();
    generateMock.mockClear();

    component['toggleStage'](2);
    await Promise.resolve();

    expect(component['expandedStages']().has(2)).toBe(false);
    expect(generateMock).toHaveBeenCalledWith([]);
  });

  it('shows an error notification when generation fails', async () => {
    generateMock.mockRejectedValue(new Error('boom'));

    await component.ngOnChanges({ subjectId: {} as any });

    expect(notificationMock.showError).toHaveBeenCalled();
    expect(component['svg']()).toBeNull();
    expect(component['loading']()).toBe(false);
  });

  describe('stageIndexForToggle', () => {
    beforeEach(() => {
      component['currentRenderId'] = 'recipe-diagram-3';
    });

    it('parses a cluster (subgraph) id', () => {
      expect(component['stageIndexForToggle']('recipe-diagram-3-S6_group')).toBe(6);
    });

    it('parses a plain collapsed-box node id with the flowchart prefix and numeric suffix', () => {
      expect(component['stageIndexForToggle']('recipe-diagram-3-flowchart-S0-0')).toBe(0);
    });

    it('does not match an inner step or port node', () => {
      expect(component['stageIndexForToggle']('recipe-diagram-3-flowchart-S0_step_Start-1')).toBeNull();
      expect(component['stageIndexForToggle']('recipe-diagram-3-flowchart-S1_in_input_file-5')).toBeNull();
    });

    it('returns null for an unrelated id', () => {
      expect(component['stageIndexForToggle']('recipe-diagram-3-some-other-thing')).toBeNull();
    });
  });
});
