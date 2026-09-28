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
import { NotificationService } from '../../core/services/notification.service';

@Component({
  selector: 'app-recipe-diagram',
  imports: [CommonModule],
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

  protected loading = signal(false);
  protected svg = signal<SafeHtml | null>(null);
  protected expandedStages = signal<Set<number>>(new Set());

  private renderCount = 0;
  private currentRenderId = '';
  private panZoom: SvgPanZoom.Instance | null = null;

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
      mermaid.initialize({ startOnLoad: false, theme: 'default', securityLevel: 'strict' });
      this.currentRenderId = `recipe-diagram-${this.renderCount++}`;
      const { svg } = await mermaid.render(this.currentRenderId, source);
      this.svg.set(this.sanitizer.bypassSecurityTrustHtml(svg));
      afterNextRender(() => void this.attachInteractivity(), { injector: this.injector });
    } catch (err) {
      this.notification.showError(`Failed to render diagram: ${err}`);
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
}
