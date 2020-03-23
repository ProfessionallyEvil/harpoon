for res in $(kubectl api-resources -o name);do kubectl get $res -o yaml | tee -a k8s.dump;done
