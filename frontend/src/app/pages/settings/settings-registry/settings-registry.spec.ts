import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SettingsRegistry } from './settings-registry';

describe('SettingsRegistry', () => {
  let component: SettingsRegistry;
  let fixture: ComponentFixture<SettingsRegistry>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SettingsRegistry]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsRegistry);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
