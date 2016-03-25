FROM golang:1.6-alpine

RUN apk add --update git openssh supervisor && rm -rf /var/cache/apk/*

RUN addgroup -S git
RUN adduser -D -S -h /data -G git -s /bin/sh git
RUN passwd -d git

COPY ./contrib/ssh_host_rsa_key* /etc/ssh/
COPY ./contrib/sshd_config /etc/ssh/

RUN chmod -R 600 /etc/ssh/ssh_host_rsa_key

COPY ./contrib/supervisord.conf /etc/supervisord.conf

# we need to have 755 permissions so sshd
# will accept gin-repo as AuthorizedKeysCommand
RUN chmod -R 755 $GOPATH

# make gin-repo available in $PATH for ssh connections
RUN ln -sf $GOPATH/bin/gin-repo /usr/bin/gin-repo

# speed up things by pre-go getting dependencies
RUN go get "github.com/docopt/docopt-go"
RUN go get "github.com/gorilla/mux"

RUN mkdir -p $GOPATH/src/github.com/G-Node/gin-repo
WORKDIR $GOPATH/src/github.com/G-Node/gin-repo

COPY . $GOPATH/src/github.com/G-Node/gin-repo
RUN go get -d -v ./...
RUN go install -v ./...

WORKDIR /data


EXPOSE 22 8888
CMD ["/usr/bin/supervisord", "-c/etc/supervisord.conf"]