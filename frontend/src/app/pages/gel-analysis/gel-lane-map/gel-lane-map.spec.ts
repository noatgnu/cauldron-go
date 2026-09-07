import { ComponentFixture, TestBed } from '@angular/core/testing';
import { GelLaneMap } from './gel-lane-map';
import { GelLaneROI } from '../../../core/services/wails';

describe('GelLaneMap', () => {
  let component: GelLaneMap;
  let fixture: ComponentFixture<GelLaneMap>;

  function lane(overrides: Partial<GelLaneROI>): GelLaneROI {
    return { id: 'lane1', label: 'Lane 1', x: 0, y: 0, width: 10, height: 10, isMarker: false, ...overrides } as GelLaneROI;
  }

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [GelLaneMap]
    })
      .compileComponents();

    fixture = TestBed.createComponent(GelLaneMap);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('laneLeft returns the lane x position as a percentage of the image width', () => {
    component.imageWidth = 1000;
    expect(component.laneLeft(lane({ x: 250 }))).toBe(25);
  });

  it('laneLeft returns 0 when the image width is unknown', () => {
    component.imageWidth = 0;
    expect(component.laneLeft(lane({ x: 250 }))).toBe(0);
  });

  it('laneWidthPercent returns the lane width as a percentage of the image width', () => {
    component.imageWidth = 1000;
    expect(component.laneWidthPercent(lane({ width: 100 }))).toBe(10);
  });

  it('laneWidthPercent floors narrow lanes to a minimum visible width', () => {
    component.imageWidth = 10000;
    expect(component.laneWidthPercent(lane({ width: 1 }))).toBe(0.6);
  });

  it('select emits the laneSelected event with the lane id', () => {
    const emitted: string[] = [];
    component.laneSelected.subscribe(id => emitted.push(id));

    component.select('lane1');

    expect(emitted).toEqual(['lane1']);
  });
});
