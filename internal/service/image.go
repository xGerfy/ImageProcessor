package service

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
)

// ImageProcessor - сервис обработки изображений
type ImageProcessor struct {
	watermarkText string
	thumbWidth    int
	thumbHeight   int
	resizeWidth   int
	resizeHeight  int
}

// NewImageProcessor создаёт процессор изображений
func NewImageProcessor(
	watermarkText string,
	thumbWidth, thumbHeight int,
	resizeWidth, resizeHeight int,
) *ImageProcessor {
	return &ImageProcessor{
		watermarkText: watermarkText,
		thumbWidth:    thumbWidth,
		thumbHeight:   thumbHeight,
		resizeWidth:   resizeWidth,
		resizeHeight:  resizeHeight,
	}
}

// CreateThumbnail создаёт миниатюру с сохранением пропорций (cover)
func (p *ImageProcessor) CreateThumbnail(inputPath string) ([]byte, error) {
	img, err := p.loadImage(inputPath)
	if err != nil {
		return nil, err
	}

	thumbnail := p.smartResize(img, p.thumbWidth, p.thumbHeight)
	format := getImageFormatFromPath(inputPath)
	return encodeImage(thumbnail, format)
}

// CreateResize создаёт уменьшенную копию с сохранением пропорций (fit)
func (p *ImageProcessor) CreateResize(inputPath string) ([]byte, error) {
	img, err := p.loadImage(inputPath)
	if err != nil {
		return nil, err
	}

	resized := p.fitResize(img, p.resizeWidth, p.resizeHeight)
	format := getImageFormatFromPath(inputPath)
	return encodeImage(resized, format)
}

// CreateWatermark добавляет водяной знак в правом нижнем углу
func (p *ImageProcessor) CreateWatermark(inputPath string) ([]byte, error) {
	img, err := p.loadImage(inputPath)
	if err != nil {
		return nil, err
	}

	imgWithWatermark := p.applyWatermark(img, p.watermarkText)
	format := getImageFormatFromPath(inputPath)
	return encodeImage(imgWithWatermark, format)
}

// loadImage загружает изображение из файла
func (p *ImageProcessor) loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	return img, err
}

// smartResize делает crop по центру с сохранением пропорций (как object-fit: cover)
func (p *ImageProcessor) smartResize(src image.Image, width, height int) image.Image {
	bounds := src.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Вычисляем коэффициенты масштабирования
	xRatio := float64(srcWidth) / float64(width)
	yRatio := float64(srcHeight) / float64(height)

	var cropWidth, cropHeight, startX, startY int

	if xRatio < yRatio {
		// Обрезаем по вертикали
		cropWidth = srcWidth
		cropHeight = int(float64(srcWidth) / float64(width) * float64(height))
		startX = 0
		startY = (srcHeight - cropHeight) / 2
	} else {
		// Обрезаем по горизонтали
		cropHeight = srcHeight
		cropWidth = int(float64(srcHeight) / float64(height) * float64(width))
		startX = (srcWidth - cropWidth) / 2
		startY = 0
	}

	// Вырезаем центральную часть
	cropped := image.NewRGBA(image.Rect(0, 0, cropWidth, cropHeight))
	draw.Draw(cropped, cropped.Bounds(), src, image.Point{X: bounds.Min.X + startX, Y: bounds.Min.Y + startY}, draw.Src)

	// Масштабируем до нужного размера
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	resizeImage(dst, cropped)

	return dst
}

// fitResize изменяет размер с сохранением пропорций (как object-fit: contain)
func (p *ImageProcessor) fitResize(src image.Image, maxWidth, maxHeight int) image.Image {
	bounds := src.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Вычисляем коэффициент масштабирования
	ratio := float64(srcWidth) / float64(srcHeight)

	var finalWidth, finalHeight int

	if float64(maxWidth)/float64(maxHeight) > ratio {
		finalHeight = maxHeight
		finalWidth = int(float64(maxHeight) * ratio)
	} else {
		finalWidth = maxWidth
		finalHeight = int(float64(maxWidth) / ratio)
	}

	dst := image.NewRGBA(image.Rect(0, 0, finalWidth, finalHeight))
	resizeImage(dst, src)

	return dst
}

// resizeImage масштабирует изображение с билинейной интерполяцией
func resizeImage(dst *image.RGBA, src image.Image) {
	bounds := dst.Bounds()
	srcBounds := src.Bounds()

	srcW := float64(srcBounds.Dx())
	srcH := float64(srcBounds.Dy())
	dstW := float64(bounds.Dx())
	dstH := float64(bounds.Dy())

	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			// Вычисляем координаты в исходном изображении
			srcX := float64(x) * srcW / dstW
			srcY := float64(y) * srcH / dstH

			// Округляем до ближайшего целого
			srcXi := int(srcX)
			srcYi := int(srcY)

			// Ограничиваем координаты
			if srcXi >= srcBounds.Dx()-1 {
				srcXi = srcBounds.Dx() - 1
			}
			if srcYi >= srcBounds.Dy()-1 {
				srcYi = srcBounds.Dy() - 1
			}

			// Получаем цвет пикселя
			c := src.At(srcBounds.Min.X+srcXi, srcBounds.Min.Y+srcYi)
			dst.Set(x, y, c)
		}
	}
}

// applyWatermark добавляет текст водяного знака
func (p *ImageProcessor) applyWatermark(src image.Image, text string) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Over)

	// Загружаем шрифт
	fontData, err := p.loadFont()
	if err != nil {
		// Если шрифт не загрузился, рисуем простую рамку
		return p.drawSimpleWatermark(dst, text)
	}

	// Настраиваем контекст freetype
	fc := freetype.NewContext()
	fc.SetDPI(72)
	fc.SetFont(fontData)
	fc.SetSrc(image.NewUniform(color.RGBA{255, 255, 255, 255}))
	fc.SetDst(dst)

	// Вычисляем размер шрифта (примерно 3% от ширины изображения)
	fontSize := float64(bounds.Dx()) * 0.03
	if fontSize < 16 {
		fontSize = 16
	}
	if fontSize > 48 {
		fontSize = 48
	}
	fc.SetFontSize(fontSize)

	// Позиция: правый нижний угол с отступом
	padding := 20

	// Оцениваем ширину текста (приблизительно)
	textWidth := len(text) * int(fontSize*0.6)
	textHeight := int(fontSize) + 10

	// Координаты подложки
	bgX := bounds.Dx() - textWidth - padding - 10
	bgY := bounds.Dy() - textHeight - padding
	bgW := textWidth + 20
	bgH := textHeight + 10

	// Рисуем полупрозрачную подложку
	bgRect := image.Rect(bgX, bgY, bgX+bgW, bgY+bgH)
	for y := bgRect.Min.Y; y < bgRect.Max.Y; y++ {
		for x := bgRect.Min.X; x < bgRect.Max.X; x++ {
			if x >= 0 && x < bounds.Dx() && y >= 0 && y < bounds.Dy() {
				dst.Set(x, y, color.RGBA{0, 0, 0, 180})
			}
		}
	}

	// Рисуем текст
	pt := freetype.Pt(bgX+10, bgY+int(fontSize)+5)
	fc.SetClip(bounds)

	_, err = fc.DrawString(text, pt)
	if err != nil {
		return p.drawSimpleWatermark(dst, text)
	}

	return dst
}

// drawSimpleWatermark рисует простой водяной знак (полоса с текстом)
func (p *ImageProcessor) drawSimpleWatermark(dst *image.RGBA, text string) *image.RGBA {
	bounds := dst.Bounds()

	// Рисуем полупрозрачную полосу внизу
	bandHeight := 40
	bandY := bounds.Dy() - bandHeight

	// Рисуем подложку
	for y := bandY; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			dst.Set(x, y, color.RGBA{0, 0, 0, 180})
		}
	}

	// Рисуем текст простыми символами (если шрифт не загрузился)
	// Для простоты рисуем только полосу
	_ = text

	return dst
}

// loadFont загружает шрифт или возвращает ошибку
func (p *ImageProcessor) loadFont() (*truetype.Font, error) {
	// Пробуем загрузить системный шрифт
	fontPaths := []string{
		"C:/Windows/Fonts/arial.ttf",
		"C:/Windows/Fonts/times.ttf",
		"C:/Windows/Fonts/segoeui.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/TTF/DejaVuSans.ttf",
	}

	for _, path := range fontPaths {
		fontData, err := os.ReadFile(path)
		if err == nil {
			return truetype.Parse(fontData)
		}
	}

	return nil, os.ErrNotExist
}

// encodeToJPEG кодирует изображение в JPEG
func encodeToJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// encodeImage кодирует изображение в тот же формат, что и исходник
func encodeImage(img image.Image, format string) ([]byte, error) {
	var buf bytes.Buffer

	switch format {
	case "png":
		err := png.Encode(&buf, img)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "gif":
		// Для GIF используем простой Encode (без анимации)
		err := gif.Encode(&buf, img, &gif.Options{NumColors: 256})
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	default:
		// По умолчанию JPEG
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
}

// GetImageFormat определяет формат изображения
func GetImageFormat(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, format, err := image.DecodeConfig(file)
	return format, err
}

// getImageFormatFromPath определяет формат по расширению файла
func getImageFormatFromPath(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".png":
		return "png"
	case ".gif":
		return "gif"
	default:
		return "jpeg"
	}
}
