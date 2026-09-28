import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { RecipeDiagram, printDiagram } from './recipe-diagram';
import { NotificationService } from '../../core/services/notification.service';
import { Wails } from '../../core/services/wails';

describe('RecipeDiagram', () => {
  let component: RecipeDiagram;
  let fixture: ComponentFixture<RecipeDiagram>;
  let generateMock: any;
  let notificationMock: any;
  let wailsMock: any;

  beforeEach(async () => {
    generateMock = vi.fn().mockResolvedValue('flowchart TD\n    S0["Stage 1"]\n');
    notificationMock = { showError: vi.fn(), showSuccess: vi.fn() };
    wailsMock = {
      saveFileDialog: vi.fn().mockResolvedValue('/tmp/recipe-diagram.svg'),
      exportDiagramSVG: vi.fn().mockResolvedValue(undefined)
    };

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
      providers: [
        { provide: NotificationService, useValue: notificationMock },
        { provide: Wails, useValue: wailsMock }
      ]
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

  describe('exportSvg', () => {
    beforeEach(() => {
      component['rawSvg'] = '<svg></svg>';
    });

    it('saves the raw SVG to the dialog-picked path', async () => {
      await component['exportSvg']();

      expect(wailsMock.saveFileDialog).toHaveBeenCalledWith('Export Diagram as SVG', 'recipe-diagram-recipe-1.svg');
      expect(wailsMock.exportDiagramSVG).toHaveBeenCalledWith('/tmp/recipe-diagram.svg', '<svg></svg>');
      expect(notificationMock.showSuccess).toHaveBeenCalled();
    });

    it('does nothing when the dialog is cancelled', async () => {
      wailsMock.saveFileDialog.mockResolvedValue('');

      await component['exportSvg']();

      expect(wailsMock.exportDiagramSVG).not.toHaveBeenCalled();
    });

    it('shows an error notification when the export fails', async () => {
      wailsMock.exportDiagramSVG.mockRejectedValue(new Error('disk full'));

      await component['exportSvg']();

      expect(notificationMock.showError).toHaveBeenCalled();
    });

    it('does nothing when there is no diagram to export', async () => {
      component['rawSvg'] = null;

      await component['exportSvg']();

      expect(wailsMock.saveFileDialog).not.toHaveBeenCalled();
    });
  });

  describe('exportPdf', () => {
    afterEach(() => {
      document.body.classList.remove('printing-diagram');
      document.querySelectorAll('.print-diagram-target').forEach(el => el.remove());
    });

    it('injects the raw SVG as a print target and prints the current window', () => {
      component['rawSvg'] = '<svg><text>hi</text></svg>';
      const printSpy = vi.spyOn(window, 'print').mockImplementation(() => {});

      component['exportPdf']();

      expect(printSpy).toHaveBeenCalled();
      expect(document.body.classList.contains('printing-diagram')).toBe(false);
      expect(document.querySelector('.print-diagram-target')).toBeNull();
      printSpy.mockRestore();
    });

    it('does nothing when there is no diagram to print', () => {
      component['rawSvg'] = null;
      const printSpy = vi.spyOn(window, 'print').mockImplementation(() => {});

      component['exportPdf']();

      expect(printSpy).not.toHaveBeenCalled();
      printSpy.mockRestore();
    });
  });

  describe('printDiagram', () => {
    afterEach(() => {
      document.body.classList.remove('printing-diagram');
      document.querySelectorAll('.print-diagram-target').forEach(el => el.remove());
    });

    it('appends the svg as a direct child of body, toggles the print class, and cleans up', () => {
      const printSpy = vi.spyOn(window, 'print').mockImplementation(() => {
        expect(document.body.classList.contains('printing-diagram')).toBe(true);
        const target = document.body.querySelector(':scope > .print-diagram-target');
        expect(target?.innerHTML).toContain('<text>hi</text>');
      });

      printDiagram('<svg><text>hi</text></svg>');

      expect(printSpy).toHaveBeenCalled();
      expect(document.body.classList.contains('printing-diagram')).toBe(false);
      expect(document.querySelector('.print-diagram-target')).toBeNull();
      printSpy.mockRestore();
    });
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
