package crypto

import (
	"image"
	"image/color"
	"image/png"
	"bytes"
	"testing"
)

func TestDHashCalculationAndHammingDistance(t *testing.T) {
	// Generate simple test image A
	imgA := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			imgA.Set(x, y, color.RGBA{R: uint8(x * 2), G: uint8(y * 2), B: 100, A: 255})
		}
	}
	var bufA bytes.Buffer
	if err := png.Encode(&bufA, imgA); err != nil {
		t.Fatalf("Failed to encode test image A: %v", err)
	}

	hexA, uA, err := ComputeDHash(bufA.Bytes())
	if err != nil {
		t.Fatalf("ComputeDHash error: %v", err)
	}
	if len(hexA) != 16 {
		t.Fatalf("Expected 16-hex length, got: %s", hexA)
	}

	// Identical image should have Hamming Distance 0
	dist := HammingDistance(uA, uA)
	if dist != 0 {
		t.Fatalf("Expected distance 0 for identical hash, got: %d", dist)
	}

	distHex := HammingDistanceHex(hexA, hexA)
	if distHex != 0 {
		t.Fatalf("Expected distance 0 for identical hex, got: %d", distHex)
	}

	// Slightly modified image B (98% identical)
	imgB := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			imgB.Set(x, y, color.RGBA{R: uint8(x * 2 + 1), G: uint8(y * 2), B: 100, A: 255})
		}
	}
	var bufB bytes.Buffer
	_ = png.Encode(&bufB, imgB)
	hexB, _, _ := ComputeDHash(bufB.Bytes())

	distNear := HammingDistanceHex(hexA, hexB)
	if distNear > 5 {
		t.Fatalf("Expected near duplicate distance <= 5, got: %d", distNear)
	}
}

