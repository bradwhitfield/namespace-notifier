package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var logger, _ = zap.NewProduction()
var client = &http.Client{
	Timeout: time.Second * 15,
}
var url = os.Getenv("SLACK_WEBHOOK")

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	var conf *rest.Config
	var err error

	// This will attempt to load the config in KUBECONFIG envvar, and default to InClusterConfig otherwise
	kc := os.Getenv("KUBECONFIG")
	conf, err = clientcmd.BuildConfigFromFlags("", kc)
	if err != nil {
		logger.Error("Failed to load kubeconfig.",
			zap.Error(err),
		)
	}

	// Validate Slack Webhook is configure, otherwise, what's the point.
	url := os.Getenv("SLACK_WEBHOOK")
	if len(url) == 0 {
		logger.Panic("SLACK_WEBHOOK must be set to a valid Webhook URL.")
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		logger.Panic("Failed creating clientset",
			zap.Error(err),
		)
	}
	listOpts := metav1.ListOptions{}

	// Watch for changes in Namespaces
	// TODO: Try informer API
	watcher, err := clientset.CoreV1().Namespaces().Watch(listOpts)
	if err != nil {
		logger.Panic("Failed to create watcher.",
			zap.Error(err),
		)
	}
	ch := watcher.ResultChan()

	for event := range ch {
		ns, ok := event.Object.(*v1.Namespace)
		if !ok {
			logger.Panic("WTF happened while casting this object?",
				zap.Reflect("event", event),
			)
		}
		switch event.Type {
		case watch.Added:
			fallthrough
		case watch.Deleted:
			sendToSlack(*ns, strings.ToLower(fmt.Sprintf("%v", event.Type)))
		case watch.Modified:
			// Log but don't notify.
			logger.Info("Modified NS.",
				zap.String("name", ns.Name),
			)
		case watch.Error:
			logger.Error("Help me, watch.Error:",
				zap.Reflect("event", "event"),
			)
		}
	}
}

func sendToSlack(ns v1.Namespace, status string) {
	var body = []byte(fmt.Sprintf("{\"text\":\"namespace %s %s\"}", ns.Name, status))

	res, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}
	if err != nil {
		logger.Error("Failed to send message to slack.",
			zap.Error(err),
		)
		return
	}
	logger.Info("Slack did some stuff.",
		zap.Reflect("body", res.Body),
	)
}
