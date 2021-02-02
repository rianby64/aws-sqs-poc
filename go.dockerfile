FROM rianby64/vscode AS go

RUN yum update -y

RUN yum install -y \
    golang.x86_64

RUN useradd rianby64

USER rianby64

ENV HOME=/home/rianby64
ENV GOPATH=$HOME/go
ENV GOBIN=$GOPATH/bin

RUN mkdir -p $GOPATH

RUN go get -v github.com/ramya-rao-a/go-outline
RUN go get -v github.com/go-delve/delve/cmd/dlv
RUN go get -v github.com/uudashr/gopkgs/v2/cmd/gopkgs
RUN go get -v golang.org/x/tools/cmd/guru
RUN go get -v github.com/sqs/goreturns
RUN go get -v golang.org/x/tools/cmd/gorename
RUN go get -v github.com/acroca/go-symbols
RUN go get -v github.com/cweill/gotests/...
RUN go get -v github.com/fatih/gomodifytags
RUN go get -v github.com/josharian/impl
RUN go get -v github.com/davidrjenni/reftools/cmd/fillstruct
RUN go get -v github.com/haya14busa/goplay/cmd/goplay
RUN go get -v github.com/godoctor/godoctor
RUN go get -v github.com/stamblerre/gocode
RUN go get -v github.com/rogpeppe/godef
RUN go get -v golang.org/x/lint/golint
RUN go get -v github.com/stretchr/testify

RUN code --install-extension golang.Go

ENV PATH=$PATH:$GOBIN

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.33.0

#RUN GO111MODULE=on go get -v golang.org/x/tools/gopls

WORKDIR $GOPATH/src/github.com/rianby64/sqs-retry

CMD ["bash"]
