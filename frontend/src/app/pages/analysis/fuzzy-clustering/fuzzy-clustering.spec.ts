import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FuzzyClustering } from './fuzzy-clustering';

describe('FuzzyClustering', () => {
  let component: FuzzyClustering;
  let fixture: ComponentFixture<FuzzyClustering>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FuzzyClustering]
    })
    .compileComponents();

    fixture = TestBed.createComponent(FuzzyClustering);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
