package http

import (
	staticfiles "github.com/cusox/watchend/web/static"
	"io/fs"
	nethttp "net/http"
)

func (s *Server) static() nethttp.Handler {
	assets, _ := fs.Sub(staticfiles.FS, ".")

	files := nethttp.StripPrefix("/static/", nethttp.FileServer(nethttp.FS(assets)))

	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		files.ServeHTTP(w, r)
	})
}

func (s *Server) health(w nethttp.ResponseWriter, _ *nethttp.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("ok\n"))
}

func (s *Server) ready(w nethttp.ResponseWriter, r *nethttp.Request) {
	if s.readiness != nil {
		if err := s.readiness.Ready(r.Context()); err != nil {
			nethttp.Error(w, "not ready", nethttp.StatusServiceUnavailable)
			return
		}
	}

	s.health(w, r)
}

// func (s *Server) dashboard(w nethttp.ResponseWriter, r *nethttp.Request) {
// 	d, err := s.store.Dashboard(r.Context(), 10)
// 	if err != nil {
// 		nethttp.Error(w, "internal server error", 500)
// 		return
// 	}
//
// 	s.render(w, "dashboard", pageData{
// 		Title:     "Dashboard",
// 		CSRFToken: csrfToken(r),
// 		Dashboard: d,
// 		Events:    d.RecentEvents,
// 	})
// }
