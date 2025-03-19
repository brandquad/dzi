package dzi

import "math"

// Cmyk2rgb converts a CMYK color value to RGB
func Cmyk2rgb(cmyk []float64) []int {
	var r, g, b float64
	r = 255.0 * (1 - cmyk[0]/100) * (1 - cmyk[3]/100)
	g = 255.0 * (1 - cmyk[1]/100) * (1 - cmyk[3]/100)
	b = 255.0 * (1 - cmyk[2]/100) * (1 - cmyk[3]/100)
	return []int{int(math.Ceil(r)), int(math.Ceil(g)), int(math.Ceil(b))}
}

// Lab2rgb converts a LAB color value to RGB
func Lab2rgb(lab []float64) []int {
	var y float64 = (lab[0] + 16) / 116
	var x float64 = lab[1]/500 + y
	var z float64 = y - lab[2]/200
	var r, g, b float64

	if x*x*x > 0.008856 {
		x = 0.95047 * x * x * x
	} else {
		x = 0.95047 * ((x - 16/116) / 7.787)
	}
	if y*y*y > 0.008856 {
		y = 1.00000 * (y * y * y)
	} else {
		y = 1.00000 * ((y - 16/116) / 7.787)
	}
	if z*z*z > 0.008856 {
		z = 1.08883 * (z * z * z)
	} else {
		z = 1.08883 * ((z - 16/116) / 7.787)
	}

	r = x*3.2406 + y*-1.5372 + z*-0.4986
	g = x*-0.9689 + y*1.8758 + z*0.0415
	b = x*0.0557 + y*-0.2040 + z*1.0570

	if r > 0.0031308 {
		r = 1.055*math.Pow(r, 1/2.4) - 0.055
	} else {
		r = 12.92 * r
	}

	if g > 0.0031308 {
		g = 1.055*math.Pow(g, 1/2.4) - 0.055
	} else {
		g = 12.92 * g
	}
	if b > 0.0031308 {
		b = 1.055*math.Pow(b, 1/2.4) - 0.055
	} else {
		b = 12.92 * b
	}

	return []int{
		int(math.Ceil(math.Max(0, math.Min(1, r)) * 255)),
		int(math.Ceil(math.Max(0, math.Min(1, g)) * 255)),
		int(math.Ceil(math.Max(0, math.Min(1, b)) * 255)),
	}
}
