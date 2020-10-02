FROM golang as builder
ARG VERSION

RUN mkdir provider
WORKDIR /go/provider

COPY . .

RUN go mod init crucible_provider

RUN mkdir -p releases/linux_amd64 &&    \
    mkdir -p releases/linux_386 &&      \
    mkdir -p releases/darwin_amd64 &&   \
    mkdir -p releases/windows_386 &&    \
    mkdir -p releases/windows_amd64

WORKDIR /go/provider/cmd

RUN GOOS=linux GOARCH=386 go build -o ../releases/linux_386/terraform-provider-crucible_$VERSION
RUN GOOS=linux GOARCH=amd64 go build -o ../releases/linux_amd64/terraform-provider-crucible_$VERSION
RUN GOOS=darwin GOARCH=amd64 go build -o ../releases/darwin_amd64/terraform-provider-crucible_$VERSION
RUN GOOS=windows GOARCH=386 go build -o ../releases/windows_386/terraform-provider-crucible_$VERSION.exe
RUN GOOS=windows GOARCH=amd64 go build -o ../releases/windows_amd64/terraform-provider-crucible_$VERSION.exe

FROM httpd as distribution
ARG VERSION
COPY --from=builder /go/provider/releases /usr/local/apache2/htdocs/

# Provide version number as main page
RUN echo "$VERSION" > /usr/local/apache2/htdocs/index.html
