package router

import (
	"image-processor/internal/handler"

	"github.com/wb-go/wbf/ginext"
)

// Router - HTTP роутер
type Router struct {
	engine *ginext.Engine
}

// New создаёт новый роутер
func New() *Router {
	engine := ginext.New("debug")
	engine.Use(ginext.Logger())
	engine.Use(ginext.Recovery())

	return &Router{engine: engine}
}

// Setup настраивает все роуты
func (r *Router) Setup(
	imageHandler *handler.ImageHandler,
	processedPath string,
) {
	// Статические файлы
	r.engine.Static("/static", "./static")
	r.engine.Static("/storage/processed", processedPath)

	// HTML шаблоны
	r.engine.LoadHTMLGlob("templates/*.html")

	// API группа
	api := r.engine.Group("/api")
	{
		api.POST("/upload", imageHandler.Upload)
		api.GET("/image/:id", imageHandler.GetImage)
		api.DELETE("/image/:id", imageHandler.DeleteImage)
		api.GET("/status/:id", imageHandler.GetStatus)
		api.GET("/images", imageHandler.ListImages)
	}

	// Роут для отдачи изображений
	r.engine.GET("/image/:id", imageHandler.GetImage)

	// Главная страница
	r.engine.GET("/", func(c *ginext.Context) {
		c.HTML(200, "index.html", ginext.H{
			"Title": "Image Processor",
		})
	})
}

// Engine возвращает ginext.Engine
func (r *Router) Engine() *ginext.Engine {
	return r.engine
}
