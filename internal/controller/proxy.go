package controller

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func (c Controller) ProxyToService(serviceName string, port int32) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if svc, err := c.client.CoreV1().Services("default").Get(context.Background(), serviceName, metaV1.GetOptions{}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			target, err := url.Parse(fmt.Sprintf("http://%v:%v", svc.Spec.ExternalName, port))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Add proxy headers here

			if svc.Spec.Type == v1.ServiceTypeExternalName {
				r.Host = r.URL.Host
			}
			httputil.NewSingleHostReverseProxy(target).ServeHTTP(w, r)
		}
	})
}
