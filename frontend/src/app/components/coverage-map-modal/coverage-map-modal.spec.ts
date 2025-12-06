import { ComponentFixture, TestBed } from '@angular/core/testing';

import { CoverageMapModal } from './coverage-map-modal';

describe('CoverageMapModal', () => {
  let component: CoverageMapModal;
  let fixture: ComponentFixture<CoverageMapModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CoverageMapModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(CoverageMapModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
