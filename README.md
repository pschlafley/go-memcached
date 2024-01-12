# go-memcached

Golang Version of a memcached Server.

go-memcached
Overview
go-memcached is a lightweight, high-performance Memcached server implementation written in Go. This project aims to provide a simple and efficient solution for caching data in memory using the Memcached protocol.

Features
Memcached Protocol Compatible: Implements the Memcached protocol, making it compatible with existing Memcached clients.
Concurrency: Utilizes Goroutines for concurrent processing, ensuring optimal performance in multi-client scenarios.
In-memory Storage: Data is stored in-memory, providing fast access times for cached items.
Expiration: Supports item expiration based on time-to-live (TTL), automatically removing outdated entries.
Configurability: Easily configurable through a YAML configuration file, allowing customization of port, cache size, and other parameters.
Getting Started
Prerequisites

Go 1.14 or later installed on your machine.

Installation:
go install github.com/pschlafley/go-memcached

Run the server:

go-memcached
Connect your Memcached clients to the specified port (default is 11211) and start caching data.
You can specify the port ther server runs on with the -p flag:
go-memcached -p PORT
