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
	Dither(img image.Image, isColor bool) image.Image
	Name() string
}

type AtkinsonDitherer struct{}

func (ad AtkinsonDitherer) Dither(img image.Image, isColor bool) image.Image {
	if isColor {
		return ditherColor(img, [][]int{{0, 0, 1, 1}, {1, 1, 1, 0}, {0, 1, 0, 0}}, 8.0)
	}
	return ditherMono(img, [][]int{{0, 0, 1, 1}, {1, 1, 1, 0}, {0, 1, 0, 0}}, 8.0)
}

func (ad AtkinsonDitherer) Name() string {
	return "atkinson"
}

type FloydSteinbergDitherer struct{}

func (fsd FloydSteinbergDitherer) Dither(img image.Image, isColor bool) image.Image {
	if isColor {
		return ditherColor(img, [][]int{{0, 0, 7}, {3, 5, 1}}, 16.0)
	}
	return ditherMono(img, [][]int{{0, 0, 7}, {3, 5, 1}}, 16.0)
}
func (fsd FloydSteinbergDitherer) Name() string {
	return "floyd_steinberg"
}

type ShtukiDitherer struct{}

func (sd ShtukiDitherer) Dither(img image.Image, isColor bool) image.Image {
	if isColor {
		return ditherColor(img, [][]int{{0, 0, 0, 8, 4}, {2, 4, 8, 4, 2}, {1, 2, 4, 2, 1}}, 42.0)
	}
	return ditherMono(img, [][]int{{0, 0, 0, 8, 4}, {2, 4, 8, 4, 2}, {1, 2, 4, 2, 1}}, 42.0)
}
func (sd ShtukiDitherer) Name() string {
	return "shtuki"
}

type SierraLiteDitherer struct{}

func (sld SierraLiteDitherer) Dither(img image.Image, isColor bool) image.Image {
	if isColor {
		return ditherColor(img, [][]int{{0, 0, 2}, {1, 1, 0}}, 4.0)
	}
	return ditherMono(img, [][]int{{0, 0, 2}, {1, 1, 0}}, 4.0)
}

func (sld SierraLiteDitherer) Name() string {
	return "sierra_lite"
}

func ditherColor(img image.Image, matrix [][]int, divisor float64) *image.RGBA {
	bounds := img.Bounds()
	ditheredImg := image.NewRGBA(bounds)

	red := extractChannel(img, 0)
	green := extractChannel(img, 1)
	blue := extractChannel(img, 2)

	ditheredRed := ditherChannel(red, matrix, divisor)
	ditheredGreen := ditherChannel(green, matrix, divisor)
	ditheredBlue := ditherChannel(blue, matrix, divisor)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r := ditheredRed.GrayAt(x, y).Y
			g := ditheredGreen.GrayAt(x, y).Y
			b := ditheredBlue.GrayAt(x, y).Y
			ditheredImg.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	return ditheredImg
}

func ditherMono(img image.Image, matrix [][]int, divisor float64) *image.Gray {
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
						nx, ny := x+j-2, y+i
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

func extractChannel(img image.Image, channelIndex int) *image.Gray {
	bounds := img.Bounds()
	grayImg := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			var value uint8
			switch channelIndex {
			case 0:
				value = uint8(r >> 8) // Red channel
			case 1:
				value = uint8(g >> 8) // Green channel
			case 2:
				value = uint8(b >> 8) // Blue channel
			}
			grayImg.SetGray(x, y, color.Gray{Y: value})
		}
	}

	return grayImg
}

func ditherChannel(channel *image.Gray, matrix [][]int, divisor float64) *image.Gray {
	bounds := channel.Bounds()
	ditheredChannel := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ditheredChannel.SetGray(x, y, channel.GrayAt(x, y))
		}
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			oldPixel := ditheredChannel.GrayAt(x, y).Y
			newPixel := findClosestPaletteColor(oldPixel)
			ditheredChannel.SetGray(x, y, color.Gray{Y: newPixel})

			quantError := int(oldPixel) - int(newPixel)

			for i := 0; i < len(matrix); i++ {
				for j := 0; j < len(matrix[i]); j++ {
					if matrix[i][j] != 0 {
						nx, ny := x+j-2, y+i
						if nx >= 0 && nx < bounds.Max.X && ny >= 0 && ny < bounds.Max.Y {
							oldNeighbor := ditheredChannel.GrayAt(nx, ny).Y
							newNeighbor := uint8(math.Min(255, math.Max(0, float64(oldNeighbor)+float64(quantError)*(float64(matrix[i][j])/divisor))))
							ditheredChannel.SetGray(nx, ny, color.Gray{Y: newNeighbor})
						}
					}
				}
			}
		}
	}

	return ditheredChannel
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

	fmt.Print("Choose dithering mode (1 for color, 2 for mono): ")
	mode, _ := reader.ReadString('\n')
	mode = strings.TrimSpace(mode)
	isColor := mode == "1"

	fmt.Println("Choose a dithering method:")
	fmt.Println("1. Atkinson")
	fmt.Println("2. FloydSteinberg")
	fmt.Println("3. Shtuki")
	fmt.Println("4. Sierra Lite")
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
		ditherer = ShtukiDitherer{}
	case "4":
		ditherer = SierraLiteDitherer{}
	default:
		fmt.Println("Invalid choice. Using Atkinson dithering by default.")
		ditherer = AtkinsonDitherer{}
	}

	ditheredImg := ditherer.Dither(img, isColor)

	modeName := "color"
	if !isColor {
		modeName = "mono"
	}
	outputPath := fmt.Sprintf("output/output_%s_%s.png", ditherer.Name(), modeName)
	outFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer outFile.Close()

	if err := png.Encode(outFile, ditheredImg); err != nil {
		fmt.Printf("Error encoding image: %v\n", err)
		return
	}

	fmt.Printf("Dithered image saved as %s\n", outputPath)
}
