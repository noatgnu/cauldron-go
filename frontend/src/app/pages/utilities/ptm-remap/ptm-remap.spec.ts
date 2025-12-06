import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PtmRemap } from './ptm-remap';

describe('PtmRemap', () => {
  let component: PtmRemap;
  let fixture: ComponentFixture<PtmRemap>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PtmRemap]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PtmRemap);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
