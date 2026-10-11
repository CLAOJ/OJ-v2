# Agent + the v1 grading engine. Build the tiervnoj image first
# (CLAOJ/claoj-docker/judge/tiervnoj -> claoj/judge-tiervnoj).
ARG TIERVNOJ_IMAGE=claoj/judge-tiervnoj

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /claoj-judge .

FROM ${TIERVNOJ_IMAGE}
COPY --from=build /claoj-judge /usr/local/bin/claoj-judge
# Runs as root: the tiervnoj entry script drops to the judge user itself.
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/claoj-judge"]
