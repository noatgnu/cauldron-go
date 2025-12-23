import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SettingsR } from './settings-r';

describe('SettingsR', () => {
  let component: SettingsR;
  let fixture: ComponentFixture<SettingsR>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SettingsR]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsR);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
