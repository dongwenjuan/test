FROM golang:1.13-alpine3.10 AS builder

WORKDIR /go/src
RUN mkdir -p /go/src/knative.dev/serving
ADD ./ /go/src/knative.dev/serving/
RUN go build -o /go/bin/autoscaler /go/src/knative.dev/serving/cmd/autoscaler/


#FROM golang:1.13-alpine
FROM alpine:3.10

# Create a group and user
RUN addgroup -g 1030 cf-vnpm && adduser -u 1030  -D --home /home/cf-vnpm -s /sbin/nologin -G cf-vnpm cf-vnpm && chown -R 1030:1030 /home/cf-vnpm

RUN touch /var/log/output.log && chown 1030:1030 /var/log/output.log && chmod 744 /var/log/output.log

# Set the Current Working Directory inside the container
WORKDIR /home/cf-vnpm

# Copy the source from the current directory to the Working Directory inside the container
COPY --from=builder --chown=1030:1030 /go/bin/autoscaler /home/cf-vnpm/

USER 1030:1030

# Expose port 8080 to the outside world
EXPOSE 8080 8008 9090

# Command to run the executable
CMD ["/home/cf-vnpm/autoscaler"]