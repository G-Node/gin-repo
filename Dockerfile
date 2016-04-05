FROM ubuntu:16.04

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update &&                                   \
    apt-get install -y --no-install-recommends          \
                       gcc g++ libc6-dev make golang    \
                       git git-annex openssh-server     \
                       python-pip python-setuptools     \
    && rm -rf /var/lib/apt/lists/*

RUN pip install supervisor

ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH

# we need to have 755 permissions so sshd
# will accept gin-repo as AuthorizedKeysCommand
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 755 "$GOPATH"
WORKDIR $GOPATH

RUN addgroup --system git
RUN adduser --system --home /data --shell /bin/sh --ingroup git --disabled-password git
RUN passwd -d git

# speed up things by pre-go getting dependencies
RUN go get "github.com/docopt/docopt-go"
RUN go get "github.com/gorilla/mux"
RUN go get "github.com/dgrijalva/jwt-go"

# make gin-repo available in $PATH for ssh connections
RUN ln -sf $GOPATH/bin/gin-repo /usr/bin/gin-repo

# setup the ssh deamon
COPY ./contrib/ssh_host_rsa_key* /etc/ssh/
COPY ./contrib/sshd_config /etc/ssh/
RUN chmod -R 600 /etc/ssh/ssh_host_rsa_key
RUN mkdir /var/run/sshd && chmod 755 /var/run/sshd

# use supervisord to start sshd and gin-repod
COPY ./contrib/supervisord.conf /etc/supervisord.conf
EXPOSE 22 8888
CMD ["supervisord", "-c/etc/supervisord.conf"]

# To provision client keys for testing uncomment
#  the following line:
# COPY ./contrib/*.rsa* /data/

RUN mkdir -p $GOPATH/src/github.com/G-Node/gin-repo
WORKDIR $GOPATH/src/github.com/G-Node/gin-repo

COPY . $GOPATH/src/github.com/G-Node/gin-repo
RUN go get -d -v ./...
RUN go install -v ./...

RUN chown -R git:git /data
WORKDIR /data
