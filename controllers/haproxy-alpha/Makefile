all: push

# 0.0 shouldn't clobber any release builds
TAG = test
PREFIX = index.caicloud.io/caicloud/haproxy-ingress

controller: controller.go
	CGO_ENABLED=0 GOOS=linux godep go build -a -installsuffix cgo -ldflags '-w' -o controller ./controller.go

container: controller
	docker build -t $(PREFIX):$(TAG) .

push: container
	gcloud docker push $(PREFIX):$(TAG)

clean:
	rm -f controller
