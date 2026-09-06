"""Automatic lane detection for Cauldron's Gel Analysis feature.

Invoked by backend/services/gel_analysis_service.go's RunAutoDetect via the same ScriptExecutor
plugins use, under the synthetic environment-binding key "gel-analysis". Reads a 16-bit grayscale
PNG or a TIFF/other image, and writes lanes.json with detected lane rectangles and a diagnostic
deskew-angle estimate. Background subtraction is internal only; the original image is never
modified or returned.
"""

import argparse
import json
import os
import sys

import numpy as np
from PIL import Image
from scipy import ndimage
from scipy.signal import find_peaks

try:
    import tifffile
    HAVE_TIFFFILE = True
except ImportError:
    HAVE_TIFFFILE = False


def load_image_array(path: str) -> np.ndarray:
    """Loads an image as a 2D float64 grayscale array, preferring tifffile for .tif/.tiff (better
    bit-depth/compression coverage than PIL) and falling back to PIL for everything else."""
    ext = os.path.splitext(path)[1].lower()

    if ext in (".tif", ".tiff") and HAVE_TIFFFILE:
        arr = tifffile.imread(path)
    else:
        img = Image.open(path)
        arr = np.array(img)

    if arr.ndim == 3:
        # Collapse to grayscale via luminance weighting, matching the Go side's conversion.
        weights = np.array([0.299, 0.587, 0.114])
        arr = (arr[..., :3].astype(np.float64) * weights).sum(axis=-1)

    return arr.astype(np.float64)


def find_deskew_angle(image: np.ndarray, max_angle: float = 5.0, step: float = 0.5) -> float:
    """Searches small rotation angles and picks the one that maximizes the variance of the
    column-sum projection: gel lanes run vertically, so when the image is properly aligned,
    columns inside a lane sum to consistently high values and gaps between lanes sum to
    consistently low values, producing a high-variance profile. Skew smears that contrast out."""
    best_angle = 0.0
    best_variance = -np.inf

    # Downsample for the angle search; the chosen angle is applied to the full-resolution image.
    scale = min(1.0, 512.0 / max(image.shape))
    small = ndimage.zoom(image, scale, order=1) if scale < 1.0 else image

    angle = -max_angle
    while angle <= max_angle + 1e-9:
        rotated = ndimage.rotate(small, angle, reshape=False, order=1, mode="nearest")
        profile = rotated.sum(axis=0)
        variance = float(np.var(profile))
        if variance > best_variance:
            best_variance = variance
            best_angle = angle
        angle += step

    return best_angle


def subtract_background(image: np.ndarray, polarity: str = "dark-bands", kernel_fraction: float = 0.05) -> np.ndarray:
    """Estimates a smooth background via large-kernel morphological filtering and subtracts it, leaving positive band signal and ~0 elsewhere."""
    kernel_size = max(3, int(min(image.shape) * kernel_fraction))
    if kernel_size % 2 == 0:
        kernel_size += 1

    if polarity == "light-bands":
        # Bands are brighter than background: opening erases small bright features -> background estimate.
        background = ndimage.grey_opening(image, size=(kernel_size, kernel_size))
        corrected = image - background
    else:
        # Bands are darker than background (default): closing erases small dark features -> background estimate.
        background = ndimage.grey_closing(image, size=(kernel_size, kernel_size))
        corrected = background - image

    corrected[corrected < 0] = 0
    return corrected


def segment_lane_boundaries(smoothed: np.ndarray, peaks: list[int], width: int) -> list[tuple[int, int]]:
    """Partitions the column axis into one non-overlapping [lo,hi) region per peak, cut at the
    valley (local minimum) between each pair of neighboring peaks. A valley cut is a single shared
    boundary point, so no two regions can overlap."""
    sorted_peaks = sorted(peaks)
    boundaries = [0]
    for a, b in zip(sorted_peaks, sorted_peaks[1:]):
        valley = a + int(np.argmin(smoothed[a:b + 1]))
        boundaries.append(valley)
    boundaries.append(width)
    return [(boundaries[i], boundaries[i + 1]) for i in range(len(sorted_peaks))], sorted_peaks


def tighten_lane_width(smoothed: np.ndarray, peak: int, lo: int, hi: int, rel_height: float = 0.9) -> tuple[int, int]:
    """Shrinks a peak's safe [lo,hi) region inward until the profile drops most of the way to the
    local floor on each side, clamped to [lo,hi)."""
    peak_val = smoothed[peak]
    left_floor = smoothed[lo:peak + 1].min()
    right_floor = smoothed[peak:hi + 1].min()
    left_thresh = peak_val - rel_height * (peak_val - left_floor)
    right_thresh = peak_val - rel_height * (peak_val - right_floor)

    x0 = lo
    for x in range(peak, lo - 1, -1):
        if smoothed[x] <= left_thresh:
            x0 = x
            break
    x1 = hi
    for x in range(peak, hi):
        if smoothed[x] <= right_thresh:
            x1 = x
            break
    return x0, x1


def detect_lanes(image: np.ndarray, min_prominence: float = 0.05) -> list[dict]:
    """Detects lane column-ranges from the column-sum profile via find_peaks, valley-cut
    segmentation, and width tightening."""
    height, width = image.shape
    profile = image.sum(axis=0)

    smooth_window = max(3, width // 200)
    kernel = np.ones(smooth_window) / smooth_window
    smoothed = np.convolve(profile, kernel, mode="same")

    min_distance = max(1, width // 50)
    prominence = (smoothed.max() - smoothed.min()) * min_prominence

    peaks, _ = find_peaks(smoothed, distance=min_distance, prominence=prominence)
    if len(peaks) == 0:
        return []

    safe_regions, sorted_peaks = segment_lane_boundaries(smoothed, peaks.tolist(), width)

    candidates = []
    for peak, (lo, hi) in zip(sorted_peaks, safe_regions):
        x0, x1 = tighten_lane_width(smoothed, peak, lo, hi)
        if x1 > x0:
            candidates.append((x0, x1))
    if not candidates:
        return []

    # Row variation filters flat artifacts; not filtering by width since width reflects sharpness, not true size.
    min_row_cv = 0.6

    lanes = []
    for x0, x1 in candidates:
        row_profile = image[:, x0:x1].sum(axis=1)
        row_mean = row_profile.mean()
        row_cv = (row_profile.std() / row_mean) if row_mean != 0 else 0.0
        if row_cv < min_row_cv:
            continue
        lanes.append({
            "x": float(x0),
            "y": 0.0,
            "width": float(x1 - x0),
            "height": float(height),
            "index": len(lanes),
        })

    return lanes


def find_raw_peaks(image: np.ndarray, width: int, min_prominence: float = 0.03) -> tuple[list[int], np.ndarray]:
    """Locates lane-peak candidates permissively. Used as evidence for anchor-guided slot
    assignment, not as a final result."""
    profile = image.sum(axis=0)
    smooth_window = max(3, width // 200)
    kernel = np.ones(smooth_window) / smooth_window
    smoothed = np.convolve(profile, kernel, mode="same")

    min_distance = max(1, width // 100)
    prominence = (smoothed.max() - smoothed.min()) * min_prominence
    peaks, _ = find_peaks(smoothed, distance=min_distance, prominence=prominence)
    return sorted(peaks.tolist()), smoothed


def fit_pitch_from_anchors(anchors: list[tuple[int, float, float]], expected_count: int, boundary_x0: float | None, boundary_x1: float | None) -> tuple[float, float] | None:
    """Returns (pitch, intercept) so slot i's center is intercept + i*pitch, or None if there isn't
    enough information. With 2+ anchors, pitch is fit via least squares. With exactly one anchor,
    pitch falls back to (boundary width - anchor width) / (expected_count - 1)."""
    if len(anchors) >= 2:
        indices = np.array([a[0] for a in anchors], dtype=np.float64)
        positions = np.array([a[1] for a in anchors], dtype=np.float64)
        design = np.vstack([indices, np.ones_like(indices)]).T
        pitch, intercept = np.linalg.lstsq(design, positions, rcond=None)[0]
        return float(pitch), float(intercept)

    if len(anchors) == 1 and boundary_x0 is not None and boundary_x1 is not None:
        anchor_index, anchor_position, anchor_width = anchors[0]
        pitch = (boundary_x1 - boundary_x0 - anchor_width) / max(1, expected_count - 1)
        return pitch, anchor_position - anchor_index * pitch

    return None


def detect_lanes_with_anchors(
    image: np.ndarray,
    expected_count: int,
    anchors: list[tuple[int, float, float]],
    boundary_x0: float | None,
    boundary_x1: float | None,
    min_row_cv: float = 0.6,
) -> list[dict] | None:
    """Detects up to `expected_count` lanes from known lane positions and indices (anchors)
    instead of inferring the count from the profile. Every slot gets an exclusive half-pitch-wide
    zone, so lanes never overlap. An anchor's own slot is trusted as-is. Other slots take the
    nearest raw peak within half a pitch, or the strongest point in the zone if none is nearby. A
    slot whose row variation is too flat is omitted from the output but still counts toward the
    pitch geometry. Returns None if there isn't enough anchor/boundary information to fit a
    pitch."""
    height, width = image.shape
    fit = fit_pitch_from_anchors(anchors, expected_count, boundary_x0, boundary_x1)
    if fit is None:
        return None
    pitch, intercept = fit
    if pitch <= 0:
        return None

    peaks, smoothed = find_raw_peaks(image, width)
    slots = [intercept + i * pitch for i in range(expected_count)]
    anchor_by_index = {a[0]: a for a in anchors}

    half = pitch / 2
    min_width = max(3, pitch * 0.3)
    assigned: dict[int, list[int]] = {i: [] for i in range(expected_count)}
    for p in peaks:
        distances = [abs(p - s) for s in slots]
        best = int(np.argmin(distances))
        if distances[best] <= half:
            assigned[best].append(p)

    lanes = []
    for i, slot in enumerate(slots):
        if i in anchor_by_index:
            continue  # already have this lane; don't echo it back as a "new" detection

        zone_lo = max(0, int(slot - half))
        zone_hi = min(width, int(slot + half))
        if zone_hi <= zone_lo:
            continue

        members = assigned[i]
        rep = max(members, key=lambda p: smoothed[p]) if members else zone_lo + int(np.argmax(smoothed[zone_lo:zone_hi]))
        x0, x1 = tighten_lane_width(smoothed, rep, zone_lo, zone_hi)

        if x1 - x0 < min_width:
            deficit = min_width - (x1 - x0)
            room_left = x0 - zone_lo
            room_right = zone_hi - x1
            take_left = min(room_left, deficit / 2)
            take_right = min(room_right, deficit - take_left)
            take_left = min(room_left, deficit - take_right)
            x0 = int(x0 - take_left)
            x1 = int(x1 + take_right)

        row_profile = image[:, x0:x1].sum(axis=1)
        row_mean = row_profile.mean()
        row_cv = (row_profile.std() / row_mean) if row_mean != 0 else 0.0
        if row_cv < min_row_cv:
            continue  # likely an intentionally empty spacer well, not a real sample lane

        lanes.append({"x": float(x0), "y": 0.0, "width": float(x1 - x0), "height": float(height), "index": i})

    return lanes


def main() -> int:
    parser = argparse.ArgumentParser(description="Gel Analysis auto-detect (lanes, deskew, background subtraction)")
    parser.add_argument("--input", required=True, help="Path to the input gel image (PNG or TIFF)")
    parser.add_argument("--output", required=True, help="Output directory for lanes.json")
    parser.add_argument("--polarity", default="dark-bands", choices=["dark-bands", "light-bands"], help="Band polarity relative to background")
    parser.add_argument("--min-prominence", type=float, default=0.05, help="Minimum lane-peak prominence as a fraction of the profile's range (0-1)")
    parser.add_argument("--expected-lane-count", type=int, default=0, help="Total number of physical lane slots in the gel (including any intentionally empty spacer wells); enables anchor-guided detection when combined with --anchors-json")
    parser.add_argument("--anchors-json", default=None, help='JSON list of known lanes, e.g. [{"index":0,"position":476,"width":72}]')
    parser.add_argument("--boundary-x0", type=float, default=None, help="Left edge (pixels) of the gel's lane region")
    parser.add_argument("--boundary-x1", type=float, default=None, help="Right edge (pixels) of the gel's lane region")
    args = parser.parse_args()

    os.makedirs(args.output, exist_ok=True)

    print(f"Loading image: {args.input}")
    image = load_image_array(args.input)
    print(f"Image shape: {image.shape}")

    print("Searching for deskew angle...")
    angle = find_deskew_angle(image)
    print(f"Estimated deskew angle: {angle:.2f} degrees (reported only, not applied)")

    print("Subtracting background (for lane-finding only, not returned)...")
    corrected = subtract_background(image, args.polarity)

    lanes = None
    if args.expected_lane_count > 0 and args.anchors_json:
        anchors = [(a["index"], a["position"], a["width"]) for a in json.loads(args.anchors_json)]
        print(f"Detecting lanes (anchor-guided, expecting {args.expected_lane_count}, {len(anchors)} anchor(s))...")
        lanes = detect_lanes_with_anchors(corrected, args.expected_lane_count, anchors, args.boundary_x0, args.boundary_x1)

    if lanes is None:
        print("Detecting lanes...")
        lanes = detect_lanes(corrected, args.min_prominence)
    print(f"Detected {len(lanes)} lane(s)")

    result = {
        "deskewAngle": angle,
        "lanes": lanes,
    }
    with open(os.path.join(args.output, "lanes.json"), "w") as f:
        json.dump(result, f, indent=2)

    print("Auto-detect complete.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
