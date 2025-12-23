import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PluginRegistry } from './plugin-registry';

describe('PluginRegistry', () => {
  let component: PluginRegistry;
  let fixture: ComponentFixture<PluginRegistry>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PluginRegistry]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PluginRegistry);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
