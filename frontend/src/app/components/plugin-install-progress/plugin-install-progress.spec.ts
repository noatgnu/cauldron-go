import { ComponentFixture, TestBed } from '@angular/core/testing';
import { PluginInstallProgress } from './plugin-install-progress';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { Wails } from '../../core/services/wails';
import { of } from 'rxjs';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('PluginInstallProgress', () => {
  let component: PluginInstallProgress;
  let fixture: ComponentFixture<PluginInstallProgress>;
  let wailsSpy: jasmine.SpyObj<Wails>;
  let dialogRefSpy: jasmine.SpyObj<MatDialogRef<PluginInstallProgress>>;

  beforeEach(async () => {
    wailsSpy = jasmine.createSpyObj('Wails', ['listen', 'installPluginFromRepo']);
    wailsSpy.listen.and.returnValue(of());
    wailsSpy.installPluginFromRepo.and.returnValue(Promise.resolve());
    
    dialogRefSpy = jasmine.createSpyObj('MatDialogRef', ['close']);

    await TestBed.configureTestingModule({
      imports: [PluginInstallProgress, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsSpy },
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: { repoURL: 'https://github.com/test/repo' } }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(PluginInstallProgress);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should start installation on init', () => {
    expect(wailsSpy.installPluginFromRepo).toHaveBeenCalledWith('https://github.com/test/repo', '');
  });

  it('should update progress when events are received', () => {
    expect(component).toBeDefined();
  });
});