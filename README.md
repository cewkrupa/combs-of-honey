# Combs Of Honey

Example app used for onboarding to OpenTelemetry + Honeycomb for the first time. This is rough on purpose -- I'm no go expert!

- https://github.com/gin-gonic/gin for routing / response handling
- https://gorm.io/ for ORM
- SQLite for the database
- OpenTelemetry for tracing
- Postman collection 

## Requirements

This is what I built it with -- no idea if other versions will work or not. YMMV

| requirement | version|
| --- | --- |
| golang | 1.18.3 |
| sqlite3 | 3.37.0 |
| MacOS | 12.5, Apple M1 Pro |


## Setup
1. Make sure you have sqlite3 installed
2. Make sure you have go 1.18.3 installed (I used [asdf](https://asdf-vm.com/) to handle this)
3. Run the app with `OTEL_EXPORTER_OTLP_ENDPOINT="grpc://api.honeycomb.io:443" OTEL_EXPORTER_OTLP_HEADERS="x-honeycomb-team=YOUR_API_KEY" OTEL_SERVICE_NAME="combs-of-honey" go run main.go`

## Features

Combs of Honey is a simple REST api that has "Combs" and "Honey". "Combs" can be created and retrieved, "Honey" can be added to a Comb, retrieved, and deleted.

There is an example Postman collection in `combs-of-honey.postman_collection.json`. 