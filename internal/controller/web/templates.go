package web

import (
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"sort"

	"embed"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*.html
var templateFiles embed.FS

func Templates() *template.Template {
	tmpl := template.New("")
	paths, _ := fs.Glob(templateFiles, "templates/*.html")
	sort.Strings(paths)
	for _, path := range paths {
		name := filepath.Base(path)
		content, err := templateFiles.ReadFile(path)
		if err != nil {
			panic(err)
		}
		if _, err = tmpl.New(name).Parse(string(content)); err != nil {
			panic(err)
		}
	}
	return tmpl
}

func RegisterRoutes(router *gin.Engine) {
	router.GET("/", func(c *gin.Context) {
		if _, err := c.Cookie("access_token"); err == nil {
			c.Redirect(http.StatusFound, "/platform")
			return
		}
		c.Redirect(http.StatusFound, "/login")
	})
	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"Title": "Monolith Trading Platform",
		})
	})
	router.GET("/platform", func(c *gin.Context) {
		if _, err := c.Cookie("access_token"); err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}

		c.HTML(http.StatusOK, "platform.html", gin.H{
			"Title":          "Monolith Trading Platform",
			"AccountInitial": "M",
		})
	})
}
