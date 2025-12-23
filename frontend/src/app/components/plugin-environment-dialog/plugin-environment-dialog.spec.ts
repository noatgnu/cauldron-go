import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PluginEnvironmentDialog } from './plugin-environment-dialog';

describe('PluginEnvironmentDialog', () => {
  let component: PluginEnvironmentDialog;
  let fixture: ComponentFixture<PluginEnvironmentDialog>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PluginEnvironmentDialog]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PluginEnvironmentDialog);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
