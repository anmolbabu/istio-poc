package main

import (
	"fmt"
	"net"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
)

func getRule() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "config.istio.io/v1alpha2",
			"kind":       "rule",
			"metadata": map[string]interface{}{
				"name":      "recommendationrequestcountprom",
				"namespace": "istio-system",
			},
			"spec": map[string]interface{}{
				"match": "destination.service == \"recommendation.tutorial.svc.cluster.local\"",
				"actions": map[string]interface{}{
					"handler":   "recommendationrequestcounthandler.prometheus",
					"instances": []string{"recommendationrequestcount.metric"},
				},
			},
		},
	}
}

func done() {
	for {
	}
}

func main() {
	metric_name := "recommendationrequestcount"
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		fmt.Printf("unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined")
	}
	metric_namespace := "istio-system"
	resource := &metav1.APIResource{Name: "rules", Namespaced: len(metric_namespace) != 0}
	kube_client, err := dynamic.NewClient(
		&restclient.Config{
			Host: "https://" + net.JoinHostPort(host, port),
			ContentConfig: restclient.ContentConfig{
				GroupVersion: &schema.GroupVersion{
					Group:   "config.istio.io",
					Version: "v1alpha2",
				},
			},
			BearerToken: os.Getenv("BEARER_STR"),
		},
	)

	if err != nil {
		fmt.Printf("unexpected error when creating client: %v", err)
		done()
	}

	got, err := kube_client.Resource(resource, metric_namespace).List(metav1.ListOptions{})
	if err != nil {
		fmt.Printf("unexpected error when listing rules %v", err)
		done()
	}
	fmt.Printf("The list of rules are %v", got)

	got, err = kube_client.Resource(resource, metric_namespace).Create(getRule())
	if err != nil {
		fmt.Printf("unexpected error when creating %q: %v", metric_name, err)
		done()
	}
	fmt.Printf("Received response: %v\n", got)
}
