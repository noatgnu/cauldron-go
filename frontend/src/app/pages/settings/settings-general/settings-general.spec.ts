import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SettingsGeneral } from './settings-general';

describe('SettingsGeneral', () => {
  let component: SettingsGeneral;
  let fixture: ComponentFixture<SettingsGeneral>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SettingsGeneral]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsGeneral);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
