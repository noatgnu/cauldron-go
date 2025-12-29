import { ComponentFixture, TestBed } from '@angular/core/testing';

import { UninstallPluginDialog } from './uninstall-plugin-dialog';

describe('UninstallPluginDialog', () => {
  let component: UninstallPluginDialog;
  let fixture: ComponentFixture<UninstallPluginDialog>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [UninstallPluginDialog]
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
