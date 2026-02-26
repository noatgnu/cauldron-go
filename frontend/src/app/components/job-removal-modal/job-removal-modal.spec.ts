import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { JobRemovalModal } from './job-removal-modal';
import { vi } from 'vitest';

describe('JobRemovalModal', () => {
  let component: JobRemovalModal;
  let fixture: ComponentFixture<JobRemovalModal>;
  let dialogRefSpy: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [JobRemovalModal],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: {} }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(JobRemovalModal);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should close with false on close()', () => {
    component.close();
    expect(dialogRefSpy.close).toHaveBeenCalledWith(false);
  });

  it('should close with true on remove()', () => {
    component.remove();
    expect(dialogRefSpy.close).toHaveBeenCalledWith(true);
  });
});
