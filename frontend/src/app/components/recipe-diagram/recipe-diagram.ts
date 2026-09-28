import {
  Component,
  Input,
  OnChanges,
  OnDestroy,
  SimpleChanges,
  signal,
  inject,
  ElementRef,
  Injector,
  afterNextRender,
  ChangeDetectionStrategy
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { NotificationService } from '../../core/services/notification.service';
import { Wails } from '../../core/services/wails';

@Component({
  selector: 'app-recipe-diagram',
  imports: [CommonModule, MatButtonModule, MatIconModule, MatTooltipModule],
  templateUrl: './recipe-diagram.html',
  styleUrl: './recipe-diagram.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class RecipeDiagram implements OnChanges, OnDestroy {
  /** Identifies the diagram's subject; a change resets expansion and re-renders. */
  @Input({ required: true }) subjectId!: string;
  @Input({ required: true }) generate!: (expandedStages: number[]) => Promise<string>;

  private notification = inject(NotificationService);
  private sanitizer = inject(DomSanitizer);
  private elementRef = inject<ElementRef<HTMLElement>>(ElementRef);
  private injector = inject(Injector);
  private wails = inject(Wails);

  protected loading = signal(false);
  protected svg = signal<SafeHtml | null>(null);
  protected expandedStages = signal<Set<number>>(new Set());

  private renderCount = 0;
  private currentRenderId = '';
  private panZoom: SvgPanZoom.Instance | null = null;
  private rawSvg: string | null = null;

  async ngOnChanges(changes: SimpleChanges) {
    if (changes['subjectId']) {
      this.expandedStages.set(new Set());
      await this.render();
    }
  }

  ngOnDestroy() {
    this.panZoom?.destroy();
  }

  private async render() {
    if (!this.subjectId) return;
    this.loading.set(true);
    this.panZoom?.destroy();
    this.panZoom = null;
    try {
      const source = await this.generate(Array.from(this.expandedStages()));
      const mermaid = (await import('mermaid')).default;
      mermaid.initialize({
        startOnLoad: false,
        theme: 'default',
        securityLevel: 'strict',
        htmlLabels: false,
        flowchart: { htmlLabels: false }
      });
      this.currentRenderId = `recipe-diagram-${this.renderCount++}`;
      const { svg } = await mermaid.render(this.currentRenderId, source);
      this.rawSvg = svg;
      this.svg.set(this.sanitizer.bypassSecurityTrustHtml(svg));
      afterNextRender(() => void this.attachInteractivity(), { injector: this.injector });
    } catch (err) {
      this.notification.showError(`Failed to render diagram: ${err}`);
      this.rawSvg = null;
      this.svg.set(null);
    } finally {
      this.loading.set(false);
    }
  }

  private async attachInteractivity() {
    const container = this.elementRef.nativeElement.querySelector<SVGSVGElement>('.diagram-canvas svg');
    if (!container) return;

    container.addEventListener('click', (event: MouseEvent) => this.onDiagramClick(event));

    const svgPanZoomModule: any = await import('svg-pan-zoom');
    const svgPanZoom = svgPanZoomModule.default ?? svgPanZoomModule;
    this.panZoom = svgPanZoom(container, {
      zoomEnabled: true,
      controlIconsEnabled: true,
      fit: true,
      center: true,
      minZoom: 0.2,
      maxZoom: 8
    });
  }

  private onDiagramClick(event: MouseEvent) {
    const el = (event.target as Element).closest('[id]');
    if (!el) return;
    const stageIndex = this.stageIndexForToggle(el.id);
    if (stageIndex === null) return;
    this.toggleStage(stageIndex);
  }

  private stageIndexForToggle(domId: string): number | null {
    let remainder = domId.startsWith(this.currentRenderId + '-') ? domId.slice(this.currentRenderId.length + 1) : domId;
    if (remainder.startsWith('flowchart-')) {
      remainder = remainder.slice('flowchart-'.length);
    }
    const match = /^S(\d+)(?:_group)?(?:-\d+)?$/.exec(remainder);
    return match ? Number(match[1]) : null;
  }

  private toggleStage(index: number) {
    this.expandedStages.update(current => {
      const next = new Set(current);
      if (next.has(index)) {
        next.delete(index);
      } else {
        next.add(index);
      }
      return next;
    });
    void this.render();
  }

  protected async exportSvg() {
    if (!this.rawSvg) return;
    try {
      const defaultName = `recipe-diagram-${this.subjectId}.svg`;
      const path = await this.wails.saveFileDialog('Export Diagram as SVG', defaultName);
      if (!path) return;
      await this.wails.exportDiagramSVG(path, this.rawSvg);
      this.notification.showSuccess('Diagram exported.');
    } catch (err) {
      this.notification.showError(`Failed to export diagram: ${err}`);
    }
  }

  protected exportPdf() {
    if (!this.rawSvg) return;
    printDiagram(this.rawSvg);
  }
}

/**
 * Prints just the diagram by temporarily injecting it as a direct child of
 * <body> and hiding every other direct child for the duration of the print,
 * via the global `body.printing-diagram` rule in styles.scss. A popup-window
 * print (the more common technique) doesn't work here: window.open() is
 * blocked inside the desktop app's WebKitGTK webview, but window.print() on
 * the current window works fine and opens the real native print dialog
 * (which offers "Print to File" / Save as PDF on Linux).
 */
export function printDiagram(svg: string): void {
  const target = document.createElement('div');
  target.className = 'print-diagram-target';
  target.innerHTML = svg;
  document.body.appendChild(target);
  document.body.classList.add('printing-diagram');

  const cleanup = () => {
    document.body.classList.remove('printing-diagram');
    target.remove();
    window.removeEventListener('afterprint', cleanup);
  };
  window.addEventListener('afterprint', cleanup);

  window.print();
  cleanup();
}
