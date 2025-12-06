import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Normalization } from './normalization';

describe('Normalization', () => {
  let component: Normalization;
  let fixture: ComponentFixture<Normalization>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Normalization]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Normalization);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
