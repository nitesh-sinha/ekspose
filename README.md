## Run the following cmds to create an nginx deployment:
kubectl create ns eksposetest
kubectl create deployment nginx-dep -n eksposetest --image nginx


## This should trigger our controller to create the corresponding service for the deployment.
watch kubectl get services -n eksposetest


## Assuming we're at a point where the ingress is not yet created(bcoz its not coded in controller yet - phase 1), we can still port forward our localhost port 8080 to nginx service port 80
kubectl port-forward -n eksposetest svc/nginx-dep 8080:80

Now access the nginx app using http://localhost:8080 on browser