# Calciotto App

## Air command

``` bash
docker/podman run -it --rm --name air -w "/app" -v ${pwd}$:/app -p 8080:8080 cosmtrek/air -c .air.toml
```

Think to create the app's container, integrate air within and plus on docker-compose.yaml