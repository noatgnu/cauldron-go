import { ComponentFixture, TestBed } from '@angular/core/testing';

import { VennDiagram } from './venn-diagram';

describe('VennDiagram', () => {
  let component: VennDiagram;
  let fixture: ComponentFixture<VennDiagram>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [VennDiagram]
    })
    .compileComponents();

    fixture = TestBed.createComponent(VennDiagram);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
