import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SampleAnnotation } from './sample-annotation';

describe('SampleAnnotation', () => {
  let component: SampleAnnotation;
  let fixture: ComponentFixture<SampleAnnotation>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SampleAnnotation]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SampleAnnotation);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
