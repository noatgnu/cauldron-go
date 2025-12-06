import { ComponentFixture, TestBed } from '@angular/core/testing';

import { JobRemovalModal } from './job-removal-modal';

describe('JobRemovalModal', () => {
  let component: JobRemovalModal;
  let fixture: ComponentFixture<JobRemovalModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [JobRemovalModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(JobRemovalModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
