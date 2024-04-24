# Use the official Golang image based on Debian
FROM golang:1.20

# Set labels
LABEL contributors="Azakonov, Khas3r0nd && Mnazarbe"
LABEL description="forum"

# Set the working directory
WORKDIR /forum

# Install SQLite3 and GCC
RUN apt-get update && apt-get install -y sqlite3 libsqlite3-dev gcc

# Enable CGO and build the Go application
ENV CGO_ENABLED=1
COPY ./ ./
RUN go build -o main cmd/web/*

# Define the command to run your application
CMD ["./main"]
