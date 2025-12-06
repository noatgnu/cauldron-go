import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Maxlfq } from './maxlfq';

describe('Maxlfq', () => {
  let component: Maxlfq;
  let fixture: ComponentFixture<Maxlfq>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Maxlfq]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Maxlfq);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
