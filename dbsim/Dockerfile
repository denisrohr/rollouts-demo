FROM golang:1.13 as build
WORKDIR /go/src/app
COPY . .
RUN make

FROM scratch
COPY --from=build /go/src/app/rollouts-demo-db /rollouts-demo-db

ARG COLOR
ENV COLOR=${COLOR}

ENTRYPOINT [ "/rollouts-demo-db" ]
