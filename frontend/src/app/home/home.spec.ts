import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { vi } from 'vitest';

import { Home } from './home';
import { Wails } from '../core/services/wails';

describe('Home', () => {
  let component: Home;
  let fixture: ComponentFixture<Home>;
  let wailsMock: any;

  beforeEach(async () => {
    wailsMock = {
      isWails: false,
      jobUpdate: () => null,
      getSettings: vi.fn().mockResolvedValue({}),
      logToFile: vi.fn().mockResolvedValue(undefined),
      waitForBackend: vi.fn().mockResolvedValue(undefined)
    };

    await TestBed.configureTestingModule({
      imports: [Home],
      providers: [
        { provide: Wails, useValue: wailsMock }
      ]
    })
    .compileComponents();
  });

  function createComponent() {
    fixture = TestBed.createComponent(Home);
    component = fixture.componentInstance;
  }

  it('should create', async () => {
    createComponent();
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });

  it('defaults debug mode to off', async () => {
    createComponent();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(component['debugModeEnabled']()).toBe(false);
  });

  it('enables debug mode when the setting is on', async () => {
    wailsMock.getSettings.mockResolvedValue({ debugMode: true });
    createComponent();
    fixture.detectChanges();
    await fixture.whenStable();
    await new Promise(resolve => setTimeout(resolve, 0));
    expect(component['debugModeEnabled']()).toBe(true);
  });

  it('does not render the debug card by default', async () => {
    createComponent();
    fixture.detectChanges();
    await fixture.whenStable();
    const el: HTMLElement = fixture.nativeElement;
    expect(el.querySelector('.debug-card')).toBeNull();
  });

  it('renders the debug card when debug mode is enabled', async () => {
    wailsMock.getSettings.mockResolvedValue({ debugMode: true });
    createComponent();
    fixture.detectChanges();
    await fixture.whenStable();
    await new Promise(resolve => setTimeout(resolve, 0));
    fixture.detectChanges();
    const el: HTMLElement = fixture.nativeElement;
    expect(el.querySelector('.debug-card')).not.toBeNull();
  });

  it('does not mark itself ready before initialization has actually run', async () => {
    let resolveBackend!: () => void;
    wailsMock.waitForBackend = vi.fn().mockReturnValue(new Promise<void>(resolve => { resolveBackend = resolve; }));
    createComponent();
    fixture.detectChanges();

    expect((component as any).allLoaded()).toBe(false);

    resolveBackend();
    await fixture.whenStable();
    fixture.detectChanges();

    expect((component as any).allLoaded()).toBe(true);
  });

  it('does not loop forever reconciling the jobs list when a job update is already pending on construction', async () => {
    const job = { id: 'job-1', name: 'Test Job', type: 'pca-analysis', status: 'completed', createdAt: Date.now() };
    wailsMock.jobUpdate = signal(job);
    createComponent();
    await fixture.whenStable();

    const jobs = (component as any).jobs();
    expect(jobs).toHaveLength(1);
    expect(jobs[0].id).toBe(job.id);
  });
});
