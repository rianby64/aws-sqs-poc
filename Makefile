
$(eval GO_DEVEL := $(shell echo 'go-devel'));
$(eval CONTAINER_ID_EXAMPLES := $(shell podman ps -a | grep $(GO_DEVEL) | awk '{ print $$1 }'))

devel:
ifeq ($(CONTAINER_ID_EXAMPLES),)
	podman build -f go.dockerfile -t $(GO_DEVEL) .
	podman run -it \
		--name $(GO_DEVEL) \
		-e DISPLAY \
		-v /run/dbus/system_bus_socket:/run/dbus/system_bus_socket:ro \
		-v /tmp/.X11-unix:/tmp/.X11-unix:ro \
		-v /home/rianby64/aws-sqs-poc:/home/rianby64/go/src/github.com/rianby64/aws-sqs-poc \
		--device /dev/dri \
		--device /dev/snd \
		--ipc=host \
		--userns keep-id \
		--security-opt seccomp=unconfined \
		-u ${USER} \
		$(GO_DEVEL)
else
	podman start -ia $(CONTAINER_ID_EXAMPLES)
endif
