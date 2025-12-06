import { ComponentFixture, TestBed } from '@angular/core/testing';

import { UniprotModal } from './uniprot-modal';

describe('UniprotModal', () => {
  let component: UniprotModal;
  let fixture: ComponentFixture<UniprotModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [UniprotModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(UniprotModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
