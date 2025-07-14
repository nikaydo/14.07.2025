package main

import (
	"fmt"
	"main/internal/config"
	"main/internal/handlers"
	"net/http"
)

func main() {
	env := config.GetConfig()
	handlers.Init(env)
	http.ListenAndServe(fmt.Sprintf("%s:%d", env.HOST, env.PORT), nil)
}
