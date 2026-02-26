import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { PromptDialogComponent, PromptDialogData } from './prompt-dialog';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('PromptDialogComponent', () => {
  let component: PromptDialogComponent;
  let fixture: ComponentFixture<PromptDialogComponent>;
  let mockDialogRef: any;
  let mockData: PromptDialogData;

  beforeEach(async () => {
    mockDialogRef = {
      close: vi.fn()
    };

    mockData = {
      title: 'Test Title',
      message: 'Test Message',
      label: 'Test Label',
      value: 'Initial Value'
    };

    await TestBed.configureTestingModule({
      imports: [PromptDialogComponent, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: mockDialogRef },
        { provide: MAT_DIALOG_DATA, useValue: mockData }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(PromptDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize with provided data', () => {
    expect(component.inputValue).toBe('Initial Value');
    expect(component.data.title).toBe('Test Title');
  });

  it('should close with input value when onConfirm is called', () => {
    component.inputValue = 'New Value';
    component.onConfirm();
    expect(mockDialogRef.close).toHaveBeenCalledWith('New Value');
  });

  it('should close with null when onCancel is called', () => {
    component.onCancel();
    expect(mockDialogRef.close).toHaveBeenCalledWith(null);
  });
});
