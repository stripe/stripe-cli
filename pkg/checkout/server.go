package checkout

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/open"
)

// Data contains the required info for checkout sessions
type Data struct {
	PublishableKey string
	SessionID      string
}

// Server maps required data to run a simple checkout integration
type Server struct {
	Cfg *config.Config

	Port      string
	sessionID string
	data      *Data
}

func (s *Server) checkoutHandler(w http.ResponseWriter, req *http.Request) {
	tmpl, err := RedirectTemplate()
	if err != nil {
		panic(err)
	}

	err = tmpl.Execute(w, s.data)
	if err != nil {
		panic(err)
	}
}

func (s *Server) successHandler(w http.ResponseWriter, req *http.Request) {
	tmpl, err := SuccessTemplate()
	if err != nil {
		panic(err)
	}

	err = tmpl.Execute(w, s.data)
	if err != nil {
		panic(err)
	}
}

func (s *Server) cancelHandler(w http.ResponseWriter, req *http.Request) {
	tmpl, err := CancelTemplate()
	if err != nil {
		panic(err)
	}

	err = tmpl.Execute(w, s.data)
	if err != nil {
		panic(err)
	}
}

// Run creates a checkout session, retrieves the publishable key, and sets up
// a simple server to show checkout
func (s *Server) Run() error {
	var err error

	session := s.sessionID
	if session == "" {
		session, err = getOrCreateSession(s.Cfg, s.Port)
		if err != nil {
			return err
		}
	}

	publishableKey, err := s.Cfg.Profile.GetPublishableKey(false)
	if err != nil {
		return err
	}

	s.data = &Data{
		SessionID:      session,
		PublishableKey: publishableKey,
	}

	fmt.Println("Starting stripe server at address", fmt.Sprintf("http://0.0.0.0:%s", s.Port))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(FS)))
	http.HandleFunc("/success", s.successHandler)
	http.HandleFunc("/cancel", s.cancelHandler)
	http.HandleFunc("/", s.checkoutHandler)

	go func() {
		time.Sleep(1 * time.Second)
		open.Browser(fmt.Sprintf("http://0.0.0.0:%s", s.Port))
	}()

	err = http.ListenAndServe(fmt.Sprintf("localhost:%s", s.Port), handlers.LoggingHandler(os.Stdout, http.DefaultServeMux))

	return err
}
