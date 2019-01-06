This is a really basic controller that watches all namespaces in Kubernetes, and spits them out to Slack.

# Configuration

## KUBECONFIG

If you specify the KUBECONFIG environment variable, it will try to use that path. Otherwise it assumes you are running in a Kubernetes cluster.

## SLACK_WEBHOOK

This must be configured in order for this to work. It's the environment variable `SLACK_WEBHOOK` set with a Slack integrations url.