FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
COPY examples/go.mod examples/go.sum ./examples/
RUN cd examples && go mod download

COPY . .
RUN cd examples && CGO_ENABLED=0 go build -trimpath -o /out/workload .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/workload /workload
ENTRYPOINT ["/workload"]
