import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PluginRegistryDetail } from './plugin-registry-detail';

describe('PluginRegistryDetail', () => {
  let component: PluginRegistryDetail;
  let fixture: ComponentFixture<PluginRegistryDetail>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PluginRegistryDetail]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PluginRegistryDetail);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
