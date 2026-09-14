import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { QuickNav } from './quick-nav';
import { Wails } from '../../core/services/wails';

describe('QuickNav', () => {
  let component: QuickNav;
  let fixture: ComponentFixture<QuickNav>;
  let wailsMock: any;

  async function create() {
    await TestBed.configureTestingModule({
      imports: [QuickNav],
      providers: [
        provideRouter([]),
        { provide: Wails, useValue: wailsMock }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(QuickNav);
    component = fixture.componentInstance;
    await fixture.whenStable();
    fixture.detectChanges();
  }

  it('is hidden in desktop mode', async () => {
    wailsMock = { getRuntimeCapabilities: vi.fn().mockResolvedValue({ serverMode: false, nativeFileAccess: true, maxUploadChunkBytes: 8388608 }) };
    await create();

    expect((component as any).visible()).toBe(false);
    expect(fixture.nativeElement.querySelector('.quick-nav-fab')).toBeNull();
  });

  it('is visible in server mode', async () => {
    wailsMock = { getRuntimeCapabilities: vi.fn().mockResolvedValue({ serverMode: true, nativeFileAccess: false, maxUploadChunkBytes: 8388608 }) };
    await create();

    expect((component as any).visible()).toBe(true);
    expect(fixture.nativeElement.querySelector('.quick-nav-fab')).not.toBeNull();
  });

  it('stays hidden if the capabilities check fails', async () => {
    wailsMock = { getRuntimeCapabilities: vi.fn().mockRejectedValue(new Error('not wails')) };
    await create();

    expect((component as any).visible()).toBe(false);
  });

  it('navigates via the router when a menu item is clicked', async () => {
    wailsMock = { getRuntimeCapabilities: vi.fn().mockResolvedValue({ serverMode: true, nativeFileAccess: false, maxUploadChunkBytes: 8388608 }) };
    await create();

    const router = TestBed.inject(Router);
    const navigateSpy = vi.spyOn(router, 'navigate').mockResolvedValue(true);

    component.navigate('/jobs');

    expect(navigateSpy).toHaveBeenCalledWith(['/jobs']);
  });
});
