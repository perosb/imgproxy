build:
	docker build -f docker/Dockerfile.archlinux -t thulecr.azurecr.io/imgproxy:latest .
push:
	docker push thulecr.azurecr.io/imgproxy:latest
run:
	docker run -ti -p 8080:8080 thulecr.azurecr.io/imgproxy:latest