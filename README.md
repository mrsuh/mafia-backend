# MAFIA backend

Party game modelling a conflict between an informed minority, the mafia, and an uninformed majority, the innocents

## Build
```bash
sh bin/build.sh
```

## Run
```bash
./bin/server --port=9000
```

## Test
```bash
go test mafia-backend/src -v
```

## Docker

#### Build
```
docker build -p mafia-backend .
```

#### Run
```
docker run mafia-backend
```

#### Run from Hub
```
docker run mrsuh/mafia-backend
```