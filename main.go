package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Try to use KUBECONFIG if set
	// Ohterwise attempt default kubeconfig path.
	var conf *rest.Config
	var err error
	kc := os.Getenv("KUBECONFIG")
	if len(kc) > 0 {
		conf, err = clientcmd.BuildConfigFromFlags("", kc)
		if err != nil {
			log.Println("Failed to laod kubeconfig set in environment variable KUBECONFIG.")
			log.Panic(err.Error())
		}
		log.Println("Using kubeconfig in environment setting.")
	} else {
		// If KUBECONIG is not defined, look in default path.
		kc := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		conf, err = clientcmd.BuildConfigFromFlags("", kc)
		if err != nil {
			log.Println("Failed to load kubeconfig in default path.")
			log.Panic(err.Error())
		}
	}
	log.Printf("Using kubeconfig: %s\n", kc)

	// Validate Slack Webhook is configure, otherwise, what's the point.
	url := os.Getenv("SLACK_WEBHOOK")
	if len(url) == 0 {
		log.Panic("SLACK_WEBHOOK must be set to a valid Webhook URL.")
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		log.Fatal(err)
	}
	listOpts := metav1.ListOptions{}

	// Watch for changes in Namespaces
	// TODO: Try informer API
	watcher, err := clientset.CoreV1().Namespaces().Watch(listOpts)
	if err != nil {
		log.Fatal(err)
	}
	ch := watcher.ResultChan()

	for event := range ch {
		ns, ok := event.Object.(*v1.Namespace)
		if !ok {
			log.Fatalf("WTF happened while casting this object?\n%v\n", event)
		}
		switch event.Type {
		case watch.Added:
			fallthrough
		case watch.Deleted:
			sendToSlack(*ns, strings.ToLower(fmt.Sprintf("%v", event.Type)))
		case watch.Modified:
			// Log but don't notify.
			log.Printf("Modified: %s\n", ns.Name)
		case watch.Error:
			log.Printf("Help me, I watch.Error: %v\n", ns)
		}
	}
}

func sendToSlack(ns v1.Namespace, status string) {
	var url = os.Getenv("SLACK_WEBHOOK")
	var body = []byte(fmt.Sprintf("{\"text\":\"namespace %s %s\"}", ns.Name, status))
	var client = &http.Client{
		Timeout: time.Second * 15,
	}

	res, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Fatalf("Failed to send message to slack.\n%v\n", err)
		return
	}
	log.Printf("Response from Slack:\n%s\n", res.Body)
}
