import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ConfirmPluginInstallDialog } from './confirm-plugin-install-dialog';

describe('ConfirmPluginInstallDialog', () => {
  let component: ConfirmPluginInstallDialog;
  let fixture: ComponentFixture<ConfirmPluginInstallDialog>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ConfirmPluginInstallDialog]
    })
    .compileComponents();

    fixture = TestBed.createComponent(ConfirmPluginInstallDialog);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
