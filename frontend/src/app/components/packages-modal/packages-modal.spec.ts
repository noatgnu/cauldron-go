import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PackagesModal } from './packages-modal';

describe('PackagesModal', () => {
  let component: PackagesModal;
  let fixture: ComponentFixture<PackagesModal>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PackagesModal]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PackagesModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
