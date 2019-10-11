
run-docker:
	docker build  -t kustomize-demo-api:testing .
	docker run -it -p 3000:3000 kustomize-demo-api:testing
