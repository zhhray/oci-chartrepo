package main

import (
	"flag"
	"net/http"
	// "time"

	"github.com/alauda/oci-chartrepo/pkg"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	var opts = &pkg.HubOptions{}
	// flags
	port := flag.String("port", "8080", "listen port")
	//TODO: remove in chart args and here
	flag.String("storage", "registry", "storage backend(only registry and harbor for now)")
	flag.StringVar(&opts.URL, "storage-registry-repo", "localhost:5000", "oci registry address")
	flag.StringVar(&opts.Scheme, "storage-registry-scheme", "", "oci registry address scheme, default is empty means that the scheme will be automatically determined. If it is found that --storage-registry-repo is a harbor v2+, it will automatically HTTPS, the value setting is invalid.")
	flag.Parse()

	// Get user info from secret config file (if it exists), and fill opts
	if err := opts.FullfillHubOptions(); err != nil {
		panic(err)
	}
	// Try to load whitelist
	pkg.LoadWhiteList()

	// Echo instance
	e := echo.New()

	pkg.GlobalBackend = pkg.NewBackend(opts)
	// When multiple instance of oci-chartrepo exist, this will make sure every instance
	// has the internal cache before it gets requrest to individual chart. Of course this will slow down
	// the startup process, we need to add heathcheck later
	// TODO: add health check for pod
	if err := pkg.RefreshIndexData(); err != nil {
		e.Logger.Fatal("init chart registry cache error", err)
	}

	// start background job
	// go func() {
	// 	for {
	// 		// every 5 min
	// 		time.Sleep(5 * time.Minute)

	// 		// ignore error in background job
	// 		pkg.RefreshIndexData()
	// 	}
	// }()

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
	return c.String(http.StatusOK, "Hello, OCI!")
}
