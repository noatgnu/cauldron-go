import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SettingsAccessibility } from './settings-accessibility';

describe('SettingsAccessibility', () => {
  let component: SettingsAccessibility;
  let fixture: ComponentFixture<SettingsAccessibility>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SettingsAccessibility]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsAccessibility);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
