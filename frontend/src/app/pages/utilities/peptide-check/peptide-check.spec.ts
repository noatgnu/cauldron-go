import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PeptideCheck } from './peptide-check';

describe('PeptideCheck', () => {
  let component: PeptideCheck;
  let fixture: ComponentFixture<PeptideCheck>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PeptideCheck]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PeptideCheck);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
