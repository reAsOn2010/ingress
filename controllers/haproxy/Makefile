all: push

# 0.0 shouldn't clobber any release builds
TAG=$(IMAGE_TAG)
NAME=$(IMAGE_NAME)

local: controller.go
	mkdir -p ./bin
	CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags '-w' -o bin/controller .

controller: controller.go
	mkdir -p ./bin
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o bin/controller .

container: controller
	docker build -t $(NAME):$(TAG) .

push: container
	docker push $(NAME):$(TAG)

clean:
	rm -rf bin/
