# deploys etcd into the cluster
.PHONY: deploy
deploy:
	helm install etcd -n etcd --create-namespace oci://registry-1.docker.io/bitnamicharts/etcd \
		--set replicaCount=3 \
		--set metrics.enabled=true \
		--set metrics.podMonitor.enabled=true \
		--set auth.rbac.create=false \
		--set auth.rbac.allowNoneAuthentication=true \
		--set auth.token.enabled=false

# deletes etcd from the cluster
.PHONY: delete
delete:
	helm delete -n etcd etcd
	kubectl delete ns etcd
