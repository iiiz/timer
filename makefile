main: clean build

build:
	@echo "Building"
	GOARCH=amd64 GOOS=linux go build -o timer *.go

clean:
	@echo "Cleaning up"
	@if [ -f ./timer ]; then\
		rm ./timer;\
	fi

install: clean build
	@echo "Installing"
	cp -f ./timer /usr/local/bin

