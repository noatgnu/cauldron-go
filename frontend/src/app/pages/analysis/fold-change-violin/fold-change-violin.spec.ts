import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FoldChangeViolin } from './fold-change-violin';

describe('FoldChangeViolin', () => {
  let component: FoldChangeViolin;
  let fixture: ComponentFixture<FoldChangeViolin>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FoldChangeViolin]
    })
    .compileComponents();

    fixture = TestBed.createComponent(FoldChangeViolin);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
