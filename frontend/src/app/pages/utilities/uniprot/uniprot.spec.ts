import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Uniprot } from './uniprot';

describe('Uniprot', () => {
  let component: Uniprot;
  let fixture: ComponentFixture<Uniprot>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Uniprot]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Uniprot);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
