package http

import (
	"context"
	"net/http"

	"github.com/4nth0/golem/pkg/template"
	"github.com/4nth0/golem/server"
	log "github.com/sirupsen/logrus"
)

// HTTPHandler
type HTTPHandler struct {
	Method   string                 `yaml:"method,omitempty"`
	Methods  map[string]HTTPHandler `yaml:"methods,omitempty"`
	Body     string                 `yaml:"body,omitempty"`
	BodyFile string                 `yaml:"body_file,omitempty"`
	Code     int                    `yaml:"code,omitempty"`
	Headers  map[string]string      `yaml:"headers,omitempty"`
	Handler  *Handler               `yaml:"handler,omitempty"` // Should be removed if not used
}

// Handler
type Handler struct {
	Type         string `yaml:"type"`
	Template     string `yaml:"template"`
	TemplateFile string `yaml:"template_file"`
}

// ServerConfig
type ServerConfig struct {
	Routes map[string]HTTPHandler
}

var (
	DefaultMethod     string = "GET"
	DefaultStatusCode int    = http.StatusOK
)

// LaunchService
func LaunchService(ctx context.Context, defaultServer *server.Client, port string, globalVars map[string]string, config ServerConfig, requests chan server.InboundRequest) {
	var s *server.Client

	log.Info("Launch new HTTP service")

	if port != "" {
		log.Debug("Port provided, create a new server")
		s = server.NewServer(port, requests)
	} else if defaultServer != nil {
		log.Debug("No port provided, use the default server")
		s = defaultServer
	} else {
		log.Info("There is no available server")
		return
	}

	log.Info("Start routes injection")
	for path, route := range config.Routes {
		if len(route.Methods) > 0 {
			for method, route := range route.Methods {
				route.Method = method
				launch(path, route, globalVars, s)
			}
		} else {
			launch(path, route, globalVars, s)
		}
	}

	if port != "" {
		s.Listen(ctx)
	}
}

func launch(path string, route HTTPHandler, globalVars map[string]string, s *server.Client) {
	if route.Code == 0 {
		log.WithFields(
			log.Fields{
				"code": DefaultStatusCode,
			}).Debug("Status code not provided, use default.")

		route.Code = DefaultStatusCode
	}
	if route.Method == "" {
		log.WithFields(
			log.Fields{
				"method": DefaultMethod,
			}).Debug("HTTP method not provided, use default.")

		route.Method = DefaultMethod
	}

	if route.Body == "" && route.BodyFile != "" {
		log.WithFields(
			log.Fields{
				"path": route.BodyFile,
			}).Debug("Use body template file.")

		result, err := template.LoadTemplate(route.BodyFile)
		if err != nil {
			log.WithFields(
				log.Fields{
					"path": route.BodyFile,
				}).Info("Adding new route")
		} else {
			route.Body = result
		}
	}

	log.WithFields(
		log.Fields{
			"method": route.Method,
			"path":   path,
		}).Info("Adding new route")

	s.Router.Add(route.Method, path, func(w http.ResponseWriter, r *http.Request, params map[string]string) {

		log.WithFields(
			log.Fields{
				"method": route.Method,
				"path":   path,
				"status": route.Code,
			}).Info("New inbound request.")

		if len(route.Headers) > 0 {
			for key, value := range route.Headers {

				log.WithFields(
					log.Fields{
						"key":   key,
						"value": value,
					}).Info("Inject response header.")

				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(route.Code)

		response := template.ExecuteTemplate(route.Body, globalVars, params)
		w.Write([]byte(response))

	})
}
