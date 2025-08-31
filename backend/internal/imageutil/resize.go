package imageutil

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"math"
)

// ResizeImage scales the provided image to the target width and height using
// bilinear interpolation. It returns an image.NRGBA (RGBA-like) result.
// If the provided size is identical to the source size, the original image is
// returned as-is (but converted to *image.NRGBA).
func ResizeImage(src image.Image, dstW, dstH int) (image.Image, error) {
	if dstW <= 0 || dstH <= 0 {
		return nil, errors.New("invalid target size")
	}

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	if srcW == 0 || srcH == 0 {
		return nil, errors.New("source image has zero size")
	}

	// Fast path: same size — normalize to NRGBA for consistent encoding.
	if srcW == dstW && srcH == dstH {
		dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
		draw.Draw(dst, dst.Bounds(), src, srcBounds.Min, draw.Src)
		return dst, nil
	}

	// Convert source to NRGBA to have 8-bit RGBA channels in a predictable layout.
	srcNRGBA := image.NewNRGBA(srcBounds)
	draw.Draw(srcNRGBA, srcBounds, src, srcBounds.Min, draw.Src)

	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))

	// Scale factors
	sx := float64(srcW) / float64(dstW)
	sy := float64(srcH) / float64(dstH)

	// Helper to fetch clamped pixel channel values
	get := func(x, y, c int) float64 {
		if x < srcBounds.Min.X {
			x = srcBounds.Min.X
		}
		if y < srcBounds.Min.Y {
			y = srcBounds.Min.Y
		}
		if x >= srcBounds.Max.X {
			x = srcBounds.Max.X - 1
		}
		if y >= srcBounds.Max.Y {
			y = srcBounds.Max.Y - 1
		}
		off := srcNRGBA.PixOffset(x, y)
		return float64(srcNRGBA.Pix[off+c])
	}

	// Bilinear interpolation per destination pixel
	for j := 0; j < dstH; j++ {
		// source Y coordinate
		fy := (float64(j)+0.5)*sy - 0.5
		y0 := int(math.Floor(fy))
		wy := fy - float64(y0)
		for i := 0; i < dstW; i++ {
			fx := (float64(i)+0.5)*sx - 0.5
			x0 := int(math.Floor(fx))
			wx := fx - float64(x0)

			// neighbors
			x1 := x0 + 1
			y1 := y0 + 1

			// clamp coordinates relative to original bounds
			x0c := x0 + srcBounds.Min.X
			x1c := x1 + srcBounds.Min.X
			y0c := y0 + srcBounds.Min.Y
			y1c := y1 + srcBounds.Min.Y

			// Interpolate each channel
			var r, g, b, a float64
			for ch := 0; ch < 4; ch++ {
				p00 := get(x0c, y0c, ch)
				p10 := get(x1c, y0c, ch)
				p01 := get(x0c, y1c, ch)
				p11 := get(x1c, y1c, ch)

				v := (1-wx)*(1-wy)*p00 + wx*(1-wy)*p10 + (1-wx)*wy*p01 + wx*wy*p11
				switch ch {
				case 0:
					r = v
				case 1:
					g = v
				case 2:
					b = v
				case 3:
					a = v
				}
			}

			off := dst.PixOffset(i, j)
			dst.Pix[off+0] = uint8(clamp(r, 0, 255))
			dst.Pix[off+1] = uint8(clamp(g, 0, 255))
			dst.Pix[off+2] = uint8(clamp(b, 0, 255))
			dst.Pix[off+3] = uint8(clamp(a, 0, 255))
		}
	}

	return dst, nil
}

// ResizePNGBytes decodes PNG bytes, resizes the image to target width/height
// and returns the resulting PNG bytes.
func ResizePNGBytes(pngBytes []byte, dstW, dstH int) ([]byte, error) {
	if dstW <= 0 || dstH <= 0 {
		return nil, errors.New("invalid target size")
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, err
	}
	outImg, err := ResizeImage(img, dstW, dstH)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, outImg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
