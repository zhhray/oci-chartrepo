package main

import (
	"flag"
	"net/http"

	"github.com/alauda/oci-chartrepo/pkg"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"k8s.io/klog"
)

func main() {
	var registyOpts = &pkg.RegistryOptions{}
	// flags
	port := flag.String("port", "8080", "listen port")
	//TODO: remove in chart args and here
	flag.String("storage", "registry", "storage backend(only registry for now)")
	flag.StringVar(&registyOpts.URL, "storage-registry-repo", "localhost:5000", "oci registry address")
	flag.StringVar(&registyOpts.Scheme, "storage-registry-scheme", "", "oci registry address scheme")
	flag.Parse()

	pkg.AnalyseResitryOptions(registyOpts)
	klog.Infof("registry scheme is %s", registyOpts.Scheme)

	// Echo instance
	e := echo.New()
	pkg.GlobalBackend = pkg.NewBackend(registyOpts.URL, registyOpts.Scheme, registyOpts.Username, registyOpts.Password)

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", hello)
	e.GET("/index.yaml", pkg.IndexHandler)
	e.GET("/charts/:name", pkg.GetChartHandler)

	// Start server
	e.Logger.Fatal(e.Start(":" + *port))
}

// Handler
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
