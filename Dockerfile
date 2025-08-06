# Defining App builder image
FROM golang:alpine AS builder

# Add git to determine build git version
RUN apk add --no-cache --update git

# Set GOPATH to build Go app
ENV GOPATH=/go

# Set apps source directory
ENV SRC_DIR=${GOPATH}/src/github.com/triapex/auth

# Define current working directory
WORKDIR ${SRC_DIR}

# Copy apps source code to the image
COPY . ${SRC_DIR}

# Build App
RUN ./build.sh

# Defining App image
FROM alpine:latest

RUN apk add --no-cache --update ca-certificates

WORKDIR /home/triapex

# Copy jwt private and public keys
COPY keys keys

# Copy App binary to image
COPY --from=builder /go/bin/auth /usr/local/bin/auth

EXPOSE 8080

ENTRYPOINT ["auth"]