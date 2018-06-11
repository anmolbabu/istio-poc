package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func getMetric(metric Metric) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": metric.ApiVersion,
			"kind":       "metric",
			"metadata": map[string]interface{}{
				"name":      metric.Name,
				"namespace": metric.Namespace,
			},
			"spec": map[string]interface{}{
				"value": "1",
				"dimensions": map[string]interface{}{
					"source":     "source.service | \"unknown\"",
					"user_agent": "request.headers[\"user-agent\"] | \"unknown\"",
				},
				"monitored_resource_type": "\"UNSPECIFIED\"",
			},
		},
	}
}

type Metric struct {
	Name       string `json:"name,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	ApiVersion string `json:"api_version,omitempty"`
}

func getPrometheus(metric Metric) *unstructured.Unstructured {
	labelNames := []string{
		"source",
		"user_agent",
	}
	metric1 := map[string]interface{}{
		"name":          "source_agent_request_count",
		"instance_name": "sourceagentrequestcount.metric.istio-system",
		"kind":          "COUNTER",
		"label_names":   labelNames,
	}
	var metrics [1]map[string]interface{}
	metrics[0] = metric1
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": metric.ApiVersion,
			"kind":       "prometheus",
			"metadata": map[string]interface{}{
				"name":      "sourceagenthandler",
				"namespace": metric.Namespace,
			},
			"spec": map[string]interface{}{
				"metrics": metrics,
			},
		},
	}
}

func getRule(metric Metric) *unstructured.Unstructured {
	/*
		apiVersion: "config.istio.io/v1alpha2"
		kind: rule
		metadata:
		  name: doubleprom
		  namespace: istio-system
		spec:
		  actions:
		  - handler: doublehandler.prometheus
		    instances:
		    - doublerequestcount.metric

	*/
	var instances [1]string
	instances[0] = "sourceagentrequestcount.metric"
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": metric.ApiVersion,
			"kind":       "rule",
			"metadata": map[string]interface{}{
				"name":      "sourceagentprom",
				"namespace": metric.Namespace,
			},
			"spec": map[string]interface{}{
				"actions": []map[string]interface{}{
					map[string]interface{}{
						"handler":   "sourceagenthandler.prometheus",
						"instances": instances,
					},
				},
			},
		},
	}
}

func done() {
	/* 	for {
	   	}
	*/
}

func CreateMetric(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	var metric Metric
	_ = json.NewDecoder(r.Body).Decode(&metric)
	metric_name := metric.Name
	fmt.Printf("The params passed are %v and body is %v", params, metric)
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		fmt.Printf("unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined")
	}
	metric_namespace := metric.Namespace

	inClusterConfig := true
	inClusterConfig, err := strconv.ParseBool(os.Getenv("IN_CLUSTER_CONFIG"))
	var config *restclient.Config
	if err != nil {
		panic(err)
	}
	if !inClusterConfig {
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		config, err = restclient.InClusterConfig()
	}
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	fmt.Println("Created clientcmd")
	config.Host = "https://" + host
	config.BearerToken = os.Getenv("BEARER_STR")
	config.APIPath = "/apis"
	config.ContentConfig = restclient.ContentConfig{
		GroupVersion: &schema.GroupVersion{
			Group:   "config.istio.io",
			Version: "v1alpha2",
		},
	}
	//fmt.Printf("The config is %v\n", config)
	flag.Parse()
	kube_client, err := dynamic.NewClient(config)
	if err != nil {
		fmt.Printf("unexpected error when creating client: %v", err)
		done()
	}

	/*
		got, err := kube_client.Resource(resource, metric_namespace).List(metav1.ListOptions{})
		if err != nil {
			fmt.Printf("unexpected error when listing rules %v", err)
			done()
		}
		fmt.Printf("The list of rules are %v", got)
	*/

	resource := &metav1.APIResource{Name: "metrics", Namespaced: len(metric_namespace) != 0}
	got, err := kube_client.Resource(resource, metric_namespace).Create(getMetric(metric))
	if err != nil {
		fmt.Printf("unexpected error when creating %q: %v", metric_name, err)
		done()
	}
	fmt.Printf("Received response: %v\n", got)

	resource = &metav1.APIResource{Name: "prometheuses", Namespaced: len(metric_namespace) != 0}
	got, err = kube_client.Resource(resource, metric_namespace).Create(getPrometheus(metric))
	if err != nil {
		fmt.Printf("unexpected error when creating %q: %v", metric_name, err)
		done()
	}
	fmt.Printf("Received response: %v\n", got)

	resource = &metav1.APIResource{Name: "rules", Namespaced: len(metric_namespace) != 0}
	got, err = kube_client.Resource(resource, metric_namespace).Create(getRule(metric))
	if err != nil {
		fmt.Printf("unexpected error when creating %q: %v", metric_name, err)
		done()
	}
	fmt.Printf("Received response: %v\n", got)

}

func Readiness(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Success")
}

func Liveness(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Success")
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/metric/{name}", CreateMetric).Methods("POST")
	router.HandleFunc("/readiness", Readiness).Methods("GET")
	router.HandleFunc("/liveness", Liveness).Methods("GET")
	log.Fatal(http.ListenAndServe(":8000", router))
}
