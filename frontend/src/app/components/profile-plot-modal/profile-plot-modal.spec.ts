import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ProfilePlotModal } from './profile-plot-modal';

describe('ProfilePlotModal', () => {
  let component: ProfilePlotModal;
  let fixture: ComponentFixture<ProfilePlotModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ProfilePlotModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(ProfilePlotModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
