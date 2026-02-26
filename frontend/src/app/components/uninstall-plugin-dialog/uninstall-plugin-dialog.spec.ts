import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { UninstallPluginDialog } from './uninstall-plugin-dialog';
import { vi } from 'vitest';

describe('UninstallPluginDialog', () => {
  let component: UninstallPluginDialog;
  let fixture: ComponentFixture<UninstallPluginDialog>;
  let dialogRefSpy: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [UninstallPluginDialog],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: { pluginName: 'Test', repoURL: 'test' } }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(UninstallPluginDialog);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
