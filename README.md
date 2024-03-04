# noderesourcetopology
report node resource topology

# build binary
`make binary`

# build image
`make image`

# deploy
`kubectl apply -f ./deployments`

```
# kubectl get pod -A -o wide
NAMESPACE     NAME                                         READY   STATUS    RESTARTS         AGE     IP               NODE               NOMINATED NODE   READIN
kube-system   topology-updater-58wx2                       1/1     Running   0                14h     172.20.84.137    k8s-work2          <none>           <none>
kube-system   topology-updater-g547p                       1/1     Running   0                14h     172.20.84.96     ruike-k8s-master   <none>           <none>
kube-system   topology-updater-x9s8h                       1/1     Running   0                13h     172.20.182.224   k8s-work1          <none>           <none>

# kubectl get nrt
NAME               AGE
k8s-work1          13h
k8s-work2          8m52s
ruike-k8s-master   7m57s
```