import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SettingsPython } from './settings-python';

describe('SettingsPython', () => {
  let component: SettingsPython;
  let fixture: ComponentFixture<SettingsPython>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SettingsPython]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsPython);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
