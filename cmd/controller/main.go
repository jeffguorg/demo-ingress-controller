package main

import (
	"github.com/guochao/demo-ingress-controller/internal/controller"

	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"

	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
)

var (
	fListenAddr     = flag.String("listen", envString("LISTEN_ADDR", ":1234"), "")
	fKubeConfigPath = flag.String("kubeconfig", envString("KUBECONFIG", ""), "")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errGroup, ctx := errgroup.WithContext(ctx)

	router := mux.NewRouter()
	httpServer := http.Server{
		Addr:    *fListenAddr,
		Handler: router,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}
	errGroup.Go(func() error {
		log.Println("listen on ", *fListenAddr)
		return httpServer.ListenAndServe()
	})

	errGroup.Go(func() error {
		ingressController, err := controller.New(*fKubeConfigPath, router)
		if err != nil {
			return err
		}

		log.Println("watching for ingress")
		return ingressController.Run(ctx)
	})
	if err := errGroup.Wait(); err != nil {
		log.Println(err)
	}
}

func envString(k, d string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return d
}
