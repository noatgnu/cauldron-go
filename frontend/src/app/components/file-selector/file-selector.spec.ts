import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FileSelector } from './file-selector';

describe('FileSelector', () => {
  let component: FileSelector;
  let fixture: ComponentFixture<FileSelector>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FileSelector]
    })
    .compileComponents();

    fixture = TestBed.createComponent(FileSelector);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
