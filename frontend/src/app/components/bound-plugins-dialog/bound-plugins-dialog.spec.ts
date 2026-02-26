import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { BoundPluginsDialogComponent } from './bound-plugins-dialog';

describe('BoundPluginsDialogComponent', () => {
  let component: BoundPluginsDialogComponent;
  let fixture: ComponentFixture<BoundPluginsDialogComponent>;
  let mockDialogRef: { close: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    mockDialogRef = { close: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [BoundPluginsDialogComponent],
      providers: [
        { provide: MatDialogRef, useValue: mockDialogRef },
        { provide: MAT_DIALOG_DATA, useValue: { plugins: [] } }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(BoundPluginsDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should close dialog', () => {
    component.close();
    expect(mockDialogRef.close).toHaveBeenCalled();
  });
});
