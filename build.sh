docker run --rm -it -v "$PWD":/usr/src/updog -w /usr/src/updog golang:1.12 bash

go get -d -v ./...

time for ARCH in 386 amd64; do
    for OS in darwin linux; do
        echo "Building build/updog-$OS-$ARCH"
        time env GOOS=$OS GOARCH=$ARCH go build -ldflags "-X main.version=abc" -o build/updog-$OS-$ARCH
    done
    echo "Building build/updog-windows-$ARCH"
    time env GOOS=windows GOARCH=$ARCH go build -ldflags "-X main.version=abc" -o build/updog-windows-$ARCH.exe
done


docker run --rm -it -v "$PWD":/usr/src/updog -w /usr/src/updog golang:1.12 ls -lh