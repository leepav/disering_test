package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"strings"
)

type Ditherer interface {
	Dither(img image.Image) *image.Gray
	Name() string
}

type AtkinsonDitherer struct{}

func (ad AtkinsonDitherer) Dither(img image.Image) *image.Gray {
	return dither(img, [][]int{{0, 0, 1, 1}, {1, 1, 1, 0}, {0, 1, 0, 0}}, 8.0)
}
func (ad AtkinsonDitherer) Name() string {
	return "atkinson"
}

type FloydSteinbergDitherer struct{}

func (fsd FloydSteinbergDitherer) Dither(img image.Image) *image.Gray {
	return dither(img, [][]int{{0, 0, 7}, {3, 5, 1}}, 16.0)
}
func (fsd FloydSteinbergDitherer) Name() string {
	return "floyd_steinberg"
}

type SierraLiteDitherer struct{}

func (sld SierraLiteDitherer) Dither(img image.Image) *image.Gray {
	return dither(img, [][]int{{0, 0, 2}, {1, 1, 0}}, 4.0)
}
func (sld SierraLiteDitherer) Name() string {
	return "sierra_lite"
}

type ShtukiDitherer struct{}

func (bd ShtukiDitherer) Dither(img image.Image) *image.Gray {
	return dither(img, [][]int{{0, 0, 0, 8, 4}, {2, 4, 8, 4, 2}, {1, 2, 4, 2, 1}}, 42.0)
}
func (sd ShtukiDitherer) Name() string {
	return "shtuki"
}

func dither(img image.Image, matrix [][]int, divisor float64) *image.Gray {
	bounds := img.Bounds()
	grayImg := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			grayImg.SetGray(x, y, color.GrayModel.Convert(img.At(x, y)).(color.Gray))
		}
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			oldPixel := grayImg.GrayAt(x, y).Y
			newPixel := findClosestPaletteColor(oldPixel)
			grayImg.SetGray(x, y, color.Gray{Y: newPixel})

			quantError := int(oldPixel) - int(newPixel)

			for i := 0; i < len(matrix); i++ {
				for j := 0; j < len(matrix[i]); j++ {
					if matrix[i][j] != 0 {
						nx, ny := x+j-1, y+i
						if nx >= 0 && nx < bounds.Max.X && ny >= 0 && ny < bounds.Max.Y {
							oldNeighbor := grayImg.GrayAt(nx, ny).Y
							newNeighbor := uint8(math.Min(255, math.Max(0, float64(oldNeighbor)+float64(quantError)*(float64(matrix[i][j])/divisor))))
							grayImg.SetGray(nx, ny, color.Gray{Y: newNeighbor})
						}
					}
				}
			}
		}
	}

	return grayImg
}

func findClosestPaletteColor(pixel uint8) uint8 {
	if pixel < 128 {
		return 0
	}
	return 255
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the path to the image file: ")
	imagePath, _ := reader.ReadString('\n')
	imagePath = strings.TrimSpace(imagePath)

	file, err := os.Open(imagePath)
	if err != nil {
		fmt.Printf("Error opening image file: %v\n", err)
		return
	}
	defer file.Close()

	var img image.Image
	if strings.HasSuffix(strings.ToLower(imagePath), ".jpg") || strings.HasSuffix(strings.ToLower(imagePath), ".jpeg") {
		img, err = jpeg.Decode(file)
	} else if strings.HasSuffix(strings.ToLower(imagePath), ".png") {
		img, err = png.Decode(file)
	} else {
		fmt.Println("Unsupported image format. Please use a JPEG or PNG image.")
		return
	}
	if err != nil {
		fmt.Printf("Error decoding image: %v\n", err)
		return
	}

	fmt.Println("Choose a dithering method:")
	fmt.Println("1. Atkinson")
	fmt.Println("2. Floyd-Steinberg")
	fmt.Println("3. Sierra Lite")
	fmt.Println("4. Shtuki")
	fmt.Print("Enter your choice: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var ditherer Ditherer
	switch choice {
	case "1":
		ditherer = AtkinsonDitherer{}
	case "2":
		ditherer = FloydSteinbergDitherer{}
	case "3":
		ditherer = SierraLiteDitherer{}
	case "4":
		ditherer = ShtukiDitherer{}
	default:
		fmt.Println("Invalid choice. Using Atkinson dithering by default.")
		ditherer = AtkinsonDitherer{}
	}

	dithereredImg := ditherer.Dither(img)

	outputPath := fmt.Sprintf("output_%s.png", ditherer.Name())
	outFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer outFile.Close()

	if err := png.Encode(outFile, dithereredImg); err != nil {
		fmt.Printf("Error encoding image: %v\n", err)
		return
	}

	fmt.Printf("Dithered image saved as %s\n", outputPath)
}
