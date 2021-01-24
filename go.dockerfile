FROM rianby64/vscode AS go

RUN yum update -y

RUN yum install -y \
    golang.x86_64

RUN useradd rianby64

USER rianby64

ENV HOME=/home/rianby64
ENV GOPATH=$HOME/go
ENV GOBIN=$GOPATH/bin

WORKDIR $HOME

RUN mkdir -p $GOPATH

RUN go get -v github.com/ramya-rao-a/go-outline \
    github.com/go-delve/delve/cmd/dlv \
    github.com/uudashr/gopkgs/cmd/gopkgs \
    golang.org/x/tools/cmd/guru \
    github.com/sqs/goreturns \
    golang.org/x/tools/cmd/gorename \
    github.com/acroca/go-symbols \
    github.com/cweill/gotests/... \
    github.com/fatih/gomodifytags \
    github.com/josharian/impl \
    github.com/davidrjenni/reftools/cmd/fillstruct \
    github.com/haya14busa/goplay/cmd/goplay \
    github.com/godoctor/godoctor \
    github.com/stamblerre/gocode \
    github.com/rogpeppe/godef \
    golang.org/x/lint/golint \
    golang.org/x/tools/gopls

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

RUN code --install-extension golang.go

ENV PATH=$PATH:$GOBIN

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.33.0

CMD ["bash"]
