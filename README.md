# KubeEP Backend
## Tech Stack
- Go
- Postgres (with Timescale DB)
- Redis

## Prerequisites
- docker
- docker-compose
- make

## How To Run
Before you run back-end or cron application, you need to create the db by executing this command `docker-compose up -d`

### Back-end
- Run command `make run-dev`

### Cron
- Run command `make run-cron`