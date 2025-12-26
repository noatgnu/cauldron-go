import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SettingsEnv } from './settings-env';

describe('SettingsEnv', () => {
  let component: SettingsEnv;
  let fixture: ComponentFixture<SettingsEnv>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SettingsEnv]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsEnv);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
