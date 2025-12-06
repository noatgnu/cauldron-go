import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Imputation } from './imputation';

describe('Imputation', () => {
  let component: Imputation;
  let fixture: ComponentFixture<Imputation>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Imputation]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Imputation);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
