import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Phate } from './phate';

describe('Phate', () => {
  let component: Phate;
  let fixture: ComponentFixture<Phate>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Phate]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Phate);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
