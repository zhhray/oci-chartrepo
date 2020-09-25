package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/alauda/oci-chartrepo/pkg"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"k8s.io/klog"
)

const secretCfgPath = "/etc/secret/dockercfg"

var (
	registryScheme = "HTTP"

	registryURL      string
	registryUsername string
	registryPassword string
)

func main() {
	// flags
	port := flag.String("port", "8080", "listen port")
	//TODO: remove in chart args and here
	flag.String("storage", "registry", "storage backend(only registry for now)")
	flag.StringVar(&registryURL, "storage-registry-repo", "localhost:5000", "oci registry address")

	// Scheme to use for connecting to the registry. Defaults to HTTP.
	if _, err := os.Stat(secretCfgPath); !os.IsNotExist(err) {
		// get user info from secretCfgPath
		body, err := ioutil.ReadFile(secretCfgPath)
		if err != nil {
			panic(err)
		}
		var cfg pkg.DockerSecretCfg
		if err := json.Unmarshal(body, &cfg); err != nil {
			panic(err)
		}

		for k, v := range cfg.Auths {
			registryURL = k
			registryScheme = "HTTPS"
			registryUsername = v.Username
			registryPassword = v.Password

			break
		}
	}

	klog.Infof("registry scheme is %s", registryScheme)
	flag.Parse()

	// Echo instance
	e := echo.New()
	pkg.GlobalBackend = pkg.NewBackend(registryURL, registryScheme, registryUsername, registryPassword)

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
