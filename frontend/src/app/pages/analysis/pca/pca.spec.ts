import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Pca } from './pca';

describe('Pca', () => {
  let component: Pca;
  let fixture: ComponentFixture<Pca>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Pca]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Pca);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
