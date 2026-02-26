import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { CoverageMapModal } from './coverage-map-modal';
import { Wails } from '../../core/services/wails';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('CoverageMapModal', () => {
  let component: CoverageMapModal;
  let fixture: ComponentFixture<CoverageMapModal>;
  let dialogRefSpy: any;
  let wailsMock: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };
    wailsMock = {
      readFile: vi.fn().mockResolvedValue('')
    };

    await TestBed.configureTestingModule({
      imports: [CoverageMapModal, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: { jobId: 'test' } },
        { provide: Wails, useValue: wailsMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(CoverageMapModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
