import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ImportDialog } from './import-dialog';

describe('ImportDialog', () => {
  let component: ImportDialog;
  let fixture: ComponentFixture<ImportDialog>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ImportDialog]
    })
    .compileComponents();

    fixture = TestBed.createComponent(ImportDialog);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
