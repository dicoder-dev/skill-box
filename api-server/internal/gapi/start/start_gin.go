package start

import (
	"errors"
	"log"
	"net/http"

	"ginp-api/configs"
	"ginp-api/pkg/server"
)

func startGinServer() {
	srv := server.New(server.Options{
		Addr:      ":" + configs.ServerPort(),
		ViewGlob:  "view/*",
		StaticDir: "./static",
	})
	println("start server on port: " + configs.ServerPort())
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
