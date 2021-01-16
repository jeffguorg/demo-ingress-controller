package controller

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	networkv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

type Controller struct {
	client *kubernetes.Clientset

	router *mux.Router
}

func New(kubeConfig string, router *mux.Router) (*Controller, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &Controller{client: clientSet, router: router}, nil
}

func (c *Controller) Run(ctx context.Context) error {
	watcher, err := c.client.NetworkingV1().Ingresses("default").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for {
		select {
		case event := <-watcher.ResultChan():
			switch obj := event.Object.(type) {
			case *v1.Ingress:
				switch event.Type {
				case watch.Added:
					c.add(obj)
					obj.Status.LoadBalancer.Ingress = []networkv1.LoadBalancerIngress{
						{
							Hostname: "my-ingress",
						},
					}
					if _, err := c.client.NetworkingV1().Ingresses("default").UpdateStatus(ctx, obj, metav1.UpdateOptions{}); err != nil {
						log.Print("error while update ingress: ", err)
					}
				case watch.Deleted:
					c.remove(obj)
				case watch.Modified:
					c.remove(obj)
					c.add(obj)
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Controller) add(ing *v1.Ingress) {
	for _, rule := range ing.Spec.Rules {
		for _, path := range rule.IngressRuleValue.HTTP.Paths {
			route := c.router.Name(fmt.Sprintf("%v:%v", rule.Host, path.Path))
			if rule.Host != "" {
				route = route.Host(rule.Host)
			}
			if path.PathType == nil || *path.PathType == v1.PathTypeImplementationSpecific || *path.PathType == v1.PathTypePrefix {
				route = route.PathPrefix(path.Path)
			} else {
				route = route.Path(path.Path)
			}
			log.Printf("%v/%v(%v) -> %v:%v", rule.Host, path.Path, *path.PathType, path.Backend.Service.Name, path.Backend.Service.Port.Number)
			route.Handler(c.ProxyToService(path.Backend.Service.Name, path.Backend.Service.Port.Number))
		}
	}
}

func (c *Controller) remove(ing *v1.Ingress) {
	for _, rule := range ing.Spec.Rules {
		for _, path := range rule.IngressRuleValue.HTTP.Paths {
			c.router.Name(fmt.Sprintf("%v:%v", rule.Host, path.Path)).Handler(nil)
			log.Printf("%v/%v(%v) -> nil", rule.Host, path.Path, *path.PathType)
		}
	}
}
