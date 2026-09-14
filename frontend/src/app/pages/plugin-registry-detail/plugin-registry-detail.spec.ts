import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { PluginRegistryDetail } from './plugin-registry-detail';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { DomSanitizer } from '@angular/platform-browser';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';

describe('PluginRegistryDetail', () => {
  let component: PluginRegistryDetail;
  let fixture: ComponentFixture<PluginRegistryDetail>;
  let activatedRouteMock: any;
  let routerMock: any;
  let wailsMock: any;
  let notificationMock: any;
  let pluginV2ServiceMock: any;
  let sanitizerMock: any;
  let dialogMock: any;

  beforeEach(async () => {
    activatedRouteMock = {
      paramMap: of({ get: () => '1' }),
      snapshot: {
        paramMap: {
          get: vi.fn().mockReturnValue('1')
        }
      }
    };
    routerMock = {
      navigate: vi.fn()
    };
    wailsMock = {
      getRegistryPlugin: vi.fn().mockResolvedValue({ id: '1', name: 'Test', description: 'Test', author: { name: 'Test' }, categories: [] }),
      isPluginInstalled: vi.fn().mockResolvedValue(false),
      getPluginVersion: vi.fn().mockResolvedValue('1.0.0'),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };
    pluginV2ServiceMock = {
      getAllPlugins: vi.fn().mockResolvedValue([])
    };
    sanitizerMock = {
      bypassSecurityTrustHtml: vi.fn().mockImplementation((val) => val)
    };
    dialogMock = {
      open: vi.fn().mockReturnValue({ afterClosed: () => of(undefined) })
    };

    await TestBed.configureTestingModule({
      imports: [PluginRegistryDetail, NoopAnimationsModule],
      providers: [
        { provide: ActivatedRoute, useValue: activatedRouteMock },
        { provide: Router, useValue: routerMock },
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock },
        { provide: PluginV2Service, useValue: pluginV2ServiceMock },
        { provide: DomSanitizer, useValue: sanitizerMock }
      ]
    })
    // overrideProvider reliably replaces MatDialog even though the component imports MatDialogModule itself.
    .overrideProvider(MatDialog, { useValue: dialogMock })
    .compileComponents();

    fixture = TestBed.createComponent(PluginRegistryDetail);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('installPlugin', () => {
    beforeEach(() => {
      (component as any).plugin.set({
        id: '1',
        name: 'Test',
        description: 'Test',
        repository: 'https://github.com/test/repo',
        author: { name: 'Test' }
      });
    });

    it('sets checkingDependencies while fetching and clears it on success', async () => {
      let sawCheckingTrue = false;
      wailsMock.fetchPluginDependencies = vi.fn().mockImplementation(async () => {
        sawCheckingTrue = (component as any).checkingDependencies();
        return { hasPythonDeps: false, hasRDeps: true, runtimeEnvironments: ['r'] };
      });

      await component.installPlugin();

      expect(sawCheckingTrue).toBe(true);
      expect((component as any).checkingDependencies()).toBe(false);
      expect(dialogMock.open).toHaveBeenCalled();
    });

    it('clears checkingDependencies even when the dependency fetch fails', async () => {
      wailsMock.fetchPluginDependencies = vi.fn().mockRejectedValue(new Error('network error'));

      await component.installPlugin();

      expect((component as any).checkingDependencies()).toBe(false);
      expect(dialogMock.open).toHaveBeenCalled();
    });
  });

  describe('convertMarkdownToHtml', () => {
    it('preserves underscores inside inline code spans', () => {
      const html = component.convertMarkdownToHtml('- **Entrypoint**: `total_proteomics_qc_normalisation.R`');
      expect(html).toContain('<code>total_proteomics_qc_normalisation.R</code>');
      expect(html).not.toContain('totalproteomicsqc_normalisation.R');
    });

    it('preserves underscores inside inline code within a markdown table', () => {
      const markdown = [
        '| Name | Label |',
        '|------|-------|',
        '| `pg_matrix_file` | Protein Group Matrix File |',
        '| `stats_file` | Stats File |',
        '| `annotation_file` | Sample Annotation File |',
        '| `min_unique_peptides` | Minimum Proteotypic Peptides |'
      ].join('\n');

      const html = component.convertMarkdownToHtml(markdown);

      expect(html).toContain('<code>pg_matrix_file</code>');
      expect(html).toContain('<code>stats_file</code>');
      expect(html).toContain('<code>annotation_file</code>');
      expect(html).toContain('<code>min_unique_peptides</code>');
      expect(html).not.toContain('pgmatrixfile');
      expect(html).not.toContain('statsfile');
      expect(html).not.toContain('annotationfile');
      expect(html).not.toContain('minuniquepeptides');
    });

    it('still applies real emphasis markers outside code spans', () => {
      const html = component.convertMarkdownToHtml('this is _italic_ and this is **bold**');
      expect(html).toContain('<em>italic</em>');
      expect(html).toContain('<strong>bold</strong>');
    });

    it('preserves underscores inside a fenced code block', () => {
      const html = component.convertMarkdownToHtml('```\nmin_unique_peptides = 2\n```');
      expect(html).toContain('<pre><code>');
      expect(html).toContain('min_unique_peptides = 2');
      expect(html).not.toContain('miniquepeptides');
    });
  });
});
