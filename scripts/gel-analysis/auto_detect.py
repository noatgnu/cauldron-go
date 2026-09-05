"""Automatic lane detection, deskew, and background subtraction for Cauldron's Gel Analysis
feature.

Invoked by backend/services/gel_analysis_service.go's RunAutoDetect via the same ScriptExecutor
plugins use, under the synthetic environment-binding key "gel-analysis" (see
plugin_environment_dialog's setup wizard). Reads a 16-bit grayscale PNG (written by the Go side
via EncodeGray16PNG, which loses no precision) or a TIFF/other image directly, and writes
lanes.json describing detected lane rectangles, the deskew angle applied, and an optional
background-subtracted image the Go side reloads as its new working buffer.
"""

import argparse
import json
import os
import sys

import numpy as np
from PIL import Image
from scipy import ndimage
from scipy.signal import find_peaks, peak_widths

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
        # Collapse to grayscale via standard luminance weighting, matching the Go side's
        # image/color conversion for any non-grayscale source.
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


def subtract_background(image: np.ndarray, kernel_fraction: float = 0.05) -> np.ndarray:
    """Estimates a smooth background via large-kernel grayscale morphological opening and
    subtracts it, clipping negative results to zero. This removes slow illumination gradients
    across a scan while leaving sharp bands intact."""
    kernel_size = max(3, int(min(image.shape) * kernel_fraction))
    if kernel_size % 2 == 0:
        kernel_size += 1

    background = ndimage.grey_opening(image, size=(kernel_size, kernel_size))
    corrected = image - background
    corrected[corrected < 0] = 0
    return corrected


def detect_lanes(image: np.ndarray) -> list[dict]:
    """Detects lane column-ranges from the image's column-sum profile via scipy.signal.find_peaks
    (one peak per lane) and peak_widths (relative-height boundary crossings), mirroring the
    Go side's own band-finding approach (peak -> prominence -> boundaries) but for lanes instead
    of bands, and using scipy's battle-tested implementation since this runs in Python anyway."""
    height, width = image.shape
    profile = image.sum(axis=0)

    smooth_window = max(3, width // 200)
    kernel = np.ones(smooth_window) / smooth_window
    smoothed = np.convolve(profile, kernel, mode="same")

    min_distance = max(1, width // 50)
    prominence = (smoothed.max() - smoothed.min()) * 0.05

    peaks, _ = find_peaks(smoothed, distance=min_distance, prominence=prominence)
    if len(peaks) == 0:
        return []

    widths_result = peak_widths(smoothed, peaks, rel_height=0.9)
    left_ips, right_ips = widths_result[2], widths_result[3]

    lanes = []
    for i, peak in enumerate(peaks):
        x0 = max(0, int(round(left_ips[i])))
        x1 = min(width, int(round(right_ips[i])))
        if x1 <= x0:
            continue
        lanes.append({
            "x": float(x0),
            "y": 0.0,
            "width": float(x1 - x0),
            "height": float(height),
            "index": i,
        })

    return lanes


def to_uint16(image: np.ndarray) -> np.ndarray:
    """Min/max-stretches a float array to the full uint16 range for lossless-enough PNG output."""
    lo, hi = image.min(), image.max()
    span = hi - lo
    if span <= 0:
        return np.zeros(image.shape, dtype=np.uint16)
    scaled = (image - lo) / span * 65535.0
    return scaled.astype(np.uint16)


def main() -> int:
    parser = argparse.ArgumentParser(description="Gel Analysis auto-detect (lanes, deskew, background subtraction)")
    parser.add_argument("--input", required=True, help="Path to the input gel image (PNG or TIFF)")
    parser.add_argument("--output", required=True, help="Output directory for lanes.json and the background-subtracted image")
    args = parser.parse_args()

    os.makedirs(args.output, exist_ok=True)

    print(f"Loading image: {args.input}")
    image = load_image_array(args.input)
    print(f"Image shape: {image.shape}")

    print("Searching for deskew angle...")
    angle = find_deskew_angle(image)
    print(f"Best deskew angle: {angle:.2f} degrees")
    if abs(angle) > 1e-6:
        image = ndimage.rotate(image, angle, reshape=False, order=1, mode="nearest")

    print("Subtracting background...")
    corrected = subtract_background(image)

    print("Detecting lanes...")
    lanes = detect_lanes(corrected)
    print(f"Detected {len(lanes)} lane(s)")

    bg_filename = "bg_subtracted.png"
    bg_path = os.path.join(args.output, bg_filename)
    Image.fromarray(to_uint16(corrected)).save(bg_path)

    result = {
        "deskewAngle": angle,
        "lanes": lanes,
        "backgroundSubtractedImage": bg_filename,
    }
    with open(os.path.join(args.output, "lanes.json"), "w") as f:
        json.dump(result, f, indent=2)

    print("Auto-detect complete.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
