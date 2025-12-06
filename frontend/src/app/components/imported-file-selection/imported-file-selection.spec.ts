import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ImportedFileSelection } from './imported-file-selection';

describe('ImportedFileSelection', () => {
  let component: ImportedFileSelection;
  let fixture: ComponentFixture<ImportedFileSelection>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ImportedFileSelection]
    })
    .compileComponents();

    fixture = TestBed.createComponent(ImportedFileSelection);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
