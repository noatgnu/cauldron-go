import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SettingsGit } from './settings-git';

describe('SettingsGit', () => {
  let component: SettingsGit;
  let fixture: ComponentFixture<SettingsGit>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SettingsGit]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsGit);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
