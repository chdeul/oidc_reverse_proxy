package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"main/auth"
	"main/rproxy"
	"main/tools"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {

	// ---- Initialisations générales ----
	fmt.Println("Début Frontal Agréments")

	// Initialisation  Config
	//time.LoadLocation("Europe/Paris")
	tools.LoadProxyConfig(".")

	// Gére l'authentification ( /frontal/login, logout, callback ) : option
	if tools.Cfg.Openidc.Active {
		err := auth.OpClient()
		if err != nil {
			fmt.Println("Erreur d'initialisation OIDC client", err)
		}
	}
	// Liveness readiness ( /healthz )
	tools.InitChecks()

	// Serve static files ( /resources ) : option
	path := tools.Cfg.Proxy.ResourcesPath
	if path != "" {
		fs := http.FileServer(http.Dir("./resources/"))
		http.Handle(path, http.StripPrefix(path, fs))
	}

	// Proxy gère toutes les requetes ( sauf /frontal /healthz /resources )
	rproxy.Init()

	// Serveur proxy HTTP
	svr := &http.Server{
		TLSConfig:    &tls.Config{MinVersion: tls.VersionTLS12},
		Addr:         tools.Cfg.Proxy.ListenIp + ":" + tools.Cfg.Proxy.Port,
		Handler:      http.DefaultServeMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Sur Ctrl C ou kill
	go detecterArret(svr)

	if tools.Cfg.Proxy.Protocol == "https" {
		// https : en production pour transfert mots de passe crypté
		err := svr.ListenAndServeTLS(tools.Cfg.Proxy.CertFile, tools.Cfg.Proxy.KeyFile)
		if err != nil {
			fmt.Println("Erreur ListenAndServeTLS: ", err)
		}
	} else {
		// Http pour test
		err := svr.ListenAndServe()
		if err != nil {
			fmt.Println("--Main-- Erreur ListenAndServe: ", err)
		}
	}

	fmt.Println("Fin Frontal Agréments")
}

func detecterArret(svr *http.Server) {

	// Capture CTRL+C ou kill -SIGINT
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop // On attend le signal

	fmt.Println("Arrêt demandé...")

	// Timeout pour l'arrêt propre
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := svr.Shutdown(ctx); err != nil {
		fmt.Println("Arrêt forcé: ", err)
	}
}
