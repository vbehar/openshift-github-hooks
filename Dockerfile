FROM golang:1.6

MAINTAINER https://github.com/vbehar/openshift-github-hooks

ENV GOPATH=/go/src/github.com/vbehar/openshift-github-hooks/Godeps/_workspace:/go

COPY . /go/src/github.com/vbehar/openshift-github-hooks/

RUN go install github.com/vbehar/openshift-github-hooks

RUN mv /go/bin/openshift-github-hooks /openshift-github-hooks

WORKDIR "/"

CMD [ "/openshift-github-hooks" ]
