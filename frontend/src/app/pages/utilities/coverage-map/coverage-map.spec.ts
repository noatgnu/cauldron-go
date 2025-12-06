import { ComponentFixture, TestBed } from '@angular/core/testing';

import { CoverageMap } from './coverage-map';

describe('CoverageMap', () => {
  let component: CoverageMap;
  let fixture: ComponentFixture<CoverageMap>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CoverageMap]
    })
    .compileComponents();

    fixture = TestBed.createComponent(CoverageMap);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
