import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Limma } from './limma';

describe('Limma', () => {
  let component: Limma;
  let fixture: ComponentFixture<Limma>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Limma]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Limma);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
